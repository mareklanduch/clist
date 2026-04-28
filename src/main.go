// Command clist is a terminal todo list with a Bubble Tea TUI and a
// flexible CLI subcommand registry.
//
//	clist                  # launch the TUI
//	clist add Buy milk !high
//	clist a Buy milk       # alias for `add`
//	clist list             # list pending tasks
//	clist help             # show every command
package main

import (
	"os"

	"clist/internal/cli"
	"clist/internal/commands"
	"clist/internal/storage"
	"clist/internal/tui"
)

func main() {
	db, err := storage.Open()
	if err != nil {
		cli.ExitOnError(err)
	}
	defer db.Close()

	registry := cli.New()
	registry.RegisterAll(commands.All())

	ctx := &cli.Context{
		DB:     db,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	// No args (or unknown command) → drop into the TUI. This preserves
	// the "just type `clist`" experience while keeping subcommands fast.
	defaultRun := func(_ *cli.Context, _ []string) error {
		return tui.Run(db)
	}

	if err := registry.Dispatch(ctx, os.Args[1:], defaultRun); err != nil {
		cli.ExitOnError(err)
	}
}
