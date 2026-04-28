package commands

import (
	"fmt"
	"strings"

	"clist/internal/cli"
)

var helpCmd = &cli.Command{
	Name:    "help",
	Aliases: []string{"h", "--help", "-h"},
	Summary: "Show help for CLIst or a specific command",
	Usage:   "clist help [command]",
}

func init() { helpCmd.Run = runHelp }

func runHelp(ctx *cli.Context, args []string) error {
	fmt.Fprintln(ctx.Stdout, Banner)
	fmt.Fprintln(ctx.Stdout)

	if len(args) == 0 {
		ctx.Registry.PrintUsage(ctx.Stdout, "clist")
		return nil
	}

	cmd := ctx.Registry.Lookup(args[0])
	if cmd == nil {
		return cli.UsageErrorf(ctx, helpCmd, "unknown command %q", args[0])
	}

	header := cmd.Name
	if len(cmd.Aliases) > 0 {
		header += " (aliases: " + strings.Join(cmd.Aliases, ", ") + ")"
	}
	fmt.Fprintln(ctx.Stdout, header)
	fmt.Fprintln(ctx.Stdout, strings.Repeat("─", len(header)))
	if cmd.Summary != "" {
		fmt.Fprintln(ctx.Stdout, cmd.Summary)
	}
	if cmd.Usage != "" {
		fmt.Fprintf(ctx.Stdout, "\nUsage: %s\n", cmd.Usage)
	}
	if cmd.Long != "" {
		fmt.Fprintln(ctx.Stdout)
		fmt.Fprintln(ctx.Stdout, cmd.Long)
	}
	return nil
}
