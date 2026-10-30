// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package optgen_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerationSessionsAreIsolated(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, sampleSource)
	_, err := execGenerate(t, "--directory", dir, "--no-config", "--no-format", "--disable-plugins=DefaultSetterPlugin")
	require.Error(t, err)
	_, err = execGenerate(t, "--directory", dir, "--no-config", "--no-format")
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, "options_gen.go"))
}

func TestDisableCheckPlugin(t *testing.T) {
	t.Parallel()
	dir := newPkgDir(t, "package sample\ntype options struct { count int `opt:\"Count\" optcheck:\"nonzero\"` }\n")
	_, err := execGenerate(t, "--directory", dir, "--no-config", "--no-format", "--disable-plugins=NonZeroCheck")
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(dir, "options_gen.go"))
	require.NoError(t, err)
	require.NotContains(t, string(data), "must be non-zero")
	_, err = execGenerate(t, "--directory", dir, "--no-config", "--no-format")
	require.NoError(t, err)
	data, err = os.ReadFile(filepath.Join(dir, "options_gen.go"))
	require.NoError(t, err)
	require.Contains(t, string(data), "must be non-zero")
}
