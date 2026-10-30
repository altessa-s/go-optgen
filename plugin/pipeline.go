// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package plugin

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/altessa-s/go-optgen/model"
)

// Pipeline produces the value-processing code consumed by field renderers.
type Pipeline interface {
	BuildPipeline(GenerationContext, model.OptField, string) PipelineResult
}

// ValuePlugins provides only the queries needed by pipeline execution.
type ValuePlugins interface {
	CollectModifiers(model.OptField) []ModifierPlugin
	FindTransformPlugin(string, string) (TransformPlugin, bool)
	FindCheckPlugin(string) (CheckPlugin, CheckSpec, error)
}

// Executor runs value plugins without owning registration or configuration.
type Executor struct{ plugins ValuePlugins }

// NewPipeline binds a value catalog to an executor.
func NewPipeline(plugins ValuePlugins) *Executor { return &Executor{plugins: plugins} }

// BuildPipeline executes all phases once, propagating values and termination.
// Final stops code generation; conditional runtime guards must leave it false.
func (r *Executor) BuildPipeline(ctx GenerationContext, field model.OptField, inputVar string) PipelineResult {
	mods := r.plugins.CollectModifiers(field)
	var lines []string
	value := inputVar
	finish := func(final bool) PipelineResult {
		return PipelineResult{Code: strings.Join(lines, "\n"), ValueVar: value, EarlyReturn: final}
	}
	for _, phase := range []Phase{PhaseGuard, PhaseTransform, PhasePostProcess, PhaseCheck} {
		if phase == PhaseTransform {
			result := r.transformValue(ctx, field, value)
			lines = append(lines, result.Code...)
			value = result.OutputVar
		}
		for _, mod := range mods {
			if mod.Phase() != phase {
				continue
			}
			result := mod.Generate(ctx, field, value)
			if result.Final {
				lines = append(lines, "_ = "+value)
			}
			lines = append(lines, result.Code...)
			if result.OutputVar != "" {
				value = result.OutputVar
			}
			if result.Final {
				return finish(true)
			}
		}
		if phase == PhaseCheck {
			lines = append(lines, r.checkValue(ctx, field, value)...)
		}
	}
	return finish(false)
}

func (r *Executor) transformValue(ctx GenerationContext, field model.OptField, value string) ModifierResult {
	typeName := field.Type
	expr := value
	if field.IsSlice {
		typeName = field.ElemType
		expr = "element"
	}
	transformed := expr
	for _, key := range field.Modifiers {
		p, ok := r.plugins.FindTransformPlugin(key, typeName)
		if !ok {
			continue
		}
		stepField := field
		stepField.Modifiers = []string{key}
		if next, ok := p.Apply(ctx, stepField, typeName, transformed); ok {
			transformed = next
		}
	}
	if transformed == expr {
		return ModifierResult{OutputVar: value}
	}
	if field.IsSlice {
		return ModifierResult{Code: []string{
			"transformed := make([]" + field.ElemType + ", len(" + value + "))",
			"for i, element := range " + value + " {",
			"  transformed[i] = " + transformed,
			"}",
		}, OutputVar: "transformed"}
	}
	// Materialize expressions so later modifiers receive an assignable value.
	return ModifierResult{Code: []string{"transformedValue := " + transformed}, OutputVar: "transformedValue"}
}

// checkValue renders the field's checks by priority, then key. Declarations
// are validated before generation; keys without an enabled plugin are skipped.
func (r *Executor) checkValue(ctx GenerationContext, field model.OptField, value string) []string {
	type check struct {
		key    string
		plugin CheckPlugin
		spec   CheckSpec
	}
	var checks []check
	for _, key := range slices.Sorted(maps.Keys(field.Checks)) {
		if p, spec, err := r.plugins.FindCheckPlugin(key); err == nil {
			checks = append(checks, check{key: key, plugin: p, spec: spec})
		}
	}
	slices.SortStableFunc(checks, func(a, b check) int { return cmp.Compare(a.spec.Priority, b.spec.Priority) })
	lines := make([]string, 0, 3*len(checks))
	for _, c := range checks {
		lines = append(lines, c.plugin.Generate(ctx, field, value, field.Checks[c.key])...)
	}
	return lines
}
