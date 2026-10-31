// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package optgen_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	optgen "github.com/altessa-s/go-optgen"
)

const sampleSource = `package sample

type options struct {
	name    string ` + "`opt:\"Name\"`" + `
	retries int    ` + "`opt:\"Retries\"`" + `
}
`

// runGenerate runs the generate command on a fresh package directory holding
// sampleSource and returns the directory and the command's stdout.
func runGenerate(tb testing.TB, args ...string) (string, string) {
	tb.Helper()
	dir := tb.TempDir()
	require.NoError(tb, os.WriteFile(filepath.Join(dir, "options.go"), []byte(sampleSource), 0o600))

	cmd := optgen.NewGenerate()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{"--directory", dir, "--no-config", "--no-format"}, args...))
	require.NoError(tb, cmd.ExecuteContext(tb.Context()))
	return dir, out.String()
}

func TestGenerate_WritesOptionsFile(t *testing.T) {
	t.Parallel()
	dir, out := runGenerate(t)

	path := filepath.Join(dir, "options_gen.go")
	require.Equal(t, "Generated "+path+" with 2 option(s)\n", out)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(content), "func WithName[")
	require.Contains(t, string(content), "func WithRetries(")
}

func TestGenerate_OptionPrefixAndOutput(t *testing.T) {
	t.Parallel()
	dir, _ := runGenerate(t, "--option-prefix", "Client", "--output", "client_gen.go")

	content, err := os.ReadFile(filepath.Join(dir, "client_gen.go"))
	require.NoError(t, err)
	require.Contains(t, string(content), "func WithClientName[")
	require.Contains(t, string(content), "func WithClientRetries(")
}

func TestGenerate_DryRunVerbose(t *testing.T) {
	t.Parallel()
	dir, out := runGenerate(t, "--dry-run", "--verbose")

	require.NoFileExists(t, filepath.Join(dir, "options_gen.go"))
	require.Contains(t, out, "Parsing directory: "+dir+"\n")
	require.Contains(t, out, "Looking for type: options\n")
	require.Contains(t, out, "Found 1 package(s)\n")
	require.Contains(t, out, "Processing package: sample\n")
	require.Contains(t, out, "Found 2 field(s) with opt tags:\n")
	require.Contains(t, out, "  - name (string) -> WithName\n")
	require.Contains(t, out, "Generating code...\n")
	require.Contains(t, out, "DRY RUN: Would generate "+filepath.Join(dir, "options_gen.go")+" with 2 option(s)\n")
	require.Contains(t, out, "--- Generated content preview (first 1000 bytes) ---\n")
	require.Contains(t, out, "(truncated, total size: ")
	require.Contains(t, out, "--- End preview ---\n")
}

func TestGenerate_NoTaggedStruct(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "x.go"), []byte("package sample\n\ntype other struct{}\n"), 0o600))

	cmd := optgen.NewGenerate()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--directory", dir, "--no-config", "--no-format"})
	require.ErrorContains(t, cmd.ExecuteContext(t.Context()), "no struct with name 'options' and opt tags found")
}

func TestGenerate_NestedModuleConfig(t *testing.T) {
	t.Parallel()
	for _, pkg := range []string{"providers/s3", "providers/s3/client"} {
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			dir := filepath.Join(root, filepath.FromSlash(pkg))
			require.NoError(t, os.MkdirAll(dir, 0o750))
			files := map[string]string{
				".git":                "gitdir: /unused/worktree\n",
				"go.mod":              "module example.com/root\n",
				"providers/s3/go.mod": "module example.com/root/providers/s3\n",
				pkg + "/options.go":   "package sample\n\ntype options struct { retries int }\n",
				".optgen.yaml": "defaults:\n  all-fields: true\n  no-format: true\n" +
					"packages:\n  " + pkg + ":\n    option-prefix: Root\n    output: root_gen.go\n",
			}
			for name, content := range files {
				require.NoError(t, os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(content), 0o600))
			}

			out, err := execGenerate(t, "--directory", dir, "--verbose")
			require.NoError(t, err)
			require.Contains(t, out, "Loaded config from: "+filepath.Join(root, ".optgen.yaml"))
			content, err := os.ReadFile(filepath.Join(dir, "root_gen.go"))
			require.NoError(t, err)
			require.Contains(t, string(content), "func WithRootRetries(")
			require.NotContains(t, string(content), "func WithRetries(")
			require.NoFileExists(t, filepath.Join(dir, "options_gen.go"))
		})
	}
}
