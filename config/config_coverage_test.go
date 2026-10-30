// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package config_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/config"
)

func writeFile(tb testing.TB, path, content string) {
	tb.Helper()
	require.NoError(tb, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(tb, os.WriteFile(path, []byte(content), 0o600))
}

func TestLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, config.ConfigFileName)
	writeFile(t, path, `
defaults:
  type: options
  output: options_gen.go
  option-error: true
  all-fields: true
  disable-plugins: [dedup]
  formatter: "goimports -w"
packages:
  pkg/a:
    type: settings
`)

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, "options", cfg.Defaults.Type)
	require.Equal(t, "options_gen.go", cfg.Defaults.Output)
	require.True(t, cfg.Defaults.OptionError)
	require.True(t, cfg.Defaults.AllFields)
	require.Equal(t, []string{"dedup"}, cfg.Defaults.DisablePlugins)
	require.Equal(t, new("goimports -w"), cfg.Defaults.Formatter)
	require.Equal(t, "settings", cfg.Packages["pkg/a"].Type)
}

func TestLoadErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, err := config.Load(filepath.Join(dir, "missing.yaml"))
	require.ErrorIs(t, err, fs.ErrNotExist)

	bad := filepath.Join(dir, "bad.yaml")
	writeFile(t, bad, "defaults: [not, a, map]\n")
	_, err = config.Load(bad)
	require.Error(t, err)

	plugins := filepath.Join(dir, "plugins.yaml")
	writeFile(t, plugins, "defaults:\n  plugins: [x.so]\n")
	_, err = config.Load(plugins)
	require.ErrorIs(t, err, config.ErrPluginsUnsupported)
}

func TestFindConfigFile(t *testing.T) {
	t.Parallel()

	t.Run("config found in ancestor", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "go.mod"), "module x\n")
		writeFile(t, filepath.Join(root, config.ConfigFileName), "defaults: {}\n")
		deep := filepath.Join(root, "a", "b")
		require.NoError(t, os.MkdirAll(deep, 0o750))

		cfgPath, projectRoot := config.FindConfigFile(deep)
		require.Equal(t, filepath.Join(root, config.ConfigFileName), cfgPath)
		require.Equal(t, root, projectRoot)
	})

	t.Run("project root via go.mod without config", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "go.mod"), "module x\n")
		sub := filepath.Join(root, "sub")
		require.NoError(t, os.MkdirAll(sub, 0o750))

		cfgPath, projectRoot := config.FindConfigFile(sub)
		require.Empty(t, cfgPath)
		require.Equal(t, root, projectRoot)
	})

	t.Run("project root via .git", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o750))

		cfgPath, projectRoot := config.FindConfigFile(root)
		require.Empty(t, cfgPath)
		require.Equal(t, root, projectRoot)
	})
}

func TestGetPackageConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.File{
		Defaults: config.Defaults{
			Type:           "options",
			Output:         "options_gen.go",
			OptionType:     "Option",
			Formatter:      new("gofmt -w"),
			DisablePlugins: []string{"x"},
		},
		Packages: map[string]config.PackageConfig{
			"pkg/full": {
				Type:            "settings",
				Output:          "settings_gen.go",
				OptionType:      "Setting",
				OptionError:     new(true),
				AllFields:       new(true),
				SkipOptionType:  new(true),
				SkipDefaultFunc: new(true),
				SkipNewFunc:     new(true),
				DefaultFuncName: "defaults",
				NewFuncName:     "build",
				OptionPrefix:    "Use",
				DisablePlugins:  []string{"dedup", "positive"},
				Formatter:       new(""),
				NoFormat:        new(true),
			},
			"pkg/partial": {Output: "p_gen.go"},
		},
	}

	t.Run("no override returns defaults", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, cfg.Defaults, cfg.GetPackageConfig("other"))
	})

	t.Run("full override with path cleaning", func(t *testing.T) {
		t.Parallel()
		got := cfg.GetPackageConfig("./pkg/full/")
		require.Equal(t, config.Defaults{
			Type:            "settings",
			Output:          "settings_gen.go",
			OptionType:      "Setting",
			OptionError:     true,
			AllFields:       true,
			SkipOptionType:  true,
			SkipDefaultFunc: true,
			SkipNewFunc:     true,
			DefaultFuncName: "defaults",
			NewFuncName:     "build",
			OptionPrefix:    "Use",
			DisablePlugins:  []string{"dedup", "positive"},
			Formatter:       new(""),
			NoFormat:        true,
		}, got)
	})

	t.Run("partial override inherits the rest", func(t *testing.T) {
		t.Parallel()
		got := cfg.GetPackageConfig("pkg/partial")
		require.Equal(t, "p_gen.go", got.Output)
		require.Equal(t, "options", got.Type)
		require.Equal(t, new("gofmt -w"), got.Formatter)
		require.Equal(t, []string{"x"}, got.DisablePlugins)
	})
}
