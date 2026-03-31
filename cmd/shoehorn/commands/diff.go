package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/manifest"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var diffFile string

var diffCmd = &cobra.Command{
	Use:   "diff -f <file|directory>",
	Short: "Show what apply would change",
	Long: `Compare local manifest files against remote state and show differences.
No changes are made -- this is a read-only preview of what "shoehorn apply" would do.

Examples:
  shoehorn diff -f service.yaml
  shoehorn diff -f catalog/
  shoehorn diff -f service.yaml -o json`,
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().StringVarP(&diffFile, "file", "f", "", "manifest file or directory")
	diffCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(diffCmd)
}

// diffEntry describes one resource's diff result.
type diffEntry struct {
	ServiceID string   `json:"service_id"`
	Name      string   `json:"name"`
	Action    string   `json:"action"` // "create", "update", "unchanged"
	Changes   []string `json:"changes,omitempty"`
}

func runDiff(cmd *cobra.Command, _ []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	mode := ui.DetectMode(interactive, noInteractive, outputFormat)

	// Collect and parse manifests (reuse apply's file collection logic)
	resources, err := collectResources(diffFile)
	if err != nil {
		return err
	}

	if len(resources) == 0 {
		fmt.Println("No resources found in manifest(s)")
		return nil
	}

	// Compare each resource against remote state
	var entries []diffEntry
	for _, r := range resources {
		remote, getErr := client.GetEntity(ctx, r.ServiceID)
		if getErr != nil && !api.IsNotFound(getErr) {
			if api.IsNotAuthenticated(getErr) {
				return fmt.Errorf("authentication failed: %w", getErr)
			}
			return fmt.Errorf("check %s: %w", r.ServiceID, getErr)
		}
		exists := getErr == nil

		if !exists {
			entries = append(entries, diffEntry{
				ServiceID: r.ServiceID,
				Name:      r.Name,
				Action:    "create",
			})
			continue
		}

		// Compare key fields
		var changes []string
		if r.Name != "" && r.Name != remote.Name {
			changes = append(changes, fmt.Sprintf("name: %q -> %q", remote.Name, r.Name))
		}
		if r.Type != "" && r.Type != remote.Type {
			changes = append(changes, fmt.Sprintf("type: %q -> %q", remote.Type, r.Type))
		}
		if r.Description != "" && r.Description != remote.Description {
			changes = append(changes, "description changed")
		}
		if len(r.Tags) > 0 {
			remoteTags := strings.Join(remote.Tags, ",")
			localTags := strings.Join(r.Tags, ",")
			if remoteTags != localTags {
				changes = append(changes, "tags changed")
			}
		}

		action := "unchanged"
		if len(changes) > 0 {
			action = "update"
		}
		entries = append(entries, diffEntry{
			ServiceID: r.ServiceID,
			Name:      r.Name,
			Action:    action,
			Changes:   changes,
		})
	}

	// Output
	if mode == ui.ModeJSON {
		return ui.RenderJSON(entries)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(entries)
	}

	// Plain text output
	var creates, updates, unchanged int
	for _, e := range entries {
		switch e.Action {
		case "create":
			creates++
			fmt.Fprintf(os.Stdout, "  + %s  %s (%s)\n", e.ServiceID, e.Name, e.Action)
		case "update":
			updates++
			fmt.Fprintf(os.Stdout, "  ~ %s  %s (%s)\n", e.ServiceID, e.Name, e.Action)
			for _, c := range e.Changes {
				fmt.Fprintf(os.Stdout, "      %s\n", c)
			}
		default:
			unchanged++
			fmt.Fprintf(os.Stdout, "  = %s  (unchanged)\n", e.ServiceID)
		}
	}

	fmt.Printf("\n%d to create, %d to update, %d unchanged\n", creates, updates, unchanged)
	return nil
}

// collectResources reads and parses manifest files from a path (file, directory, or stdin).
func collectResources(path string) ([]manifest.Resource, error) {
	if path == "-" {
		content, err := manifest.ReadFile("-")
		if err != nil {
			return nil, err
		}
		return manifest.Parse(strings.NewReader(content))
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	var files []string
	if info.IsDir() {
		err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && (strings.HasSuffix(p, ".yaml") || strings.HasSuffix(p, ".yml")) {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk directory: %w", err)
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("no .yaml or .yml files found in %s", path)
		}
	} else {
		files = []string{path}
	}

	var resources []manifest.Resource
	for _, f := range files {
		content, err := manifest.ReadFile(f)
		if err != nil {
			return nil, err
		}
		parsed, err := manifest.Parse(strings.NewReader(content))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		resources = append(resources, parsed...)
	}
	return resources, nil
}
