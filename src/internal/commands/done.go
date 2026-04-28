package commands

import (
	"fmt"
	"strconv"

	"clist/internal/cli"
	"clist/internal/storage"
	"clist/internal/task"
)

var doneCmd = &cli.Command{
	Name:    "done",
	Aliases: []string{"d"},
	Summary: "Mark a task as done",
	Usage:   "clist done <id>",
}

func init() { doneCmd.Run = runDone }

func runDone(ctx *cli.Context, args []string) error {
	if len(args) < 1 {
		return cli.UsageErrorf(ctx, doneCmd, "missing task id")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return cli.UsageErrorf(ctx, doneCmd, "invalid id %q", args[0])
	}
	if err := storage.UpdateStatus(ctx.DB, id, task.StatusDone); err != nil {
		return fmt.Errorf("done: %w", err)
	}
	fmt.Fprintf(ctx.Stdout, "Marked #%d as done\n", id)
	return nil
}
