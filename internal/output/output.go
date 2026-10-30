// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package output

import (
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// allowedFormatters is the allowlist of permitted formatter commands.
// Only these command names, resolved from PATH, are allowed to be executed.
var allowedFormatters = map[string]bool{
	"gofmt":     true,
	"goimports": true,
	"gofumpt":   true,
	"golines":   true,
	"crlfmt":    true,
	"gci":       true,
}

// filePlaceholder is the placeholder for the file path in formatter commands.
const filePlaceholder = "{file}"
const defaultFilePermission = 0o644

// validateFormatter checks that the formatter command uses an allow-listed executable
// and contains no shell meta-characters (pipes, semicolons, backticks, etc.).
//
// It returns the executable name, parsed arguments, whether the "{file}" placeholder
// appears in the command string, and any validation error. An empty executable with
// a nil error means the formatter string was empty and formatting should be skipped.
func validateFormatter(formatter string, extra []string) (executable string, args []string, hasFilePlaceholder bool, err error) {
	// Check for shell execution patterns - these indicate attempts to run shell commands
	// Note: quotes are allowed since we use exec.Command directly (no shell interpretation)
	dangerousPatterns := []string{"|", "&&", "||", ";", "`", "$(", ">", "<", "\n"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(formatter, pattern) {
			return "", nil, false, fmt.Errorf("formatter contains disallowed pattern %q; use simple commands like 'gofmt -w'", pattern)
		}
	}

	// Check if {file} placeholder is used
	hasFilePlaceholder = strings.Contains(formatter, filePlaceholder)

	// Parse the formatter string, handling quoted arguments
	parts := parseFormatterArgs(formatter)
	if len(parts) == 0 {
		return "", nil, false, nil // Empty formatter, will be skipped
	}

	executable = parts[0]
	args = parts[1:]

	// A path would let a repository-supplied script borrow an allowlisted name;
	// bare names are resolved from PATH, never from the current directory.
	if strings.ContainsAny(executable, `/\`) {
		return "", nil, false, fmt.Errorf("formatter %q must be a command name resolved from PATH, not a path", executable)
	}
	baseName := executable

	if !allowedFormatters[baseName] && !slices.Contains(extra, baseName) {
		allowed := slices.Concat(slices.Collect(maps.Keys(allowedFormatters)), extra)
		slices.Sort(allowed)
		return "", nil, false, fmt.Errorf("formatter %q is not in the allowlist; allowed formatters: %s", baseName, strings.Join(allowed, ", "))
	}

	return executable, args, hasFilePlaceholder, nil
}

// parseFormatterArgs splits a formatter string into arguments, respecting quotes.
// Supports both double and single quotes. Quotes are removed from the result.
func parseFormatterArgs(s string) []string {
	var args []string
	var current strings.Builder
	var inQuote rune // 0 if not in quote, otherwise the quote character

	for _, r := range s {
		switch {
		case inQuote != 0:
			if r == inQuote {
				// End of quoted section
				inQuote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '"' || r == '\'':
			// Start of quoted section
			inQuote = r
		case r == ' ' || r == '\t':
			// Whitespace outside quotes - end of argument
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	// Don't forget the last argument
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// Writer formats temporary files before atomically publishing them.
type Writer struct {
	out         io.Writer
	err         io.Writer
	diagnostics io.Writer
	options     *options
}

// New binds formatter streams and optional progress diagnostics.
func New(out, err, diagnostics io.Writer, opts ...Option) *Writer {
	return &Writer{out: out, err: err, diagnostics: diagnostics, options: newOptions(opts...)}
}

// Write preserves an existing output if formatting fails.
func (w Writer) Write(ctx context.Context, path string, content []byte, formatter string) error {
	executable, args, placeholder, err := validateFormatter(formatter, w.options.allowedFormatters)
	if err != nil {
		return fmt.Errorf("invalid formatter: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".optgen-*.go")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer func() { _ = os.Remove(temporary) }()
	if _, err = file.Write(content); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if executable != "" {
		if placeholder {
			for i := range args {
				args[i] = strings.ReplaceAll(args[i], filePlaceholder, temporary)
			}
		} else {
			args = append(args, temporary)
		}
		if w.diagnostics != nil {
			display := strings.ReplaceAll(strings.Join(args, " "), temporary, path)
			_, _ = fmt.Fprintf(w.diagnostics, "Formatting with: %s %s\n", executable, display)
		}
		command := exec.CommandContext(ctx, executable, args...) // #nosec G204 -- Executable validated against the formatter allowlist.
		command.Stdout = w.out
		command.Stderr = w.err
		if err = command.Run(); err != nil {
			return fmt.Errorf("failed to format %s with %q: %w", path, formatter, err)
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err := os.Chmod(temporary, defaultFilePermission); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
