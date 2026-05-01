// Command clist is a terminal todo list with a Bubble Tea TUI and a
// flexible CLI subcommand registry.
//
//	clist                  # launch the TUI
//	clist add Buy milk !high
//	clist a Buy milk       # alias for `add`
//	clist list             # list pending tasks
//	clist vault list       # list all vaults
//	clist help             # show every command
package main

import (
	"os"

	"clist/internal/cli"
	"clist/internal/commands"
	"clist/internal/storage"
	"clist/internal/tui"
	"clist/internal/vault"
)

func main() {
	dataDir, err := vault.DataDir()
	if err != nil {
		cli.ExitOnError(err)
	}

	vc, err := vault.Load(dataDir)
	if err != nil {
		cli.ExitOnError(err)
	}

	av := vc.ActiveVault()
	if av == nil {
		cli.ExitOnError(err)
	}

	backend, err := storage.NewBackend(av.IsRemote(), av.Path, av.Token)
	if err != nil {
		cli.ExitOnError(err)
	}
	defer backend.Close()

	registry := cli.New()
	registry.RegisterAll(commands.All())

	ctx := &cli.Context{
		Backend: backend,
		Vault:   vc,
		DataDir: dataDir,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}

	defaultRun := func(_ *cli.Context, _ []string) error {
		return tui.Run(backend, vc, dataDir)
	}

	if err := registry.Dispatch(ctx, os.Args[1:], defaultRun); err != nil {
		cli.ExitOnError(err)
	}
}
