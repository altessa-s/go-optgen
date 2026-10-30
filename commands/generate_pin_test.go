// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/commands"
	"github.com/altessa-s/go-optgen/config"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin"
	"github.com/altessa-s/go-optgen/plugin"
)

// execGeneratePin runs the generate command with args and returns its error.
func execGeneratePin(tb testing.TB, args ...string) error {
	tb.Helper()
	cmd := commands.NewGenerate()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	return cmd.ExecuteContext(tb.Context())
}

// sampleDir returns a package directory holding sampleSource, plus an
// optional .optgen.yaml with config.
func sampleDir(tb testing.TB, config string) string {
	tb.Helper()
	dir := tb.TempDir()
	require.NoError(tb, os.WriteFile(filepath.Join(dir, "options.go"), []byte(sampleSource), 0o600))
	if config != "" {
		require.NoError(tb, os.WriteFile(filepath.Join(dir, ".optgen.yaml"), []byte(config), 0o600))
	}
	return dir
}

func TestGenerate_PreRunValidationPin(t *testing.T) {
	t.Parallel()
	dir := sampleDir(t, "")
	file := filepath.Join(dir, "options.go")
	locked := filepath.Join(t.TempDir(), "locked")
	require.NoError(t, os.Mkdir(locked, 0o700))
	require.NoError(t, os.Chmod(locked, 0))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	base := []string{"--no-config", "--no-format", "--dry-run"}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"missing directory", []string{"--directory", filepath.Join(dir, "missing")}, "directory does not exist: "},
		{"file as directory", []string{"--directory", file}, "path is not a directory: " + file},
		{"inaccessible directory", []string{"--directory", filepath.Join(locked, "x")}, "failed to access directory"},
		{"empty type", []string{"--directory", dir, "--type", ""}, "type name cannot be empty"},
		{"bad type", []string{"--directory", dir, "--type", "1x"}, "invalid type name"},
		{"empty option type", []string{"--directory", dir, "--option-type", ""}, "option type cannot be empty"},
		{"bad option type", []string{"--directory", dir, "--option-type", "a-b"}, "invalid option type"},
		{"bad default func", []string{"--directory", dir, "--default-func-name", "a-b"}, "invalid default function name"},
		{"bad new func", []string{"--directory", dir, "--new-func-name", "a-b"}, "invalid new function name"},
		{"same funcs", []string{"--directory", dir, "--default-func-name", "f", "--new-func-name", "f"}, `must differ (both are "f")`},
		{"unknown plugin", []string{"--directory", dir, "--disable-plugins", "nosuchplugin"}, `unknown plugin "nosuchplugin"`},
		{"removed plugin flag", []string{"--directory", dir, "--plugin", "x.so"}, "unknown flag: --plugin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorContains(t, execGeneratePin(t, append(base, tc.args...)...), tc.want)
		})
	}
}

func TestGenerate_PreRunPluginsPin(t *testing.T) {
	t.Parallel()
	r := plugin.NewRegistry()
	require.NoError(t, builtin.Register(r))
	var known []string
	for _, d := range r.Descriptors() {
		known = append(known, d.Name)
	}
	require.NotEmpty(t, known)

	t.Run("known plugins are disabled, blanks ignored", func(t *testing.T) {
		t.Parallel()
		dir := sampleDir(t, "")
		require.NoError(t, execGeneratePin(t, "--directory", dir, "--no-config", "--no-format", "--dry-run",
			"--disable-plugins", " , "+known[0]+" ,"))
	})

	t.Run("config plugins and disabled plugins apply", func(t *testing.T) {
		t.Parallel()
		dir := sampleDir(t, "defaults:\n  disable-plugins: [nosuchplugin]\n")
		require.ErrorContains(t, execGeneratePin(t, "--directory", dir, "--no-format", "--dry-run"), `unknown plugin "nosuchplugin"`)

		dir = sampleDir(t, "defaults:\n  plugins: [missing.so]\n")
		require.ErrorIs(t, execGeneratePin(t, "--directory", dir, "--no-format", "--dry-run"), config.ErrPluginsUnsupported)
	})

	t.Run("the CLI flag overrides config disabled plugins", func(t *testing.T) {
		t.Parallel()
		dir := sampleDir(t, "defaults:\n  disable-plugins: [nosuchplugin]\n")
		require.NoError(t, execGeneratePin(t, "--directory", dir, "--no-format", "--dry-run", "--disable-plugins", known[0]))
	})

	t.Run("a malformed config fails", func(t *testing.T) {
		t.Parallel()
		dir := sampleDir(t, "defaults: [\n")
		require.Error(t, execGeneratePin(t, "--directory", dir, "--no-format", "--dry-run"))
	})
}
