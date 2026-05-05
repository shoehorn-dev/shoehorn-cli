package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const maxMoldManifestSize = 1 << 20 // 1 MiB — matches the API's MaxBytesReader

// moldsRegisterCmd reads a YAML manifest and POSTs it to /api/v1/forge/molds.
var moldsRegisterCmd = &cobra.Command{
	Use:   "register <file>",
	Short: "Register a mold from a YAML manifest",
	Long: `Register a new mold by uploading a YAML manifest. Pass "-" to read from stdin.

The manifest mirrors the API payload — top-level keys: slug, name, version,
visibility, category, schema, actions (and optional description, tags, icon,
defaults, ownerTeamIds).

Example:
  shoehorn forge molds register ./mold.yaml
  cat mold.yaml | shoehorn forge molds register -`,
	Args: cobra.ExactArgs(1),
	RunE: runMoldsRegister,
}

func init() {
	moldsCmd.AddCommand(moldsRegisterCmd)
}

func runMoldsRegister(cmd *cobra.Command, args []string) error {
	path := args[0]

	data, err := readMoldManifest(path)
	if err != nil {
		return err
	}

	req, err := parseMoldManifest(data)
	if err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner(fmt.Sprintf("Registering mold %q...", req.Slug), func() (any, error) {
		return client.CreateMold(cmd.Context(), req)
	})
	if spinErr != nil {
		return fmt.Errorf("register mold: %w", spinErr)
	}

	mold := result.(*api.MoldDetail)

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	if mode == ui.ModeJSON {
		return ui.RenderJSON(mold)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(mold)
	}

	body := fmt.Sprintf(
		"%s  %s\n%s  %s\n%s  %s",
		tui.LabelStyle.Render("Slug"), mold.Slug,
		tui.LabelStyle.Render("Name"), mold.Name,
		tui.LabelStyle.Render("Version"), mold.Version,
	)
	fmt.Println(tui.SuccessBox("Mold Registered", body))
	return nil
}

// readMoldManifest reads bytes from a file path or stdin ("-"), capped at 1 MiB.
func readMoldManifest(path string) ([]byte, error) {
	if path == "-" {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, maxMoldManifestSize+1))
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		if int64(len(data)) > maxMoldManifestSize {
			return nil, fmt.Errorf("stdin exceeds maximum manifest size (%d bytes)", maxMoldManifestSize)
		}
		return data, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if info.Size() > maxMoldManifestSize {
		return nil, fmt.Errorf("%s exceeds maximum manifest size (%d bytes)", path, maxMoldManifestSize)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}

// parseMoldManifest decodes a YAML manifest into a CreateMoldRequest and runs
// minimal client-side validation so users see errors locally instead of as
// 400s from the server. The full validation contract still lives in the API.
func parseMoldManifest(data []byte) (api.CreateMoldRequest, error) {
	var req api.CreateMoldRequest
	if len(data) == 0 {
		return req, errors.New("manifest is empty")
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&req); err != nil {
		return req, err
	}

	switch {
	case req.Slug == "":
		return req, errors.New("slug is required")
	case req.Name == "":
		return req, errors.New("name is required")
	case req.Version == "":
		return req, errors.New("version is required")
	case req.Category == "":
		return req, errors.New("category is required")
	case req.Visibility == "":
		return req, errors.New("visibility is required (private | tenant | public)")
	case len(req.Schema) == 0:
		return req, errors.New("schema is required")
	case len(req.Actions) == 0:
		return req, errors.New("at least one action is required")
	}
	return req, nil
}
