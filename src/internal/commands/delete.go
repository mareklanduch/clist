package commands

import (
	"fmt"
	"strconv"

	"clist/internal/cli"
)

var deleteCmd = &cli.Command{
	Name:    "delete",
	Aliases: []string{"del", "rm"},
	Summary: "Delete a task by id",
	Usage:   "clist delete <id>",
}

func init() { deleteCmd.Run = runDelete }

func runDelete(ctx *cli.Context, args []string) error {
	if len(args) < 1 {
		return cli.UsageErrorf(ctx, deleteCmd, "missing task id")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return cli.UsageErrorf(ctx, deleteCmd, "invalid id %q", args[0])
	}
	if err := ctx.Backend.DeleteTask(id); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	fmt.Fprintf(ctx.Stdout, "Deleted #%d\n", id)
	return nil
}
