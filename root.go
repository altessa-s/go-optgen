// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package optgen

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"

	"github.com/altessa-s/go-optgen/internal/rootcmd"
)

func rootOptions(opts []Option) []rootcmd.Option {
	return []rootcmd.Option{
		rootcmd.WithLong(heredoc.Doc(`
			optgen generates functional option functions from struct field tags.

			The generator looks for fields with 'opt' tags in the specified struct type
			and generates corresponding inline WithXxx functions.

			Tag Format:
			  opt:"Name"  - Generate option
			  opt:"-"     - Skip generation

			Examples:
			  type options struct {
			      logger     *slog.Logger     opt:"Logger"                     // generates WithLogger
			      timeout    time.Duration    opt:"Timeout" optgen:"default=30*time.Second" // generates WithTimeout
			      hosts      []string         opt:"Hosts" optgen:"append"      // slice append semantics
			      name       string           opt:"Name" optval:"trimspaces,lower" // value modifiers
			      custom     string           opt:"-"                          // skipped - implement manually
			  }
		`)),
		rootcmd.WithExample(heredoc.Doc(`
			# Generate options for a struct type
			optgen generate --type options

			# Generate with custom output file
			optgen generate --type options --output my_options_gen.go

			# Preview changes before committing (dry-run mode)
			optgen generate --type options --dry-run

			# Show detailed generation progress (verbose mode)
			optgen generate --type options --verbose

			# Combine dry-run with verbose for complete visibility
			optgen generate --type options --dry-run --verbose

			# Show version information
			optgen --version
		`)),
		rootcmd.WithSubcommands(NewGenerate(opts...), NewListPlugins(opts...)),
	}
}

// New creates the CLI root command. Options such as [WithPlugins] apply to
// every subcommand.
func New(opts ...Option) *cobra.Command {
	return rootcmd.New("optgen", "Generate functional option functions from struct tags", rootOptions(opts)...)
}

// Run executes the CLI with panic recovery. A program that registers its own
// plugins wraps it in its main function:
//
//	func main() {
//		if err := optgen.Run(os.Args[1:], optgen.WithPlugins(myplugins.Register)); err != nil {
//			os.Exit(1)
//		}
//	}
func Run(args []string, opts ...Option) error {
	return rootcmd.Run("optgen", "Generate functional option functions from struct tags", args, rootOptions(opts)...)
}
