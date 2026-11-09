package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/addon"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/spf13/cobra"
)

var addonPublishDir string

var addonPublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish addon to the marketplace",
	Long: `Publish an addon to your Shoehorn instance's marketplace.

Reads manifest.json and uploads it along with any built bundles
(dist/addon.js, dist/frontend.js).

Examples:
  # Publish from the current directory
  shoehorn addon publish

  # Publish from a specific directory
  shoehorn addon publish --dir ./addons/jira-sync`,
	RunE: runAddonPublish,
}

func runAddonPublish(cmd *cobra.Command, _ []string) error {
	dir := addonPublishDir
	if dir == "" {
		dir = "."
	}

	// Read manifest.json with size check (S7)
	manifestPath := filepath.Join(dir, "manifest.json")
	mInfo, err := os.Stat(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no manifest.json found in %s", dir)
		}
		return fmt.Errorf("stat manifest.json: %w", err)
	}
	if mInfo.Size() > addon.MaxBundleSize {
		return fmt.Errorf("manifest.json exceeds maximum size of %d bytes (2MB)", addon.MaxBundleSize)
	}
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest.json: %w", err)
	}

	var manifest map[string]any
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("invalid manifest.json: %w", err)
	}

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	// Step 1: Publish manifest
	result, spinErr := tui.RunSpinner("Publishing manifest...", func() (any, error) {
		return client.PublishAddonManifest(cmd.Context(), manifest)
	})
	if spinErr != nil {
		return fmt.Errorf("publish addon: %w", spinErr)
	}

	pub := result.(*api.PublishResult)

	action := "updated"
	if pub.Created {
		action = "published"
	}

	fmt.Printf("Addon %q %s successfully.\n", pub.Slug, action)
	fmt.Printf("  Name: %s\n", pub.Name)
	if pub.Installed {
		fmt.Println("  Auto-installed for your tenant.")
	}

	// Step 2: Upload bundles if they exist (check size before reading - S7)
	bundles := map[string][]byte{}

	for _, bundle := range []struct{ name, file string }{
		{"backend", filepath.Join(dir, "dist", "addon.js")},
		{"frontend", filepath.Join(dir, "dist", "frontend.js")},
	} {
		info, err := os.Stat(bundle.file)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat bundle %s: %w", bundle.name, err)
		}
		if info.Size() > addon.MaxBundleSize {
			return fmt.Errorf("bundle %s (%s) exceeds maximum size of %d bytes (2MB)",
				bundle.name, bundle.file, addon.MaxBundleSize)
		}
		data, err := os.ReadFile(bundle.file)
		if err != nil {
			return fmt.Errorf("read bundle %s: %w", bundle.name, err)
		}
		bundles[bundle.name] = data
	}

	if len(bundles) > 0 {
		uploadResult, uploadErr := tui.RunSpinner("Uploading bundles...", func() (any, error) {
			return client.UploadAddonBundle(cmd.Context(), pub.Slug, bundles)
		})
		if uploadErr != nil {
			return fmt.Errorf("upload bundles: %w", uploadErr)
		}

		upload := uploadResult.(*api.BundleUploadResult)
		for name, size := range upload.Uploaded {
			fmt.Printf("  Bundle %s: %d bytes uploaded\n", name, size)
		}

		// On a first-ever publish, the server auto-installs the addon as
		// part of step 1 above -- before any bundle exists, since bundles
		// only arrive here in step 2. That install's attempt to load the
		// addon into the runtime always fails for exactly this reason
		// (confirmed live: "no backend bundle uploaded"), and the server
		// swallows that failure rather than rejecting the install, leaving
		// a row that *looks* installed but was never actually loaded.
		// Enable is safe to call unconditionally here (idempotent, retries
		// the load with the bundle now in place) and is cheap compared to
		// the alternative of asking the user to disable/re-enable by hand.
		if pub.Created {
			if _, enableErr := tui.RunSpinner("Loading addon into the runtime...", func() (any, error) {
				return nil, client.EnableAddon(cmd.Context(), pub.Slug)
			}); enableErr != nil {
				return fmt.Errorf("load addon after bundle upload: %w", enableErr)
			}
		}
	}

	return nil
}

func init() {
	addonPublishCmd.Flags().StringVarP(&addonPublishDir, "dir", "d", "", "addon project directory (default: current directory)")
	addonCmd.AddCommand(addonPublishCmd)
}
