// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin/defaults"
	"github.com/altessa-s/go-optgen/plugin"
)

func TestTypeDefaultPlugins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		p        plugin.TypeDefaultPlugin
		typ      string
		other    string
		wantDef  string
		priority int
	}{
		{"logger", &defaults.LoggerDefaultPlugin{}, "*slog.Logger", "slog.Logger", "slog.New(slog.DiscardHandler)", 0},
		{"map", &defaults.MapDefaultPlugin{}, "map[string]int", "[]string", "make(map[string]int)", 0},
		{"configured", defaults.NewConfigured(map[string]string{"pkg.Client": "pkg.NewClient()"}), "pkg.Client", "*pkg.Client", "pkg.NewClient()", 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.True(t, tc.p.CanProvideDefault(tc.typ))
			require.False(t, tc.p.CanProvideDefault(tc.other))
			require.Equal(t, tc.wantDef, tc.p.GetDefault(tc.typ))
			meta := tc.p.Meta()
			require.Equal(t, plugin.KindTypeDefault, meta.Kind)
			require.Equal(t, tc.priority, meta.Priority)
		})
	}
}

func TestTypeDefaultRegistered(t *testing.T) {
	t.Parallel()

	require.Equal(t, "slog.New(slog.DiscardHandler)", testRegistry().FindTypeDefaultPlugin("*slog.Logger"))
	require.Equal(t, "make(map[int]bool)", testRegistry().FindTypeDefaultPlugin("map[int]bool"))
	require.Empty(t, testRegistry().FindTypeDefaultPlugin("serializer.Serializer"), "no default for types of a specific project")
}

func testRegistry() *plugin.Registry {
	r := plugin.NewRegistry()
	if err := defaults.Register(r); err != nil {
		panic(err)
	}
	return r
}
