package commands

import (
	"fmt"
	"strings"
	"time"

	"clist/internal/cli"
	"clist/internal/storage"
	"clist/internal/task"
)

var addCmd = &cli.Command{
	Name:    "add",
	Aliases: []string{"a"},
	Summary: "Add a new task",
	Usage:   "clist add <task description>",
	Long: `Add a new task using GTD syntax:

    clist add Buy milk #groceries !high due:today

Modifiers:
    #tag      add a tag (repeatable)
    !priority one of: critical, high, medium, low
    due:DATE  YYYY-MM-DD, today, 1d, -1d, 1m, 2m`,
}

func init() { addCmd.Run = runAdd }

func runAdd(ctx *cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageErrorf(ctx, addCmd, "missing task description")
	}

	text := strings.Join(args, " ")
	title, tags, priority, dueDate := task.ParseInput(text)
	if title == "" {
		return cli.UsageErrorf(ctx, addCmd, "task title is empty")
	}

	t := task.Task{
		Title:     title,
		Tags:      tags,
		Priority:  priority,
		DueDate:   dueDate,
		CreatedAt: time.Now(),
	}
	if err := storage.AddTask(ctx.DB, t); err != nil {
		return fmt.Errorf("add task: %w", err)
	}
	fmt.Fprintf(ctx.Stdout, "Added: %s\n", title)
	return nil
}
