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
	Summary: "Manage vaults (local databases and remote clist-server vaults)",
	Usage:   "clist vault <list|add|switch|remove|remote-new|remote-connect> [args]",
	Long: `Vaults are separate task stores. Local vaults are SQLite files; remote vaults
sync with a clist-server instance via an access token.

Subcommands:
  list                              list all vaults (* marks the active one)
  add <name>                        create a new local vault
  switch <name>                     switch the active vault
  remove <name>                     remove a vault from the registry (db file kept)
  remote-new <name>           generate a new vault on the clist-server
  remote-connect <name> <token>  connect to an existing remote vault by token

Examples:
  clist vault list
  clist vault add work
  clist vault switch work
  clist vault remove work
  clist vault remote-new home
  clist vault remote-connect team X7kQm2pL9nRvTwYz`,
	Run: runVault,
}

const vaultUsage = "clist vault <list|add|switch|remove|remote-new|remote-connect> [args]"

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
	case "remote-new", "rn":
		return runVaultRemoteNew(ctx, rest)
	case "remote-connect", "rc":
		return runVaultRemoteConnect(ctx, rest)
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
		if v.IsRemote() {
			fmt.Fprintf(ctx.Stdout, "%s%-20s  remote  [%s]\n", marker, v.Name, v.Token)
		} else {
			count, _ := vaultTaskCount(v.Path)
			fmt.Fprintf(ctx.Stdout, "%s%-20s  %s  (%d tasks)\n", marker, v.Name, v.Path, count)
		}
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

func runVaultRemoteNew(ctx *cli.Context, args []string) error {
	if len(args) < 1 {
		return vaultUsageError(ctx, "remote-new requires: <name>")
	}
	name := args[0]
	fmt.Fprintln(ctx.Stdout, "Creating vault on clist-server…")
	token, err := storage.CreateRemoteVault()
	if err != nil {
		return fmt.Errorf("create remote vault: %w", err)
	}
	if err := ctx.Vault.AddRemoteVault(name, token); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Remote vault %q created.\nToken: %s\nStore this token — it is the only way to access your vault.\n", name, token)
	return nil
}

func runVaultRemoteConnect(ctx *cli.Context, args []string) error {
	if len(args) < 2 {
		return vaultUsageError(ctx, "remote-connect requires: <name> <token>")
	}
	name, token := args[0], args[1]
	if err := ctx.Vault.AddRemoteVault(name, token); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Remote vault %q connected.\n", name)
	return nil
}

// vaultTaskCount opens a local vault db briefly to count active tasks.
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
