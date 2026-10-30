// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package transform_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin/transform"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

func newCtx() plugin.GenerationContext {
	return plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option", OptionReturnsError: false}, nil, nil, model.GenericInfo{})
}

func mustFind(tb testing.TB, key string) plugin.TransformPlugin {
	tb.Helper()
	p, ok := testRegistry().FindTransformPlugin(key, "string")
	require.True(tb, ok, key)
	return p
}

func TestSimpleStringTransforms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key  string
		want string
	}{
		{"lower", "strings.ToLower(v)"},
		{"upper", "strings.ToUpper(v)"},
		{"trimspaces", "strings.TrimSpace(v)"},
	}
	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()
			p := mustFind(t, tc.key)
			require.Equal(t, tc.key, p.Key())
			require.Equal(t, []string{"string"}, p.Meta().ApplicableTypes)

			ctx := newCtx()
			got, ok := p.Apply(ctx, model.OptField{}, "string", "v")
			require.True(t, ok)
			require.Equal(t, tc.want, got)
			imps := ctx.GetAdditionalImports()
			require.Len(t, imps, 1)
			require.Equal(t, "strings", imps[0].Path)
		})
	}
}

func TestTrimSpacesDefaults(t *testing.T) {
	t.Parallel()

	meta := mustFind(t, "trimspaces").Meta()
	require.Equal(t, []string{"string"}, meta.DefaultForTypes)
	require.Equal(t, "notrim", meta.DisabledBy)
}

func TestTrimPrefixSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key  string
		mods []string
		want string
		ok   bool
	}{
		{"trimprefix", []string{"upper", "trimprefix=api/"}, `strings.TrimPrefix(v, "api/")`, true},
		{"trimprefix", []string{"trimprefix=a\"b\\c\n\r\td"}, `strings.TrimPrefix(v, "a\"b\\c\n\r\td")`, true},
		{"trimprefix", []string{"upper"}, "v", false},
		{"trimprefix", []string{"trimprefix="}, "v", false},
		{"trimsuffix", []string{"trimsuffix=.json"}, `strings.TrimSuffix(v, ".json")`, true},
		{"trimsuffix", []string{"trimsuffix=\t\""}, `strings.TrimSuffix(v, "\t\"")`, true},
		{"trimsuffix", nil, "v", false},
	}
	for _, tc := range tests {
		t.Run(tc.key+"/"+tc.want, func(t *testing.T) {
			t.Parallel()
			ctx := newCtx()
			got, ok := mustFind(t, tc.key).Apply(ctx, model.OptField{Modifiers: tc.mods}, "string", "v")
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.want, got)
			if !tc.ok {
				require.Empty(t, ctx.GetAdditionalImports())
			}
		})
	}
}

func testRegistry() *plugin.Registry { r := plugin.NewRegistry(); transform.Register(r); return r }
