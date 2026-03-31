package tui

import (
	"strings"
	"testing"
)

// TestConfirm_YesFlag_SkipsPrompt verifies --yes bypasses interactive prompt.
func TestConfirm_YesFlag_SkipsPrompt(t *testing.T) {
	got, err := Confirm("Delete entity?", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Confirm with yesFlag=true should return true")
	}
}

// TestConfirm_InteractiveYes_Proceeds verifies "y" input is accepted.
func TestConfirm_InteractiveYes_Proceeds(t *testing.T) {
	r := strings.NewReader("y\n")
	got, err := Confirm("Delete entity?", false, r)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Confirm with 'y' input should return true")
	}
}

// TestConfirm_InteractiveYES_CaseInsensitive verifies case insensitivity.
func TestConfirm_InteractiveYES_CaseInsensitive(t *testing.T) {
	r := strings.NewReader("YES\n")
	got, err := Confirm("Delete entity?", false, r)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Confirm with 'YES' input should return true")
	}
}

// TestConfirm_InteractiveNo_Aborts verifies "n" input is rejected.
func TestConfirm_InteractiveNo_Aborts(t *testing.T) {
	r := strings.NewReader("n\n")
	got, err := Confirm("Delete entity?", false, r)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Confirm with 'n' input should return false")
	}
}

// TestConfirm_InteractiveEmpty_Aborts verifies empty input defaults to no.
func TestConfirm_InteractiveEmpty_Aborts(t *testing.T) {
	r := strings.NewReader("\n")
	got, err := Confirm("Delete entity?", false, r)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Confirm with empty input should return false (default no)")
	}
}

// TestConfirm_NilReader_NonInteractive_ReturnsError verifies non-interactive
// mode without --yes flag returns an error telling user to pass --yes.
func TestConfirm_NilReader_NonInteractive_ReturnsError(t *testing.T) {
	_, err := Confirm("Delete entity?", false, nil)
	if err == nil {
		t.Fatal("expected error when reader is nil and yesFlag is false")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("error should mention --yes flag, got: %v", err)
	}
}

// TestConfirm_EOF_ReturnsNo verifies that EOF without a newline defaults to no.
func TestConfirm_EOF_ReturnsNo(t *testing.T) {
	r := strings.NewReader("") // EOF immediately, no newline
	got, err := Confirm("Continue?", false, r)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Confirm with EOF should return false")
	}
}

// TestConfirm_InteractiveYesLowercase verifies all affirmative inputs are accepted.
func TestConfirm_InteractiveYesLowercase(t *testing.T) {
	for _, input := range []string{"y\n", "yes\n", "Y\n", "Yes\n", "YES\n"} {
		r := strings.NewReader(input)
		got, err := Confirm("Continue?", false, r)
		if err != nil {
			t.Fatalf("input %q: %v", input, err)
		}
		if !got {
			t.Errorf("input %q should confirm", input)
		}
	}
}

// TestConfirm_InteractiveNoVariants verifies rejection variants.
func TestConfirm_InteractiveNoVariants(t *testing.T) {
	for _, input := range []string{"n\n", "no\n", "N\n", "No\n", "NO\n", "nope\n", "x\n"} {
		r := strings.NewReader(input)
		got, err := Confirm("Continue?", false, r)
		if err != nil {
			t.Fatalf("input %q: %v", input, err)
		}
		if got {
			t.Errorf("input %q should NOT confirm", input)
		}
	}
}
