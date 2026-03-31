package ui

import (
	"testing"
)

// TestDetectMode_ExplicitJSON tests that --output json always wins.
func TestDetectMode_ExplicitJSON(t *testing.T) {
	got := DetectMode(true, true, "json")
	if got != ModeJSON {
		t.Errorf("DetectMode(true, true, json) = %q, want %q", got, ModeJSON)
	}
}

// TestDetectMode_ExplicitYAML tests that --output yaml always wins.
func TestDetectMode_ExplicitYAML(t *testing.T) {
	got := DetectMode(true, true, "yaml")
	if got != ModeYAML {
		t.Errorf("DetectMode(true, true, yaml) = %q, want %q", got, ModeYAML)
	}
}

// TestDetectMode_NoInteractiveForcesPain tests --no-interactive flag.
func TestDetectMode_NoInteractiveForcesPlain(t *testing.T) {
	got := DetectMode(false, true, "")
	if got != ModePlain {
		t.Errorf("DetectMode(false, true, '') = %q, want %q", got, ModePlain)
	}
}

// TestDetectMode_DefaultIsPlain tests default behavior.
func TestDetectMode_DefaultIsPlain(t *testing.T) {
	got := DetectMode(false, false, "")
	if got != ModePlain {
		t.Errorf("DetectMode(false, false, '') = %q, want %q", got, ModePlain)
	}
}

// TestDetectMode_FormatPrecedence tests that format flag beats all other flags.
func TestDetectMode_FormatPrecedence(t *testing.T) {
	tests := []struct {
		name        string
		interactive bool
		noInteract  bool
		format      string
		want        OutputMode
	}{
		{"json beats interactive", true, false, "json", ModeJSON},
		{"json beats no-interactive", false, true, "json", ModeJSON},
		{"yaml beats interactive", true, false, "yaml", ModeYAML},
		{"yaml beats no-interactive", false, true, "yaml", ModeYAML},
		{"no-interactive beats interactive", true, true, "", ModePlain},
		{"empty format with no flags", false, false, "", ModePlain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectMode(tt.interactive, tt.noInteract, tt.format)
			if got != tt.want {
				t.Errorf("DetectMode(%v, %v, %q) = %q, want %q",
					tt.interactive, tt.noInteract, tt.format, got, tt.want)
			}
		})
	}
}

// TestIsInteractive tests mode check helper.
func TestIsInteractive(t *testing.T) {
	tests := []struct {
		mode OutputMode
		want bool
	}{
		{ModeInteractive, true},
		{ModePlain, false},
		{ModeJSON, false},
		{ModeYAML, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			got := IsInteractive(tt.mode)
			if got != tt.want {
				t.Errorf("IsInteractive(%q) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

// TestShouldUseColor_JSONAndYAML_NeverColor tests that JSON/YAML never use color.
func TestShouldUseColor_JSONAndYAML_NeverColor(t *testing.T) {
	if ShouldUseColor(ModeJSON) {
		t.Error("JSON mode should not use color")
	}
	if ShouldUseColor(ModeYAML) {
		t.Error("YAML mode should not use color")
	}
}

// TestDetectMode_Metamorphic_JSONAlwaysJSON verifies that regardless of
// interactive/noInteractive flags, format=json always returns ModeJSON.
func TestDetectMode_Metamorphic_JSONAlwaysJSON(t *testing.T) {
	bools := []bool{true, false}
	for _, i := range bools {
		for _, n := range bools {
			got := DetectMode(i, n, "json")
			if got != ModeJSON {
				t.Errorf("DetectMode(%v, %v, json) = %q, want json", i, n, got)
			}
		}
	}
}
