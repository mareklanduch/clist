package commands

import (
	"fmt"
	"strings"

	"clist/internal/cli"
	"clist/internal/storage"
)

var vaultCmd = &cli.Command{
	Name:    "vault",
	Aliases: []string{"vaults"},
	Summary: "Manage vaults (separate task databases)",
	Usage:   "clist vault <list|add|switch|remove> [name]",
	Long: `Vaults are separate task databases. Each vault stores an independent set of tasks.

Subcommands:
  list              list all vaults (* marks the active one)
  add <name>        create a new vault
  switch <name>     switch the active vault
  remove <name>     remove a vault from the registry (db file is kept)

Examples:
  clist vault list
  clist vault add work
  clist vault switch work
  clist vault remove work`,
	Run: runVault,
}

const vaultUsage = "clist vault <list|add|switch|remove> [name]"

func vaultUsageError(ctx *cli.Context, format string, a ...any) error {
	fmt.Fprintf(ctx.Stderr, "clist vault: "+format+"\nusage: "+vaultUsage+"\n", a...)
	return cli.ErrSilent
}

func runVault(ctx *cli.Context, args []string) error {
	if len(args) == 0 {
		return runVaultList(ctx)
	}
	sub, rest := args[0], args[1:]
	switch strings.ToLower(sub) {
	case "list", "ls", "l":
		return runVaultList(ctx)
	case "add", "a", "new":
		return runVaultAdd(ctx, rest)
	case "switch", "sw", "use", "default":
		return runVaultSwitch(ctx, rest)
	case "remove", "rm", "delete", "del":
		return runVaultRemove(ctx, rest)
	default:
		return vaultUsageError(ctx, "unknown subcommand %q", sub)
	}
}

func runVaultList(ctx *cli.Context) error {
	vc := ctx.Vault
	if vc == nil || len(vc.Vaults) == 0 {
		fmt.Fprintln(ctx.Stdout, "No vaults configured.")
		return nil
	}
	fmt.Fprintln(ctx.Stdout, "Vaults:")
	for _, v := range vc.Vaults {
		marker := "  "
		if v.Name == vc.Active {
			marker = "* "
		}
		count, _ := vaultTaskCount(v.Path)
		fmt.Fprintf(ctx.Stdout, "%s%-20s  %s  (%d tasks)\n", marker, v.Name, v.Path, count)
	}
	return nil
}

func runVaultAdd(ctx *cli.Context, args []string) error {
	if len(args) == 0 {
		return vaultUsageError(ctx, "add requires a vault name")
	}
	name := args[0]
	if err := ctx.Vault.Add(name); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Vault %q created.\n", name)
	return nil
}

func runVaultSwitch(ctx *cli.Context, args []string) error {
	if len(args) == 0 {
		return vaultUsageError(ctx, "switch requires a vault name")
	}
	name := args[0]
	if err := ctx.Vault.Switch(name); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Switched to vault %q.\n", name)
	return nil
}

func runVaultRemove(ctx *cli.Context, args []string) error {
	if len(args) == 0 {
		return vaultUsageError(ctx, "remove requires a vault name")
	}
	name := args[0]
	activeChanged, err := ctx.Vault.Remove(name)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Vault %q removed (db file kept on disk).\n", name)
	if activeChanged {
		fmt.Fprintf(ctx.Stdout, "Active vault switched to %q.\n", ctx.Vault.Active)
	}
	return nil
}

// vaultTaskCount opens the vault db briefly to count active tasks.
func vaultTaskCount(dbPath string) (int, error) {
	db, err := storage.OpenAt(dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	tasks, err := storage.AllTasks(db)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, t := range tasks {
		if !t.Archived {
			count++
		}
	}
	return count, nil
}
