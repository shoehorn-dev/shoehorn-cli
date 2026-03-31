package main

import (
	"fmt"
	"os"

	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	_ "github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands/create" // register create subcommands
	_ "github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands/delete" // register delete subcommands
	_ "github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands/get"    // register get subcommands
	_ "github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands/update" // register update subcommands
)

// version is injected at build time via ldflags:
//
//	go build -ldflags="-X main.version=1.2.3" ./cmd/shoehorn
var version = "dev"

func init() {
	commands.Version = version
}

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
