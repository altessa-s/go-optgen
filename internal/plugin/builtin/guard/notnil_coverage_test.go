// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package guard_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin/guard"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

func newCtx(returnsErr bool) plugin.GenerationContext {
	return plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option", OptionReturnsError: returnsErr}, nil, nil, model.GenericInfo{})
}

func TestNotNilModifierMeta(t *testing.T) {
	t.Parallel()

	m := &guard.NotNilModifier{}
	require.Equal(t, "notnil", m.Key())
	require.Equal(t, plugin.PhaseGuard, m.Phase())
	meta := m.Meta()
	require.Equal(t, "allownull", meta.DisabledBy)
	require.ElementsMatch(t, []string{"*", "interface", "[]"}, meta.DefaultForTypes)
}

func TestNotNilModifierCanHandle(t *testing.T) {
	t.Parallel()

	m := &guard.NotNilModifier{}
	tests := []struct {
		name  string
		field model.OptField
		want  bool
	}{
		{"explicit tag", model.OptField{Type: "int", Metadata: map[string]string{"notnil": ""}}, true},
		{"nilable", model.OptField{Type: "chan int", IsNilable: true}, true},
		{"interface", model.OptField{Type: "io.Reader", IsInterface: true}, true},
		{"pointer by type string", model.OptField{Type: "*Foo"}, true},
		{"slice", model.OptField{Type: "[]int", IsSlice: true}, true},
		{"map by type string", model.OptField{Type: "map[string]int"}, true},
		{"scalar", model.OptField{Type: "int"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, m.CanHandle(tc.field))
		})
	}
}

func TestNotNilModifierGenerate(t *testing.T) {
	t.Parallel()

	m := &guard.NotNilModifier{}

	t.Run("slice uses len and default var", func(t *testing.T) {
		t.Parallel()
		res := m.Generate(newCtx(false), model.OptField{IsSlice: true}, "")
		require.Equal(t, []string{"if len(v) == 0 {", "  return", "}"}, res.Code)
		require.Equal(t, "v", res.OutputVar)
		require.False(t, res.Final)
	})

	t.Run("pointer uses nil comparison", func(t *testing.T) {
		t.Parallel()
		ctx := newCtx(true)
		res := m.Generate(ctx, model.OptField{Type: "*Foo", IsNilable: true}, "p")
		require.Equal(t, []string{"if p == nil {", "  return nil", "}"}, res.Code)
		require.Empty(t, ctx.GetAdditionalImports())
	})

	t.Run("interface uses reflect nil check", func(t *testing.T) {
		t.Parallel()
		ctx := newCtx(false)
		res := m.Generate(ctx, model.OptField{Type: "any", IsInterface: true, IsNilable: true}, "v")
		require.Equal(t, plugin.BuildNilEarlyReturn(newCtx(false), "v"), res.Code)
		require.Equal(t, []model.GeneratedImport{{Path: "reflect"}}, ctx.GetAdditionalImports())
	})

	t.Run("explicit notnil on named type uses reflect nil check", func(t *testing.T) {
		t.Parallel()
		field := model.OptField{Type: "metrics.Collector", Metadata: map[string]string{"notnil": ""}}
		res := m.Generate(newCtx(true), field, "v")
		require.Equal(t, "switch rv := reflect.ValueOf(v); rv.Kind() {", res.Code[0])
		require.Equal(t, "  return nil", res.Code[2])
	})

	t.Run("explicit notnil on map stays nil comparison", func(t *testing.T) {
		t.Parallel()
		field := model.OptField{Type: "map[string]int", Metadata: map[string]string{"notnil": ""}}
		res := m.Generate(newCtx(false), field, "v")
		require.Equal(t, "if v == nil {", res.Code[0])
	})
}
