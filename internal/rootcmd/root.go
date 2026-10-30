// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package rootcmd

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

// Command wraps a [cobra.Command] with shared pre-run and post-run hooks
// including panic recovery.
type Command struct {
	*cobra.Command
}

// New creates the root command from required identity and generated options.
func New(use, short string, opts ...Option) *cobra.Command {
	cfg := newOptions(opts...)

	root := &Command{
		Command: &cobra.Command{
			Use:          use,
			Short:        short,
			Long:         cfg.long,
			Example:      cfg.example,
			SilenceUsage: true,
			Version:      Version(),
		},
	}

	root.configure(cfg.subcommands)
	if cfg.out != nil {
		root.SetOut(cfg.out)
	}
	if cfg.err != nil {
		root.SetErr(cfg.err)
	}

	return root.Command
}

// configure sets up the root command with all its subcommands and configurations.
func (c *Command) configure(subcommands []*cobra.Command) {
	// Set standard streams
	// Generated options override these stream defaults.
	c.SetErr(os.Stderr)
	c.SetOut(os.Stdout)

	// Configure command suggestions
	c.SuggestionsMinimumDistance = 1

	// Add version template
	c.SetVersionTemplate(heredoc.Doc(`
		{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
	`))

	// Set pre-run hook with panic recovery wrapper
	c.PersistentPreRunE = c.preRunWithRecovery

	// Set post-run hook for cleanup
	c.PersistentPostRun = c.postRun

	// Add subcommands
	if len(subcommands) > 0 {
		c.AddCommand(subcommands...)
	}
}

// preRunWithRecovery wraps preRun with panic recovery mechanism.
func (c *Command) preRunWithRecovery(cmd *cobra.Command, args []string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Panic recovered: %v\n", r)
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Stack trace:\n%s\n", debug.Stack())
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()
	return c.preRun(cmd, args)
}

// preRun performs common initialization before any command execution.
func (c *Command) preRun(_ *cobra.Command, _ []string) error { return nil }

// postRun performs cleanup after command execution.
func (c *Command) postRun(_ *cobra.Command, _ []string) {}

// Run creates a root command from cfg, sets its args, and executes it.
// A top-level deferred recover wraps the entire Execute call, so panics
// that escape PersistentPreRunE (e.g., in a subcommand's RunE) are still
// caught, logged to stderr, and returned as an error.
func Run(use, short string, args []string, opts ...Option) (err error) {
	cmd := New(use, short, opts...)
	cmd.SetArgs(args)

	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Panic recovered: %v\n", r)
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Stack trace:\n%s\n", debug.Stack())
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	return cmd.Execute()
}

// version is set at build time:
//
//	go build -ldflags "-X 'github.com/altessa-s/go-optgen/internal/rootcmd.version=1.2.3'"
var version string

// Version returns the version set at build time, else the main module version
// recorded by go install (a tag such as v1.2.3), else "devel".
func Version() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "devel"
}
