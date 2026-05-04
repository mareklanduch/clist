// Package commands defines the concrete CLIst subcommands.
//
// To add a new command:
//  1. Drop a new file in this folder, e.g. edit.go
//  2. Declare `var editCmd = &cli.Command{Name: "edit", ..., Run: runEdit}`
//  3. Add it to the All() slice below
//
// That's it — main.go does not need to change.
package commands

import "clist/internal/cli"

// All returns every registered subcommand in display order.
//
// Order here controls the order in --help output (after alphabetical
// sort within the registry). Add new commands by appending here.
func All() []*cli.Command {
	return []*cli.Command{
		addCmd,
		listCmd,
		doneCmd,
		deleteCmd,
		statsCmd,
		vaultCmd,
		helpCmd,
	}
}

// Banner is the CLIst ASCII art used by `help` and the TUI's stats view.
const Banner = `   ___  _      ___      _   
  / __\| |    |_ _| ___| |_ 
 / /  | |     | | / __| __|
/ /___| |___  | | \__ \ |_ 
\____/|_____|___||___/\__|`
