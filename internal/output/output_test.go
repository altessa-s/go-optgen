// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package output_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/output"
)

func TestFailedFormattingPreservesExistingOutput(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "options_gen.go")
	old := []byte("previous output")
	require.NoError(t, os.WriteFile(path, old, 0o600))
	w := output.New(io.Discard, io.Discard, nil)
	require.Error(t, w.Write(t.Context(), path, []byte("package sample"), "gofmt --invalid-flag"))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, old, data)
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestCanceledWriteDoesNotPublish(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	path := filepath.Join(t.TempDir(), "options_gen.go")
	require.ErrorIs(t, output.New(io.Discard, io.Discard, nil).Write(ctx, path, nil, ""), context.Canceled)
	require.NoFileExists(t, path)
}

func TestWritePublishesFormattedOutput(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("gofmt"); err != nil {
		t.Skip("gofmt not on PATH")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "options_gen.go")
	var diagnostics bytes.Buffer
	w := output.New(io.Discard, io.Discard, &diagnostics)
	require.NoError(t, w.Write(t.Context(), path, []byte("package sample\nvar  x=1\n"), `gofmt -w "{file}"`))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "package sample\n\nvar x = 1\n", string(data))
	require.Contains(t, diagnostics.String(), "Formatting with: gofmt -w "+path)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())

	require.NoError(t, w.Write(t.Context(), path, []byte("unformatted"), ""), "an empty formatter publishes as is")
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "unformatted", string(data))
}

func TestWriteValidatesFormatter(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "options_gen.go")
	for _, formatter := range []string{"gofmt -w; rm x", "cat", "./gofmt", `C:\bin\gofmt`, "mytool"} {
		err := output.New(io.Discard, io.Discard, nil).Write(t.Context(), path, nil, formatter)
		require.ErrorContains(t, err, "invalid formatter", formatter)
	}
	require.NoFileExists(t, path)

	err := output.New(io.Discard, io.Discard, nil, output.WithAllowedFormatters(" mytool ")).Write(t.Context(), path, nil, "mytool -w")
	require.Error(t, err, "an allowed but missing command fails at run time")
	require.NotContains(t, err.Error(), "invalid formatter")
}

func TestWriteFailsForMissingDirectory(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "missing", "options_gen.go")
	require.Error(t, output.New(io.Discard, io.Discard, nil).Write(t.Context(), path, nil, ""))
}
