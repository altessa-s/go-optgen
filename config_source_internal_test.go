// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package optgen

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	optconfig "github.com/altessa-s/go-optgen/config"
)

func writeConfigTestFile(tb testing.TB, path, content string) {
	tb.Helper()
	require.NoError(tb, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(tb, os.WriteFile(path, []byte(content), 0o600))
}

func TestReadConfigFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := readConfigFile(filepath.Join(dir, "missing.yaml"))
	require.ErrorIs(t, err, fs.ErrNotExist)

	path := filepath.Join(dir, optconfig.ConfigFileName)
	writeConfigTestFile(t, path, "defaults:\n  type: settings\n")
	cfg, err := readConfigFile(path)
	require.NoError(t, err)
	require.Equal(t, "settings", cfg.Defaults.Type)
}

func TestFindConfigFile(t *testing.T) {
	t.Parallel()

	t.Run("config found in ancestor", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeConfigTestFile(t, filepath.Join(root, "go.mod"), "module x\n")
		writeConfigTestFile(t, filepath.Join(root, optconfig.ConfigFileName), "defaults: {}\n")
		deep := filepath.Join(root, "a", "b")
		require.NoError(t, os.MkdirAll(deep, 0o750))

		cfgPath, projectRoot := findConfigFile(deep)
		require.Equal(t, filepath.Join(root, optconfig.ConfigFileName), cfgPath)
		require.Equal(t, root, projectRoot)
	})

	t.Run("project root via go.mod without config", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeConfigTestFile(t, filepath.Join(root, "go.mod"), "module x\n")
		sub := filepath.Join(root, "sub")
		require.NoError(t, os.MkdirAll(sub, 0o750))

		cfgPath, projectRoot := findConfigFile(sub)
		require.Empty(t, cfgPath)
		require.Equal(t, root, projectRoot)
	})

	t.Run("project root via .git", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o750))

		cfgPath, projectRoot := findConfigFile(root)
		require.Empty(t, cfgPath)
		require.Equal(t, root, projectRoot)
	})
}
