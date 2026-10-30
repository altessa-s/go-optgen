// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/commands"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const twoTypesSource = "package sample\n" +
	"type checkOptions struct { limit int `opt:\"Limit\"` }\n" +
	"type options struct { name string `opt:\"Name\"` }\n"

func TestGenerateSeveralTypes(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, twoTypesSource)
	_, err := execGenerate(t, "--directory", dir, "--no-config", "--no-format", "--type", "checkOptions, options")
	require.NoError(t, err)
	check, err := os.ReadFile(filepath.Join(dir, "check_options_gen.go"))
	require.NoError(t, err)
	require.Contains(t, string(check), "type CheckOption func(o *checkOptions)")
	require.Contains(t, string(check), "func newCheckOptions(opts ...CheckOption) *checkOptions")
	opts, err := os.ReadFile(filepath.Join(dir, "options_gen.go"))
	require.NoError(t, err)
	require.Contains(t, string(opts), "type Option func(o *options)")

	for _, flag := range []string{"--output=x.go", "--option-type=X", "--default-func-name=d", "--new-func-name=n"} {
		_, err = execGenerate(t, "--directory", dir, "--no-config", "--no-format", "--dry-run", "--type", "checkOptions,options", flag)
		require.ErrorContains(t, err, "cannot be used with several types", flag)
	}
}

func TestGenerateShapeFlags(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, sampleSource)
	out, err := execGenerate(t, "--directory", dir, "--no-config", "--no-format", "--dry-run", "--verbose",
		"--func-prefix", "Set", "--string-setter", "plain", "--keep-empty-strings", "--no-default-modifiers")
	require.NoError(t, err)
	require.Contains(t, out, "func SetName(v string) Option")
	require.Contains(t, out, "  - name (string) -> SetName\n")
	require.NotContains(t, out, "strings.TrimSpace", "default modifiers are off")

	for _, tc := range []struct{ args []string }{
		{[]string{"--string-setter", "fancy"}},
		{[]string{"--check-failure", "ignore"}},
		{[]string{"--func-prefix", "1x"}},
		{[]string{"--allow-formatter", "./fmt"}},
	} {
		_, err := execGenerate(t, append([]string{"--directory", dir, "--no-config", "--dry-run"}, tc.args...)...)
		require.Error(t, err, tc.args)
	}
}

func TestGenerateConfiguredTypeDefaults(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, "package sample\nimport \"time\"\ntype options struct { wait time.Duration `opt:\"Wait\"` }\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".optgen.yaml"),
		[]byte("defaults:\n  type-defaults:\n    time.Duration: 5 * time.Second\n"), 0o600))
	out, err := execGenerate(t, "--directory", dir, "--no-format", "--dry-run")
	require.NoError(t, err)
	require.Contains(t, out, "wait: 5 * time.Second")
}

// rename is an external transform registered through commands.WithPlugins.
type rename struct{ plugin.TransformBase }

func (rename) Apply(_ plugin.GenerationContext, _ model.OptField, _, expr string) (string, bool) {
	return "renamed(" + expr + ")", true
}

func TestWithPluginsRegistersExternalPlugins(t *testing.T) {
	t.Parallel()
	register := func(r *plugin.Registry) error {
		return r.Register(&rename{TransformBase: plugin.NewTransformBase("rename", plugin.WithApplicableTypes("string"))})
	}
	dir := newPkgDir(t, "package sample\ntype options struct { name string `opt:\"Name\" optval:\"rename\"` }\n")
	cmd := commands.New(commands.WithPlugins(register))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"generate", "--directory", dir, "--no-config", "--no-format", "--dry-run"})
	require.NoError(t, cmd.ExecuteContext(t.Context()))
	require.Contains(t, out.String(), "renamed(")

	cmd = commands.New(commands.WithPlugins(register))
	out.Reset()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"list-plugins"})
	require.NoError(t, cmd.ExecuteContext(t.Context()))
	require.Contains(t, out.String(), `rename (key="rename")`)
}
