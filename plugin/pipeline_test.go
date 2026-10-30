// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package plugin_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

func TestPipelinePropagatesValuesAndStopsInEveryPhase(t *testing.T) {
	t.Parallel()
	for _, finalPhase := range []plugin.Phase{0, plugin.PhaseGuard, plugin.PhaseTransform, plugin.PhasePostProcess, plugin.PhaseCheck} {
		t.Run(finalPhase.String(), func(t *testing.T) {
			t.Parallel()
			r := plugin.NewRegistry()
			var calls []string
			field := model.OptField{Modifiers: []string{"guard", "transform", "postprocess", "check"}}
			for _, phase := range []plugin.Phase{plugin.PhaseGuard, plugin.PhaseTransform, plugin.PhasePostProcess, plugin.PhaseCheck} {
				require.NoError(t, r.Register(&mockModifierPlugin{key: phase.String(), phase: phase, canHandle: func(model.OptField) bool { return true },
					generate: func(_ plugin.GenerationContext, _ model.OptField, in string) plugin.ModifierResult {
						calls = append(calls, in)
						return plugin.ModifierResult{Code: []string{phase.String()}, OutputVar: in + "x", Final: phase == finalPhase}
					}}))
			}
			result := plugin.NewPipeline(r).BuildPipeline(plugin.GenerationContext{}, field, "v")
			count := 4
			if finalPhase != 0 {
				count = int(finalPhase) / 10
			}
			require.Len(t, calls, count)
			require.Equal(t, finalPhase != 0, result.EarlyReturn)
			value := "v"
			for _, in := range calls {
				require.Equal(t, value, in)
				value += "x"
			}
			require.Equal(t, value, result.ValueVar)
		})
	}
}

func TestPipelineRunsEnabledChecksByPriority(t *testing.T) {
	t.Parallel()
	r := plugin.NewRegistry()
	require.NoError(t, r.Register(
		&mockCheckPlugin{spec: plugin.CheckSpec{Key: "late", Priority: 20}},
		&mockCheckPlugin{spec: plugin.CheckSpec{Key: "early", Priority: 10}},
		&mockCheckPlugin{spec: plugin.CheckSpec{Key: "off", Priority: 0}},
	))
	require.NoError(t, r.DisablePlugins("off"))
	field := model.OptField{Checks: map[string]string{"late": "2", "early": "1", "off": "", "unknown": ""}}
	result := plugin.NewPipeline(r).BuildPipeline(plugin.GenerationContext{}, field, "v")
	require.Equal(t, "early\nv\n1\nlate\nv\n2", result.Code)
}

func TestValueKindOf(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		field model.OptField
		want  plugin.ValueKind
	}{
		{model.OptField{Type: "[]string", IsSlice: true}, plugin.ValueSlice},
		{model.OptField{Type: "string"}, plugin.ValueString},
		{model.OptField{Type: "*T"}, plugin.ValuePointer},
		{model.OptField{Type: "map[string]int"}, plugin.ValueMap},
		{model.OptField{Type: "any", IsInterface: true}, plugin.ValueInterface},
		{model.OptField{Type: "io.Writer"}, plugin.ValueOther},
	} {
		require.Equal(t, tc.want, plugin.ValueKindOf(tc.field), tc.field.Type)
	}
}

func TestBuildNilCheck(t *testing.T) {
	t.Parallel()
	ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "T", OptionType: "Option", OptionReturnsError: true}, nil, nil, model.GenericInfo{})
	require.Equal(t, []string{
		"switch rv := reflect.ValueOf(v); rv.Kind() {",
		"case reflect.Invalid:",
		"  return nil",
		"case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:",
		"  if rv.IsNil() {",
		"    return nil",
		"  }",
		"}",
	}, plugin.BuildNilEarlyReturn(ctx, "v"))
	require.Equal(t, []model.GeneratedImport{{Path: "reflect"}}, ctx.GetAdditionalImports())
}
