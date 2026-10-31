// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/config"
)

func TestParse(t *testing.T) {
	t.Parallel()

	cfg, err := config.Parse([]byte(`
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
`))
	require.NoError(t, err)
	require.Equal(t, "options", cfg.Defaults.Type)
	require.Equal(t, "options_gen.go", cfg.Defaults.Output)
	require.True(t, cfg.Defaults.OptionError)
	require.True(t, cfg.Defaults.AllFields)
	require.Equal(t, []string{"dedup"}, cfg.Defaults.DisablePlugins)
	require.Equal(t, new("goimports -w"), cfg.Defaults.Formatter)
	require.Equal(t, "settings", cfg.Packages["pkg/a"].Type)
}

func TestParseErrors(t *testing.T) {
	t.Parallel()

	_, err := config.Parse([]byte("defaults: [not, a, map]\n"))
	require.Error(t, err)

	_, err = config.Parse([]byte("defaults:\n  plugins: [x.so]\n"))
	require.ErrorIs(t, err, config.ErrPluginsUnsupported)
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
