package commands

import (
	"fmt"

	"clist/internal/cli"
	"clist/internal/storage"
	"clist/internal/task"
)

var listCmd = &cli.Command{
	Name:    "list",
	Aliases: []string{"ls", "l"},
	Summary: "List all pending tasks",
	Usage:   "clist list",
	Run:     runList,
}

func runList(ctx *cli.Context, args []string) error {
	tasks, err := storage.AllTasks(ctx.DB)
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}

	pending := 0
	for _, t := range tasks {
		if t.Status == task.StatusDone {
			continue
		}
		pending++
		due := ""
		if t.DueDate != nil {
			due = " due:" + t.DueDate.Format("2006-01-02")
		}
		fmt.Fprintf(ctx.Stdout, "[%d] %s %s %s%s\n", t.ID, t.PriorityIcon(), t.StatusIcon(), t.Title, due)
	}

	if pending == 0 {
		fmt.Fprintln(ctx.Stdout, "No tasks.")
	}
	return nil
}
