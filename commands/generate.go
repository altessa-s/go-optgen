// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package commands

import (
	"cmp"
	"fmt"
	"io"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"

	"github.com/altessa-s/go-optgen/internal/application"
	"github.com/altessa-s/go-optgen/internal/generator"
	"github.com/altessa-s/go-optgen/internal/output"
	"github.com/altessa-s/go-optgen/internal/semantic"
	"github.com/altessa-s/go-optgen/internal/source"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// GenerateCommand implements the "generate" subcommand.
//
// It parses the target directory for Go source files, locates the struct type
// identified by --type, collects opt-tagged fields, runs the plugin pipeline,
// and writes the generated WithXxx functions to --output (default options_gen.go).
// Use --dry-run to preview output without writing files.
type GenerateCommand struct {
	*cobra.Command
	s        generateSettings
	options  *options
	registry *plugin.Registry
	targets  []target
}

// NewGenerate returns a cobra command that generates functional option functions
// from struct field tags. The returned command validates inputs in PersistentPreRunE
// (directory existence, identifier validity, config loading) before running generation.
func NewGenerate(opts ...Option) *cobra.Command {
	cmd := &GenerateCommand{
		options: newOptions(opts...),
		Command: &cobra.Command{
			Use:   "generate",
			Short: "Generate functional option functions",
			Long: heredoc.Doc(`
				Generate functional option functions from struct field tags.

				The generator scans the specified directory (default: current directory)
				for Go files containing struct types with 'opt' tags, then generates
				WithXxx functions for each tagged field.

				The 'opt' tag format:
				  opt:"Name"  - Generate option
				  opt:"-"     - Skip generation

				Where:
				  - Name: The option name (used in WithName function).
				  - Use optgen/optval/optcheck tags for defaults/modifiers/validations.

				The generator creates a file with:
				  - WithXxx functions for each tagged field
				  - defaultOptions() function with default values
				  - newOptions() function to create instances with options
			`),
			Example: heredoc.Doc(`
				# Generate options for 'options' struct type in current directory
				optgen generate

				# Generate with custom type name
				optgen generate --type config

				# Generate with custom output file
				optgen generate --output my_options.go

				# Generate with custom option type name
				optgen generate --option-type ConfigOption

				# Generate for specific directory
				optgen generate --directory ./mypackage

				# Preview what would be generated without writing files
				optgen generate --dry-run

				# Show detailed generation progress
				optgen generate --verbose

				# Combine dry-run with verbose for maximum visibility
				optgen generate --dry-run --verbose
			`),
		},
	}

	cmd.configure()

	return cmd.Command
}

// configure sets up the generate command with flags and run function.
func (c *GenerateCommand) configure() {
	c.Flags().StringVarP(&c.s.typeName, "type", "t", defaultTypeName, heredoc.Doc(`
		Name of the options struct type to generate functions for.
		The generator will search for this type in the specified directory.
	`))

	c.Flags().StringVarP(&c.s.output, "output", "o", "", heredoc.Doc(`
		Output file name for generated code.
		Default: options_gen.go
	`))

	c.Flags().StringVar(&c.s.optionType, "option-type", defaultOptionType, heredoc.Doc(`
		Name of the Option type (function signature).
		This will be used as the return type of generated WithXxx functions.
	`))

	c.Flags().BoolVar(&c.s.optionError, "option-error", false, heredoc.Doc(`
		Generate Option type as func(*T) error and newOptions returning (*T, error).
		This is useful when some options may validate input and return errors.
	`))

	c.Flags().StringVarP(&c.s.directory, "directory", "d", ".", heredoc.Doc(`
		Directory to process.
		The generator will scan this directory for Go files containing
		the specified struct type with opt tags.
	`))

	c.Flags().BoolVar(&c.s.skipOptionType, "skip-option-type", false, heredoc.Doc(`
		Skip generating the Option type definition in the generated file.
		By default, the generator includes:
		  type Option func(o *options)
		Use this flag if you want to define the Option type manually.
	`))

	c.Flags().BoolVar(&c.s.skipDefaultFunc, "skip-default-func", false, heredoc.Doc(`
		Skip generating the defaultOptions function in the generated file.
		Use this flag if you want to define the default function manually.
	`))

	c.Flags().BoolVar(&c.s.skipNewFunc, "skip-new-func", false, heredoc.Doc(`
		Skip generating the newOptions function in the generated file.
		Use this flag if you want to define the new function manually.
	`))

	c.Flags().BoolVar(&c.s.dryRun, "dry-run", false, heredoc.Doc(`
		Show what would be generated without writing to files.
		Useful for previewing changes before committing them.
	`))

	c.Flags().BoolVarP(&c.s.verbose, "verbose", "v", false, heredoc.Doc(`
		Enable verbose output showing detailed generation progress.
		Displays information about parsed fields and imports used.
	`))

	c.Flags().StringVar(&c.s.disablePlugins, "disable-plugins", "", heredoc.Doc(`
		Comma-separated list of plugin names to disable.
		Plugins are identified by their type name and are matched case-insensitively.
		Use this to disable specific plugins if they interfere with your code generation.
		Example: --disable-plugins=LoggerPlugin,NetIPParsePlugin
	`))

	c.Flags().StringVar(&c.s.defaultFunc, "default-func-name", "", heredoc.Doc(`
		Name of the function that returns default values.
		By default, uses "default<TypeName>" pattern (e.g., defaultOptions, defaultVerifierOptions).
		Example: --default-func-name=myCustomDefaultFunc
	`))

	c.Flags().StringVar(&c.s.newFunc, "new-func-name", "", heredoc.Doc(`
		Name of the function that creates new instances with options.
		By default, uses "new<TypeName>" pattern (e.g., newOptions, newVerifierOptions).
		Example: --new-func-name=myCustomNewFunc
	`))

	c.Flags().StringVar(&c.s.optionPrefix, "option-prefix", "", heredoc.Doc(`
		Prefix to add to all generated option function names.
		For field "issuer" with --option-prefix=Validation:
		  - Without prefix: WithIssuer
		  - With prefix:    WithValidationIssuer
		This is useful when generating validation options or other specialized option sets.
	`))

	c.Flags().BoolVar(&c.s.processAllFields, "all-fields", false, heredoc.Doc(`
		Process all struct fields, not just those with opt/optgen/optval/optcheck tags.
		Fields with opt:"-" are still skipped.
		Useful with --option-prefix to generate options for all fields without adding tags.
	`))

	c.Flags().BoolVar(&c.s.noConfig, "no-config", false, heredoc.Doc(`
		Skip loading .optgen.yaml configuration file.
		By default, optgen searches for .optgen.yaml in the current directory
		and parent directories up to the project root (go.mod or .git).
	`))

	c.Flags().StringVar(&c.s.formatter, "formatter", "gofmt -w", heredoc.Doc(`
		Command to format generated files. The file path is appended to the command.
		Set to empty string or use --no-format to disable formatting.

		For security, only allowlisted formatter names, resolved from PATH, are permitted:
		  gofmt, goimports, gofumpt, golines, crlfmt, gci

		Shell operators (|, &&, ;, etc.) are not allowed.
		Examples: "gofmt -w", "goimports -w", "gofumpt -w"
	`))

	c.Flags().BoolVar(&c.s.noFormat, "no-format", false, heredoc.Doc(`
		Disable formatting of generated files.
		By default, generated files are formatted with "gofmt -w".
	`))

	// Set pre-run hook for validation
	c.Flags().StringArrayVar(&c.s.allowFormatters, "allow-formatter", nil, heredoc.Doc(`
		Add a command name, resolved from PATH, to the formatter allowlist.
		Repeat the flag for several names. Not available in .optgen.yaml.
	`))

	c.Flags().StringVar(&c.s.funcPrefix, "func-prefix", "", heredoc.Doc(`
		Prefix of generated option functions (default "With").
	`))

	c.Flags().StringVar(&c.s.invertFuncPrefix, "invert-func-prefix", "", heredoc.Doc(`
		Prefix of inverted bool flag options (optval:"invert"; default "Without").
	`))

	c.Flags().StringVar(&c.s.stringSetter, "string-setter", "", heredoc.Doc(`
		Signature of string options: "generic" accepts string or *string
		(default), "plain" accepts a string.
	`))

	c.Flags().BoolVar(&c.s.keepEmptyStrings, "keep-empty-strings", false, heredoc.Doc(`
		Assign empty strings instead of skipping them.
	`))

	c.Flags().StringVar(&c.s.checkFailure, "check-failure", "", heredoc.Doc(`
		How options without an error result report a failed optcheck:
		"panic" (default) or "skip", which keeps the previous value.
	`))

	c.Flags().BoolVar(&c.s.noDefaultMods, "no-default-modifiers", false, heredoc.Doc(`
		Do not apply the modifiers plugins add by default, such as trimspaces
		for strings, notnil for pointers and dedup for slices.
	`))

	c.PersistentPreRunE = c.preRun

	c.RunE = c.run
}

// preRun resolves settings from flags and .optgen.yaml, validates them, and
// assembles the plugin registry of this invocation.
func (c *GenerateCommand) preRun(cmd *cobra.Command, _ []string) error {
	if !c.s.noConfig {
		if err := c.s.loadConfig(cmd.Flags(), cmd.OutOrStdout()); err != nil {
			return err
		}
	}
	registry, err := newSessionRegistry(cmd.ErrOrStderr(), c.options, c.s.typeDefaults, c.s.disabledPluginNames())
	if err != nil {
		return err
	}
	c.registry = registry
	if err = c.s.validateDirectory(); err != nil {
		return err
	}
	if c.targets, err = c.s.targets(cmd.Flags()); err != nil {
		return err
	}
	return c.s.validateNames()
}

func (c *GenerateCommand) run(cmd *cobra.Command, _ []string) error {
	s := c.s
	out := cmd.OutOrStdout()
	if s.verbose {
		_, _ = fmt.Fprintf(out, "Parsing directory: %s\nLooking for type: %s\n", s.directory, s.typeName)
	}
	var diagnostics io.Writer
	if s.verbose {
		diagnostics = out
	}
	preparer := semantic.New(c.registry,
		semantic.WithPrefix(s.optionPrefix),
		semantic.WithDefaultModifiers(!s.noDefaultMods),
		semantic.WithWarningSink(warningSink(cmd.ErrOrStderr())),
	)
	service := application.New(
		reportingSource{source: source.Reader{}, out: out, verbose: s.verbose},
		reportingPreparer{preparer: preparer, out: out, verbose: s.verbose,
			funcPrefix: cmp.Or(s.funcPrefix, model.DefaultFuncPrefix)},
		generator.New(c.registry, plugin.NewPipeline(c.registry)),
		output.New(out, cmd.ErrOrStderr(), diagnostics, output.WithAllowedFormatters(s.allowFormatters...)),
	)
	var artifacts []application.Artifact
	for _, t := range c.targets {
		generated, err := service.Generate(cmd.Context(), s.request(t))
		if err != nil {
			return err
		}
		artifacts = append(artifacts, generated...)
	}
	for _, artifact := range artifacts {
		if s.dryRun {
			printPreview(out, artifact.Path, artifact.Options, artifact.Content)
		} else {
			_, _ = fmt.Fprintf(out, "Generated %s with %d option(s)\n", artifact.Path, artifact.Options)
		}
	}
	return nil
}

// printPreview writes the dry-run report for outputPath: the option count and
// the first previewBytes of the generated content.
func printPreview(w io.Writer, outputPath string, options int, content []byte) {
	const previewBytes = 1000

	_, _ = fmt.Fprintf(w, "DRY RUN: Would generate %s with %d option(s)\n", outputPath, options)
	_, _ = fmt.Fprintf(w, "\n--- Generated content preview (first %d bytes) ---\n", previewBytes)
	preview := content
	if len(preview) > previewBytes {
		preview = content[:previewBytes]
		_, _ = fmt.Fprintf(w, "%s\n...\n(truncated, total size: %d bytes)\n", preview, len(content))
	} else {
		_, _ = fmt.Fprintf(w, "%s\n", preview)
	}
	_, _ = fmt.Fprintf(w, "--- End preview ---\n")
}
