// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package optgen_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const featuresSource = `package sample

import (
	"io"
	"time"
)

type options struct {
	out      io.Writer     ` + "`opt:\"Out\" optgen:\"mandatory\"`" + `
	name     string        ` + "`opt:\"Name\" optcheck:\"required\"`" + `
	wait     time.Duration ` + "`opt:\"Wait\" optgen:\"default=1m30s\"`" + `
	hosts    []string      ` + "`opt:\"Hosts\"`" + `
	debugRaw int           ` + "`opt:\"DebugRaw\"`" + `
	set      optionsFieldSet
}

var defaultOptionsValue = options{name: "fallback"}
`

const featuresTest = `package sample

import (
	"os"
	"testing"
	"time"
)

func TestFeatures(t *testing.T) {
	o, err := newOptions(os.Stdout, WithHosts([]string{"a", "b"}))
	if err != nil {
		t.Fatal(err)
	}
	if o.out != os.Stdout || o.name != "fallback" || o.wait != 90*time.Second {
		t.Fatalf("unexpected options %+v", o)
	}
	if !o.isSet(optionsFieldHosts) || o.isSet(optionsFieldName) {
		t.Fatal("isSet does not track applied options")
	}
	if _, err := newOptions(os.Stdout, WithName("  ")); err == nil {
		t.Fatal("expected the final validation to reject an empty name")
	}
}
`

func TestGenerateFeaturesCompileAndRun(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	dir := newPkgDir(t, featuresSource)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module sample\n\ngo 1.26\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "options_test.go"), []byte(featuresTest), 0o600))
	_, err := execGenerate(t, "--directory", dir, "--no-config", "--no-format", "--option-error", "--validate", "--with-isset",
		"--defaults-from", "var", "--slice-setter", "slice", "--exclude", "debug*", "--source-file", "options.go")
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(dir, "options_gen.go"))
	require.NoError(t, err)
	require.NotContains(t, string(data), "DebugRaw", "excluded fields get no option")
	require.Contains(t, string(data), "func newOptions(out io.Writer, opts ...Option) (*options, error)")
	command := exec.CommandContext(t.Context(), "go", "test", "./...")
	command.Dir = dir
	out, err := command.CombinedOutput()
	require.NoError(t, err, "%s\n%s", out, data)
}

func TestGenerateStrictRejectsWarnings(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, "package sample\ntype options struct { Name string `opt:\"Name\"` }\n")
	out, err := execGenerate(t, "--directory", dir, "--no-config", "--no-format", "--strict")
	require.ErrorContains(t, err, "warnings reported in strict mode")
	require.Contains(t, out, "field Name is exported")
	require.NoFileExists(t, filepath.Join(dir, "options_gen.go"))

	_, err = execGenerate(t, "--directory", dir, "--no-config", "--no-format")
	require.NoError(t, err, "without --strict warnings do not fail")
}

func TestGenerateRejectsInvalidFeatureFlags(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, sampleSource)
	for _, args := range [][]string{
		{"--defaults-from", "file"},
		{"--defaults-from", "tag=x"},
		{"--defaults-from", "var=1x"},
		{"--slice-setter", "array"},
		{"--exclude", "[bad"},
		{"--with-isset"},
	} {
		_, err := execGenerate(t, append([]string{"--directory", dir, "--no-config", "--no-format", "--dry-run"}, args...)...)
		require.Error(t, err, args)
	}
}
