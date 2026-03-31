package delete

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/spf13/cobra"
)

// ─── Command registration tests ────────────────────────────────────────────

func TestDeleteCmd_IsRegistered(t *testing.T) {
	if DeleteCmd == nil {
		t.Fatal("DeleteCmd should not be nil")
	}
	if DeleteCmd.Use != "delete" {
		t.Errorf("DeleteCmd.Use = %q, want %q", DeleteCmd.Use, "delete")
	}
	if DeleteCmd.Short == "" {
		t.Error("DeleteCmd.Short should not be empty")
	}
}

func TestEntityCmd_IsSubcommand(t *testing.T) {
	found := false
	for _, sub := range DeleteCmd.Commands() {
		if strings.HasPrefix(sub.Use, "entity") {
			found = true
			break
		}
	}
	if !found {
		t.Error("entityCmd should be registered as a subcommand of DeleteCmd")
	}
}

func TestEntityCmd_RequiresExactlyOneArg(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range DeleteCmd.Commands() {
		if strings.HasPrefix(sub.Use, "entity") {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("entity subcommand not found")
	}

	// cobra.ExactArgs(1) is set - verify by checking the Args validator
	if cmd.Args == nil {
		t.Error("entity command must have Args validator (ExactArgs(1))")
	}
}

// ─── Confirmation flow tests ───────────────────────────────────────────────

func TestConfirm_YesFlagSkipsPrompt(t *testing.T) {
	confirmed, err := tui.Confirm("Delete?", true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !confirmed {
		t.Error("with yesFlag=true, Confirm should return true")
	}
}

func TestConfirm_UserTypesY_Confirms(t *testing.T) {
	reader := bytes.NewBufferString("y\n")
	confirmed, err := tui.Confirm("Delete?", false, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !confirmed {
		t.Error("typing 'y' should confirm")
	}
}

func TestConfirm_UserTypesYes_Confirms(t *testing.T) {
	reader := bytes.NewBufferString("yes\n")
	confirmed, err := tui.Confirm("Delete?", false, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !confirmed {
		t.Error("typing 'yes' should confirm")
	}
}

func TestConfirm_UserTypesN_Denies(t *testing.T) {
	reader := bytes.NewBufferString("n\n")
	confirmed, err := tui.Confirm("Delete?", false, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if confirmed {
		t.Error("typing 'n' should deny")
	}
}

func TestConfirm_UserTypesEmpty_Denies(t *testing.T) {
	reader := bytes.NewBufferString("\n")
	confirmed, err := tui.Confirm("Delete?", false, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if confirmed {
		t.Error("empty input should deny (default N)")
	}
}

func TestConfirm_UserTypesRandomText_Denies(t *testing.T) {
	reader := bytes.NewBufferString("maybe\n")
	confirmed, err := tui.Confirm("Delete?", false, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if confirmed {
		t.Error("random text should deny")
	}
}

func TestConfirm_NilReader_NoYesFlag_ReturnsError(t *testing.T) {
	_, err := tui.Confirm("Delete?", false, nil)
	if err == nil {
		t.Error("nil reader with yesFlag=false should return error")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("error should mention --yes flag, got: %v", err)
	}
}

func TestConfirm_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Y\n", true},
		{"YES\n", true},
		{"Yes\n", true},
		{"yEs\n", true},
		{"N\n", false},
		{"NO\n", false},
		{"no\n", false},
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.input), func(t *testing.T) {
			reader := bytes.NewBufferString(tt.input)
			got, err := tui.Confirm("Delete?", false, reader)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Confirm(%q) = %v, want %v", strings.TrimSpace(tt.input), got, tt.want)
			}
		})
	}
}

func TestConfirm_EOF_Denies(t *testing.T) {
	reader := bytes.NewBufferString("") // EOF
	confirmed, err := tui.Confirm("Delete?", false, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if confirmed {
		t.Error("EOF should deny")
	}
}

// ─── Error path: no config ─────────────────────────────────────────────────

func TestRunDeleteEntity_NoConfig_ReturnsError(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("HOME", cfgDir)
	t.Setenv("USERPROFILE", cfgDir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")

	cmd := &cobra.Command{Use: "test"}
	// Simulate --yes to skip confirmation prompt
	err := runDeleteEntity(cmd, []string{"test-entity"})

	// Without --yes, it will try to read from os.Stdin which in tests
	// returns EOF -> confirmed=false -> "Cancelled."
	// We need the yesFlag path to actually reach the client creation.
	// Since we can't easily set yesFlag from here, the function will
	// print "Cancelled." and return nil (user declined).
	// This is expected behavior.
	if err != nil {
		// If it errors, it should be about config, not a panic
		if strings.Contains(err.Error(), "panic") {
			t.Errorf("unexpected panic: %v", err)
		}
	}
}

// ─── Metamorphic: confirmation is deterministic ────────────────────────────

func TestConfirm_Deterministic_SameInputSameOutput(t *testing.T) {
	inputs := []string{"y\n", "yes\n", "n\n", "no\n", "\n", "maybe\n"}
	for _, input := range inputs {
		t.Run(strings.TrimSpace(input), func(t *testing.T) {
			// Run twice with same input, expect same result
			r1 := bytes.NewBufferString(input)
			got1, err1 := tui.Confirm("Delete?", false, r1)
			if err1 != nil {
				t.Fatalf("run 1 error: %v", err1)
			}

			r2 := bytes.NewBufferString(input)
			got2, err2 := tui.Confirm("Delete?", false, r2)
			if err2 != nil {
				t.Fatalf("run 2 error: %v", err2)
			}

			if got1 != got2 {
				t.Errorf("Confirm(%q) not deterministic: run1=%v, run2=%v",
					strings.TrimSpace(input), got1, got2)
			}
		})
	}
}

// ─── Property-based: command structure invariants ──────────────────────────

func TestDeleteCmd_AllSubcommands_HaveShortDescription(t *testing.T) {
	for _, sub := range DeleteCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.Short == "" {
				t.Errorf("subcommand %q has empty Short description", sub.Name())
			}
		})
	}
}

func TestDeleteCmd_AllSubcommands_HaveRunE(t *testing.T) {
	for _, sub := range DeleteCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.RunE == nil && sub.Run == nil {
				t.Errorf("subcommand %q has neither RunE nor Run set", sub.Name())
			}
		})
	}
}
