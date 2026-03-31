package commands

import (
	"fmt"
	"os"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	checkMinScore int
	checkHasOwner bool
	checkHasDocs  bool
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run checks for CI/CD pipelines",
	Long:  `Run validation checks on catalog entities. Exit code 0 = pass, 1 = fail.`,
}

var checkScorecardCmd = &cobra.Command{
	Use:   "scorecard <entity-id>",
	Short: "Check entity scorecard meets minimum score",
	Long: `Verify an entity's scorecard score meets the minimum threshold.
Exit code 0 if score >= threshold, 1 if below.

Examples:
  shoehorn check scorecard my-service --min-score 70
  shoehorn check scorecard my-service --min-score 80 -o json`,
	Args: cobra.ExactArgs(1),
	RunE: runCheckScorecard,
}

var checkEntityCmd = &cobra.Command{
	Use:   "entity <entity-id>",
	Short: "Check entity meets quality requirements",
	Long: `Verify an entity meets basic quality requirements.
Exit code 0 if all checks pass, 1 if any fail.

Examples:
  shoehorn check entity my-service --has-owner
  shoehorn check entity my-service --has-owner --has-docs`,
	Args: cobra.ExactArgs(1),
	RunE: runCheckEntity,
}

func init() {
	checkScorecardCmd.Flags().IntVar(&checkMinScore, "min-score", 0, "minimum required scorecard score (0-100)")
	checkScorecardCmd.MarkFlagRequired("min-score")

	checkEntityCmd.Flags().BoolVar(&checkHasOwner, "has-owner", false, "require entity has an owner")
	checkEntityCmd.Flags().BoolVar(&checkHasDocs, "has-docs", false, "require entity has documentation links")

	checkCmd.AddCommand(checkScorecardCmd)
	checkCmd.AddCommand(checkEntityCmd)
	rootCmd.AddCommand(checkCmd)
}

func runCheckScorecard(cmd *cobra.Command, args []string) error {
	id := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	sc, err := client.GetEntityScorecard(ctx, id)
	if err != nil {
		return fmt.Errorf("get scorecard: %w", err)
	}

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	pass := sc.Score >= checkMinScore

	result := map[string]any{
		"entity":    id,
		"score":     sc.Score,
		"grade":     sc.Grade,
		"max_score": sc.MaxScore,
		"min_score": checkMinScore,
		"pass":      pass,
	}

	if mode == ui.ModeJSON || mode == ui.ModeYAML {
		var renderErr error
		if mode == ui.ModeJSON {
			renderErr = ui.RenderJSON(result)
		} else {
			renderErr = ui.RenderYAML(result)
		}
		if renderErr != nil {
			return renderErr
		}
		if !pass {
			return fmt.Errorf("scorecard score %d below minimum %d", sc.Score, checkMinScore)
		}
		return nil
	}

	if pass {
		fmt.Printf("PASS: %s scorecard %d/%d (min: %d)\n", id, sc.Score, sc.MaxScore, checkMinScore)
	} else {
		fmt.Fprintf(os.Stderr, "FAIL: %s scorecard %d/%d (min: %d)\n", id, sc.Score, sc.MaxScore, checkMinScore)
		return fmt.Errorf("scorecard score %d below minimum %d", sc.Score, checkMinScore)
	}
	return nil
}

func runCheckEntity(cmd *cobra.Command, args []string) error {
	id := args[0]

	if !checkHasOwner && !checkHasDocs {
		return fmt.Errorf("at least one check flag is required (--has-owner, --has-docs)")
	}

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	entity, err := client.GetEntity(ctx, id)
	if err != nil {
		return fmt.Errorf("get entity: %w", err)
	}

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)

	type checkResult struct {
		Check  string `json:"check"`
		Pass   bool   `json:"pass"`
		Detail string `json:"detail,omitempty"`
	}

	var checks []checkResult
	allPass := true

	if checkHasOwner {
		pass := entity.Owner != ""
		detail := ""
		if pass {
			detail = entity.Owner
		}
		checks = append(checks, checkResult{Check: "has-owner", Pass: pass, Detail: detail})
		if !pass {
			allPass = false
		}
	}

	if checkHasDocs {
		pass := len(entity.Links) > 0
		detail := ""
		if pass {
			detail = fmt.Sprintf("%d links", len(entity.Links))
		}
		checks = append(checks, checkResult{Check: "has-docs", Pass: pass, Detail: detail})
		if !pass {
			allPass = false
		}
	}

	result := map[string]any{
		"entity": id,
		"pass":   allPass,
		"checks": checks,
	}

	if mode == ui.ModeJSON || mode == ui.ModeYAML {
		var renderErr error
		if mode == ui.ModeJSON {
			renderErr = ui.RenderJSON(result)
		} else {
			renderErr = ui.RenderYAML(result)
		}
		if renderErr != nil {
			return renderErr
		}
		if !allPass {
			return fmt.Errorf("entity %s failed quality checks", id)
		}
		return nil
	}

	for _, c := range checks {
		if c.Pass {
			fmt.Printf("PASS: %s %s", id, c.Check)
			if c.Detail != "" {
				fmt.Printf(" (%s)", c.Detail)
			}
			fmt.Println()
		} else {
			fmt.Fprintf(os.Stderr, "FAIL: %s %s\n", id, c.Check)
		}
	}

	if !allPass {
		return fmt.Errorf("entity %s failed quality checks", id)
	}
	return nil
}
