package commands

import (
	"fmt"

	"clist/internal/cli"
	"clist/internal/task"
)

var statsCmd = &cli.Command{
	Name:    "stats",
	Aliases: []string{"st"},
	Summary: "Show task statistics",
	Usage:   "clist stats",
	Run:     runStats,
}

func runStats(ctx *cli.Context, args []string) error {
	tasks, err := ctx.Backend.AllTasks()
	if err != nil {
		return fmt.Errorf("stats: %w", err)
	}
	total, done, overdue := len(tasks), 0, 0
	for _, t := range tasks {
		if t.Status == task.StatusDone {
			done++
		}
		if t.IsOverdue() {
			overdue++
		}
	}
	fmt.Fprintf(ctx.Stdout, "Total tasks: %d\n", total)
	fmt.Fprintf(ctx.Stdout, "Done:        %d\n", done)
	fmt.Fprintf(ctx.Stdout, "Pending:     %d\n", total-done)
	fmt.Fprintf(ctx.Stdout, "Overdue:     %d\n", overdue)
	return nil
}
