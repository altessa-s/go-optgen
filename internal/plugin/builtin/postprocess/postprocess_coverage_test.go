// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package postprocess_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin/postprocess"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

func newCtx(returnsErr bool) plugin.GenerationContext {
	return plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option", OptionReturnsError: returnsErr}, nil, nil, model.GenericInfo{})
}

func TestDedupModifier(t *testing.T) {
	t.Parallel()

	m := &postprocess.DedupModifier{}
	require.Equal(t, "dedup", m.Key())
	require.Equal(t, plugin.PhasePostProcess, m.Phase())
	require.Equal(t, "nodup", m.Meta().DisabledBy)

	tests := []struct {
		name  string
		field model.OptField
		want  bool
	}{
		{"not a slice", model.OptField{Type: "string"}, false},
		{"string elems", model.OptField{IsSlice: true, ElemType: "string"}, true},
		{"pointer elems", model.OptField{IsSlice: true, ElemType: "*Foo"}, true},
		{"int64 elems", model.OptField{IsSlice: true, ElemType: "int64"}, true},
		{"duration elems", model.OptField{IsSlice: true, ElemType: "time.Duration"}, true},
		{"time elems", model.OptField{IsSlice: true, ElemType: "time.Time"}, true},
		{"struct elems", model.OptField{IsSlice: true, ElemType: "Foo"}, false},
		{"slice elems", model.OptField{IsSlice: true, ElemType: "[]byte"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, m.CanHandle(tc.field))
		})
	}

	ctx := newCtx(false)
	res := m.Generate(ctx, model.OptField{ElemType: "string"}, "xs")
	require.Equal(t, []string{
		"{",
		"  seen := make(map[string]struct{}, len(xs))",
		"  deduped := make([]string, 0, len(xs))",
		"  for _, item := range xs {",
		"    if _, ok := seen[item]; ok {",
		"      continue",
		"    }",
		"    seen[item] = struct{}{}",
		"    deduped = append(deduped, item)",
		"  }",
		"  xs = deduped",
		"}",
	}, res.Code)
	require.Equal(t, "xs", res.OutputVar)
	require.False(t, res.Final)
	require.Empty(t, ctx.GetAdditionalImports(), "deduplication needs no imports")
}

func TestNonEmptyModifier(t *testing.T) {
	t.Parallel()

	m := &postprocess.NonEmptyModifier{}
	require.Equal(t, "nonempty", m.Key())
	require.Equal(t, plugin.PhasePostProcess, m.Phase())

	require.True(t, m.CanHandle(model.OptField{Type: "string"}))
	require.True(t, m.CanHandle(model.OptField{Type: "[]string", IsSlice: true, ElemType: "string"}))
	require.False(t, m.CanHandle(model.OptField{Type: "[]int", IsSlice: true, ElemType: "int"}))
	require.False(t, m.CanHandle(model.OptField{Type: "int"}))

	t.Run("scalar string early-returns", func(t *testing.T) {
		t.Parallel()
		res := m.Generate(newCtx(true), model.OptField{Type: "string"}, "")
		require.Equal(t, []string{`if v == "" {`, "  return nil", "}"}, res.Code)
		require.False(t, res.Final)
		require.Equal(t, "v", res.OutputVar)
	})

	t.Run("string slice filters in place", func(t *testing.T) {
		t.Parallel()
		field := model.OptField{Type: "[]string", IsSlice: true, ElemType: "string"}
		res := m.Generate(newCtx(false), field, "v")
		require.Equal(t, "nonEmpty := make([]string, 0, len(v))", res.Code[0])
		require.Equal(t, "v = nonEmpty", res.Code[len(res.Code)-1])
		require.False(t, res.Final)
	})

	t.Run("temp name avoids collision with input", func(t *testing.T) {
		t.Parallel()
		field := model.OptField{Type: "[]string", IsSlice: true, ElemType: "string"}
		res := m.Generate(newCtx(false), field, "nonEmpty")
		require.Equal(t, "nonEmpty2 := make([]string, 0, len(nonEmpty))", res.Code[0])
		require.Equal(t, "nonEmpty = nonEmpty2", res.Code[len(res.Code)-1])
	})

	t.Run("unsupported type is a no-op", func(t *testing.T) {
		t.Parallel()
		res := m.Generate(newCtx(false), model.OptField{Type: "int"}, "v")
		require.Empty(t, res.Code)
		require.Equal(t, "v", res.OutputVar)
	})
}

func TestPositiveModifier(t *testing.T) {
	t.Parallel()

	m := &postprocess.PositiveModifier{}
	require.Equal(t, "positive", m.Key())
	require.Equal(t, plugin.PhasePostProcess, m.Phase())
	require.Equal(t, "nonpositive", m.Meta().DisabledBy)
	require.Equal(t, []string{"time.Duration"}, m.Meta().DefaultForTypes)

	require.True(t, m.CanHandle(model.OptField{Type: "time.Duration"}))
	require.True(t, m.CanHandle(model.OptField{Type: "uint16"}))
	require.False(t, m.CanHandle(model.OptField{Type: "string"}))

	res := m.Generate(newCtx(false), model.OptField{Type: "int"}, "")
	require.Equal(t, []string{"if v <= 0 {", "  return", "}"}, res.Code)
	require.False(t, res.Final)

	res = m.Generate(newCtx(true), model.OptField{Type: "int", Modifiers: []string{"positive=allow_zero"}}, "n")
	require.Equal(t, []string{"if n < 0 {", "  return nil", "}"}, res.Code)
}
