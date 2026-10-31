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
	tests := []struct {
		name       string
		files      []string
		gitDir     bool
		wantConfig string
		wantRoot   string
	}{
		{
			name:   "root config across nested module",
			files:  []string{"repo/go.mod", "repo/nested/go.mod", "repo/.optgen.yaml"},
			gitDir: true, wantConfig: "repo/.optgen.yaml", wantRoot: "repo",
		},
		{
			name:   "nested config wins",
			files:  []string{"repo/go.mod", "repo/nested/go.mod", "repo/.optgen.yaml", "repo/nested/.optgen.yaml"},
			gitDir: true, wantConfig: "repo/nested/.optgen.yaml", wantRoot: "repo/nested",
		},
		{
			name:       "worktree git file",
			files:      []string{"repo/.git", "repo/go.mod", "repo/nested/go.mod", "repo/.optgen.yaml"},
			wantConfig: "repo/.optgen.yaml", wantRoot: "repo",
		},
		{
			name:     "no git stops at nested module",
			files:    []string{"repo/go.mod", "repo/nested/go.mod", "repo/.optgen.yaml"},
			wantRoot: "repo/nested",
		},
		{
			name:       "no git includes module config",
			files:      []string{"repo/nested/go.mod", "repo/nested/.optgen.yaml"},
			wantConfig: "repo/nested/.optgen.yaml", wantRoot: "repo/nested",
		},
		{
			name:   "config above git ignored",
			files:  []string{".optgen.yaml", "go.mod"},
			gitDir: true, wantRoot: "repo",
		},
		{
			name:   "no config uses nearest module",
			files:  []string{"repo/go.mod", "repo/nested/go.mod"},
			gitDir: true, wantRoot: "repo/nested",
		},
		{
			name:     "no config uses git file",
			files:    []string{"repo/.git", ".optgen.yaml"},
			wantRoot: "repo",
		},
		{
			name:       "config below module defines root",
			files:      []string{"repo/go.mod", "repo/nested/.optgen.yaml"},
			wantConfig: "repo/nested/.optgen.yaml", wantRoot: "repo/nested",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for _, file := range tc.files {
				writeConfigTestFile(t, filepath.Join(root, file), "")
			}
			if tc.gitDir {
				require.NoError(t, os.MkdirAll(filepath.Join(root, "repo", ".git"), 0o750))
			}
			start := filepath.Join(root, "repo", "nested", "pkg")
			require.NoError(t, os.MkdirAll(start, 0o750))

			cfgPath, projectRoot := findConfigFile(start)
			if tc.wantConfig == "" {
				require.Empty(t, cfgPath)
			} else {
				require.Equal(t, filepath.Join(root, tc.wantConfig), cfgPath)
			}
			require.Equal(t, filepath.Join(root, tc.wantRoot), projectRoot)
		})
	}
}
