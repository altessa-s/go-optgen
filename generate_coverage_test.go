// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package optgen_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	optgen "github.com/altessa-s/go-optgen"
)

// newPkgDir writes src as options.go into a fresh directory that is marked
// as a project root (go.mod), so config lookup never escapes the temp dir.
func newPkgDir(tb testing.TB, src string) string {
	tb.Helper()
	dir := tb.TempDir()
	require.NoError(tb, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module sample\n"), 0o600))
	require.NoError(tb, os.WriteFile(filepath.Join(dir, "options.go"), []byte(src), 0o600))
	return dir
}

func execGenerate(tb testing.TB, args ...string) (string, error) {
	tb.Helper()
	cmd := optgen.NewGenerate()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(tb.Context())
	return out.String(), err
}

func requireGofmt(tb testing.TB) {
	tb.Helper()
	if _, err := exec.LookPath("gofmt"); err != nil {
		tb.Skip("gofmt not on PATH")
	}
}

func TestGenerate_PreRunValidation(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, sampleSource)
	file := filepath.Join(dir, "options.go")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"missing directory", []string{"--directory", filepath.Join(dir, "nope")}, "directory does not exist"},
		{"not a directory", []string{"--directory", file}, "path is not a directory"},
		{"empty type", []string{"--directory", dir, "--type", ""}, "type name cannot be empty"},
		{"invalid type", []string{"--directory", dir, "--type", "1bad"}, "invalid type name"},
		{"empty option type", []string{"--directory", dir, "--option-type", ""}, "option type cannot be empty"},
		{"invalid option type", []string{"--directory", dir, "--option-type", "Opt-x"}, "invalid option type"},
		{"invalid default func", []string{"--directory", dir, "--default-func-name", "a b"}, "invalid default function name"},
		{"invalid new func", []string{"--directory", dir, "--new-func-name", "9x"}, "invalid new function name"},
		{"same func names", []string{"--directory", dir, "--default-func-name", "mk", "--new-func-name", "mk"}, "must differ"},
		{"invalid option prefix", []string{"--directory", dir, "--option-prefix", "bad-prefix"}, "invalid option prefix"},
		{"unknown plugin", []string{"--directory", dir, "--disable-plugins", " , NoSuchPlugin"}, `unknown plugin "NoSuchPlugin"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := execGenerate(t, append([]string{"--no-config", "--no-format"}, tc.args...)...)
			require.ErrorContains(t, err, tc.want)
			require.NoFileExists(t, filepath.Join(dir, "options_gen.go"))
		})
	}
}

func TestGenerate_ConfigFileDefaults(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, `package sample

type settings struct {
	name string `+"`opt:\"Name\"`"+`
}
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".optgen.yaml"), []byte(`
defaults:
  type: options
  no-format: true
packages:
  ".":
    type: settings
    output: settings_gen.go
    option-type: Setting
    option-prefix: My
    option-error: true
    skip-option-type: true
    default-func-name: makeDefaults
    new-func-name: makeSettings
`), 0o600))

	out, err := execGenerate(t, "--directory", dir, "--verbose")
	require.NoError(t, err)
	require.Contains(t, out, "Loaded config from: "+filepath.Join(dir, ".optgen.yaml"))

	content, err := os.ReadFile(filepath.Join(dir, "settings_gen.go"))
	require.NoError(t, err)
	src := string(content)
	require.Contains(t, src, "func WithMyName[")
	require.Contains(t, src, ") Setting {")
	require.Contains(t, src, "func makeDefaults()")
	require.Contains(t, src, "func makeSettings(")
	require.NotContains(t, src, "type Setting func")
	require.Contains(t, src, "error")
}

func TestGenerate_ConfigDisablesUnknownPlugin(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, sampleSource)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".optgen.yaml"),
		[]byte("defaults:\n  disable-plugins: [DefinitelyNotAPlugin]\n"), 0o600))

	_, err := execGenerate(t, "--directory", dir)
	require.ErrorContains(t, err, `unknown plugin "DefinitelyNotAPlugin"`)
}

func TestGenerate_InvalidConfigFile(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, sampleSource)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".optgen.yaml"), []byte("defaults: [1, 2]\n"), 0o600))

	_, err := execGenerate(t, "--directory", dir)
	require.ErrorContains(t, err, "failed to load config")
}

func TestGenerate_Formatter(t *testing.T) {
	t.Parallel()
	requireGofmt(t)

	t.Run("default formatter rewrites file", func(t *testing.T) {
		t.Parallel()
		dir := newPkgDir(t, sampleSource)
		out, err := execGenerate(t, "--directory", dir, "--no-config", "--verbose")
		require.NoError(t, err)
		require.Contains(t, out, "Formatting with: gofmt -w "+filepath.Join(dir, "options_gen.go"))
		require.FileExists(t, filepath.Join(dir, "options_gen.go"))
	})

	t.Run("file placeholder and quotes", func(t *testing.T) {
		t.Parallel()
		dir := newPkgDir(t, sampleSource)
		out, err := execGenerate(t, "--directory", dir, "--no-config", "--verbose",
			"--formatter", `gofmt '-l' "{file}"`)
		require.NoError(t, err)
		path := filepath.Join(dir, "options_gen.go")
		require.Contains(t, out, "Formatting with: gofmt -l "+path+"\n")
	})

	t.Run("formatter failure", func(t *testing.T) {
		t.Parallel()
		dir := newPkgDir(t, sampleSource)
		_, err := execGenerate(t, "--directory", dir, "--no-config", "--formatter", "gofmt --no-such-flag")
		require.ErrorContains(t, err, "failed to format")
	})

	t.Run("whitespace formatter is skipped", func(t *testing.T) {
		t.Parallel()
		dir := newPkgDir(t, sampleSource)
		out, err := execGenerate(t, "--directory", dir, "--no-config", "--verbose", "--formatter", "  ")
		require.NoError(t, err)
		require.NotContains(t, out, "Formatting with")
	})

	t.Run("package config empty formatter disables formatting", func(t *testing.T) {
		t.Parallel()
		dir := newPkgDir(t, sampleSource)
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".optgen.yaml"), []byte("packages:\n  \".\":\n    formatter: \"\"\n"), 0o600))
		out, err := execGenerate(t, "--directory", dir, "--verbose")
		require.NoError(t, err)
		require.NotContains(t, out, "Formatting with")
	})
}

func TestGenerate_FormatterRejected(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		formatter string
		want      string
	}{
		{"shell pipe", "gofmt -w | cat", `disallowed pattern "|"`},
		{"command substitution", "gofmt $(rm -rf x)", `disallowed pattern "$("`},
		{"not allowlisted", "cat", `formatter "cat" is not in the allowlist`},
		{"path to an allowlisted name", "./tools/gofmt -w", `formatter "./tools/gofmt" must be a command name resolved from PATH`},
		{"absolute path", "/usr/bin/gofmt -w", "not a path"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := newPkgDir(t, sampleSource)
			_, err := execGenerate(t, "--directory", dir, "--no-config", "--formatter", tc.formatter)
			require.ErrorContains(t, err, "invalid formatter")
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestGenerate_ShortDryRunPreview(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, `package sample

type options struct {
	n int `+"`opt:\"N\"`"+`
}
`)
	out, err := execGenerate(t, "--directory", dir, "--no-config", "--dry-run",
		"--skip-option-type", "--skip-default-func", "--skip-new-func")
	require.NoError(t, err)
	require.Contains(t, out, "func WithN(")
	require.NotContains(t, out, "(truncated")
	require.NoFileExists(t, filepath.Join(dir, "options_gen.go"))
}

func TestGenerate_VerboseGenericAndNoFields(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, `package sample

type options[T any] struct {
	v T `+"`opt:\"V\"`"+`
}

func (o *options[T]) normalization() {}
`)
	out, err := execGenerate(t, "--directory", dir, "--no-config", "--no-format", "--dry-run", "--verbose")
	require.NoError(t, err)
	require.Contains(t, out, "Generic type parameters: [T any]")
	require.Contains(t, out, "Found normalization method: normalization")

	empty := newPkgDir(t, "package sample\n\ntype options struct{ n int }\n")
	out, err = execGenerate(t, "--directory", empty, "--no-config", "--verbose")
	require.Error(t, err)
	require.Contains(t, out, "No opt fields found in package sample")
}

func TestGenerate_ParseErrors(t *testing.T) {
	t.Parallel()
	broken := newPkgDir(t, "package sample\n\ntype options struct {\n")
	_, err := execGenerate(t, "--directory", broken, "--no-config")
	require.Error(t, err)

	noGo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(noGo, "go.mod"), []byte("module x\n"), 0o600))
	_, err = execGenerate(t, "--directory", noGo, "--no-config")
	require.ErrorContains(t, err, "no packages found")
}

func TestListPlugins(t *testing.T) {
	t.Parallel()
	cmd := optgen.New()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list-plugins"})
	require.NoError(t, cmd.ExecuteContext(t.Context()))

	s := out.String()
	for _, section := range []string{"Field plugins:", "Type default providers:", "Transform plugins:", "Modifier plugins:", "Optcheck plugins:"} {
		require.Contains(t, s, section)
	}
	require.True(t, strings.Contains(s, "BoolFlagPlugin") || strings.Contains(s, "priority="), s)
	require.Contains(t, s, `  - RequiredCheck (key="required", aliases=nonempty)`)
}
