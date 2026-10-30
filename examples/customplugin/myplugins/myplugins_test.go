// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package myplugins_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/examples/customplugin/myplugins"
	"github.com/altessa-s/go-optgen/plugin"

	optgen "github.com/altessa-s/go-optgen"
)

func TestRegisterAlongsideBuiltins(t *testing.T) {
	t.Parallel()
	cmd := optgen.New(optgen.WithPlugins(myplugins.Register))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"list-plugins"})
	require.NoError(t, cmd.ExecuteContext(t.Context()))
	require.Contains(t, out.String(), `Collapse (key="collapse")`)
	require.Contains(t, out.String(), `HostPort (key="hostport")`)
}

func TestHostPortRejectsNonStringField(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := "package sample\ntype options struct { port int `opt:\"Port\" optcheck:\"hostport\"` }\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module sample\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "options.go"), []byte(src), 0o600))
	cmd := optgen.New(optgen.WithPlugins(myplugins.Register))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"generate", "--directory", dir, "--no-config", "--no-format", "--dry-run"})
	require.ErrorIs(t, cmd.ExecuteContext(t.Context()), plugin.ErrNotApplicable)
}
