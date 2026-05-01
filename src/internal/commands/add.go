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
    clist add Fix bug !critical @work

Modifiers:
    #tag       add a tag (repeatable)
    !priority  one of: critical, high, medium, low
    due:DATE   YYYY-MM-DD, today, 1d, -1d, 1m, 2m
    @vault     target vault (defaults to the active vault)`,
}

func init() { addCmd.Run = runAdd }

func runAdd(ctx *cli.Context, args []string) error {
	if len(args) == 0 {
		return cli.UsageErrorf(ctx, addCmd, "missing task description")
	}

	text := strings.Join(args, " ")
	title, tags, priority, dueDate, vaultName := task.ParseInput(text)
	if title == "" {
		return cli.UsageErrorf(ctx, addCmd, "task title is empty")
	}

	db := ctx.DB
	targetVault := ctx.Vault.Active

	if vaultName != "" {
		v := ctx.Vault.Get(vaultName)
		if v == nil {
			return fmt.Errorf("vault %q not found (use `clist vault list` to see available vaults)", vaultName)
		}
		if vaultName != ctx.Vault.Active {
			altDB, err := storage.OpenAt(v.Path)
			if err != nil {
				return fmt.Errorf("open vault %q: %w", vaultName, err)
			}
			defer altDB.Close()
			db = altDB
		}
		targetVault = vaultName
	}

	t := task.Task{
		Title:     title,
		Tags:      tags,
		Priority:  priority,
		DueDate:   dueDate,
		CreatedAt: time.Now(),
	}
	if err := storage.AddTask(db, t); err != nil {
		return fmt.Errorf("add task: %w", err)
	}

	if targetVault != ctx.Vault.Active {
		fmt.Fprintf(ctx.Stdout, "Added to [%s]: %s\n", targetVault, title)
	} else {
		fmt.Fprintf(ctx.Stdout, "Added: %s\n", title)
	}
	return nil
}
