package main

import (
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
)

func TestVersionDefault_IsDev(t *testing.T) {
	// When not built with ldflags, version defaults to "dev"
	if version != "dev" {
		t.Errorf("default version = %q, want %q", version, "dev")
	}
}

func TestInit_SetsCommandsVersion(t *testing.T) {
	// init() copies package-level version to commands.Version
	if commands.Version != version {
		t.Errorf("commands.Version = %q, want %q (should match main.version)", commands.Version, version)
	}
}

func TestRootCmd_HasExpectedSubcommands(t *testing.T) {
	root := commands.RootCmd()
	if root == nil {
		t.Fatal("RootCmd() should not return nil")
	}

	// After blank imports in main.go, these commands should be registered
	expected := []string{"create", "delete", "get", "update", "auth", "version", "completion"}
	registered := make(map[string]bool)
	for _, sub := range root.Commands() {
		registered[sub.Name()] = true
	}

	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			if !registered[name] {
				t.Errorf("subcommand %q not registered on root", name)
			}
		})
	}
}

func TestRootCmd_HasGlobalFlags(t *testing.T) {
	root := commands.RootCmd()

	flags := []struct {
		name      string
		shorthand string
	}{
		{"config", ""},
		{"profile", ""},
		{"no-interactive", "I"},
		{"interactive", "i"},
		{"output", "o"},
		{"debug", ""},
		{"yes", "y"},
	}

	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := root.PersistentFlags().Lookup(f.name)
			if flag == nil {
				t.Errorf("global flag --%s not found", f.name)
				return
			}
			if f.shorthand != "" && flag.Shorthand != f.shorthand {
				t.Errorf("flag --%s shorthand = %q, want %q", f.name, flag.Shorthand, f.shorthand)
			}
		})
	}
}

func TestRootCmd_Use(t *testing.T) {
	root := commands.RootCmd()
	if root.Use != "shoehorn" {
		t.Errorf("root Use = %q, want %q", root.Use, "shoehorn")
	}
}
