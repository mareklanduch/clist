// Package cli provides a tiny command registry that makes it trivial to add
// new top-level subcommands to CLIst.
//
// Adding a new command:
//  1. Create a file under internal/commands/<name>.go
//  2. Declare a `var <name>Cmd = &cli.Command{...}`
//  3. Append it to the slice returned by commands.All()
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"clist/internal/storage"
	"clist/internal/vault"
)

// Context is passed to every command's Run function. It bundles shared
// dependencies (backend, output streams) so commands stay decoupled
// from globals and easy to test.
type Context struct {
	Backend storage.Backend
	Vault   *vault.Config
	DataDir string
	Stdout  io.Writer
	Stderr  io.Writer
	// Registry is set by Dispatch so commands like `help` can introspect.
	Registry *Registry
}

// RunFunc is the signature every command implements.
type RunFunc func(ctx *Context, args []string) error

// Command describes a single CLI subcommand.
type Command struct {
	Name    string   // primary name, e.g. "add"
	Aliases []string // alternate names, e.g. ["a"]
	Summary string   // short one-line description for help
	Usage   string   // usage line shown on errors / help
	Long    string   // optional longer help text
	Run     RunFunc
}

// Names returns the primary name plus any aliases.
func (c *Command) Names() []string {
	out := append([]string{c.Name}, c.Aliases...)
	return out
}

// Registry holds all known commands and resolves names/aliases.
type Registry struct {
	primary []*Command          // preserves registration order for help
	byName  map[string]*Command // name + aliases → command
}

// New creates an empty registry.
func New() *Registry {
	return &Registry{byName: make(map[string]*Command)}
}

// Register adds a command. Panics on duplicate name/alias to surface
// programmer errors at startup.
func (r *Registry) Register(c *Command) {
	if c == nil || c.Name == "" {
		panic("cli: refusing to register nil or unnamed command")
	}
	for _, n := range c.Names() {
		if _, exists := r.byName[n]; exists {
			panic(fmt.Sprintf("cli: duplicate command name or alias %q", n))
		}
		r.byName[n] = c
	}
	r.primary = append(r.primary, c)
}

// RegisterAll is a convenience for bulk registration.
func (r *Registry) RegisterAll(cmds []*Command) {
	for _, c := range cmds {
		r.Register(c)
	}
}

// Lookup finds a command by name or alias. Returns nil if not found.
func (r *Registry) Lookup(name string) *Command {
	return r.byName[name]
}

// Commands returns commands in registration order.
func (r *Registry) Commands() []*Command {
	out := make([]*Command, len(r.primary))
	copy(out, r.primary)
	return out
}

// Dispatch resolves args[0] to a command and runs it. If no command
// matches, defaultCmd is invoked with the full args (useful for falling
// back to the TUI).
func (r *Registry) Dispatch(ctx *Context, args []string, defaultCmd RunFunc) error {
	ctx.Registry = r
	if len(args) == 0 {
		return defaultCmd(ctx, args)
	}
	if cmd := r.Lookup(args[0]); cmd != nil {
		return cmd.Run(ctx, args[1:])
	}
	return defaultCmd(ctx, args)
}

// PrintUsage writes a formatted overview of every registered command.
func (r *Registry) PrintUsage(w io.Writer, programName string) {
	fmt.Fprintf(w, "Usage: %s [command] [args...]\n", programName)
	fmt.Fprintf(w, "       %s                # launch interactive TUI\n\n", programName)
	fmt.Fprintln(w, "Commands:")

	cmds := r.Commands()
	sort.SliceStable(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })

	width := 0
	for _, c := range cmds {
		if w := len(commandHeader(c)); w > width {
			width = w
		}
	}

	for _, c := range cmds {
		fmt.Fprintf(w, "  %-*s   %s\n", width, commandHeader(c), c.Summary)
	}

	fmt.Fprintf(w, "\nRun '%s help <command>' for details on a single command.\n", programName)
}

func commandHeader(c *Command) string {
	if len(c.Aliases) == 0 {
		return c.Name
	}
	return c.Name + ", " + strings.Join(c.Aliases, ", ")
}

// UsageErrorf prints an error and a usage hint to stderr; intended for use
// from inside a command's Run when args validation fails. It is a free
// function (not a method) so that command vars can reference it without
// creating an initialization cycle.
func UsageErrorf(ctx *Context, c *Command, format string, a ...any) error {
	fmt.Fprintf(ctx.Stderr, "clist %s: ", c.Name)
	fmt.Fprintf(ctx.Stderr, format+"\n", a...)
	if c.Usage != "" {
		fmt.Fprintf(ctx.Stderr, "usage: %s\n", c.Usage)
	}
	return ErrSilent
}

// ErrSilent signals that an error has already been printed and main()
// should exit non-zero without re-printing.
var ErrSilent = silentError{}

type silentError struct{}

func (silentError) Error() string { return "" }

// IsSilent reports whether err was produced via UsageErrorf and
// already printed. Wrapped errors are unwrapped.
func IsSilent(err error) bool {
	return errors.Is(err, ErrSilent)
}

// ExitOnError prints err (unless silent) and exits with code 1.
func ExitOnError(err error) {
	if err == nil {
		return
	}
	if !IsSilent(err) {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)
}
