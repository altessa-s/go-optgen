// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package postprocess

import (
	"cmp"
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// CloneModifier copies slice or map values at option creation or application time.
type CloneModifier struct{}

// Meta returns modifier metadata.
func (m *CloneModifier) Meta() plugin.Meta {
	return plugin.Meta{Kind: plugin.KindModifier, Name: "Clone", ApplicableTypes: []string{"[]", "map["}}
}

// Key returns the optval modifier key.
func (m *CloneModifier) Key() string { return "clone" }

// Phase returns the final collection-copy phase.
func (m *CloneModifier) Phase() plugin.Phase { return plugin.PhaseClone }

// CanHandle reports whether the field is a slice or map.
func (m *CloneModifier) CanHandle(field model.OptField) bool { return field.IsSlice || field.IsMap }

// Generate renders a closure-time copy unless the pipeline already made a fresh collection.
func (m *CloneModifier) Generate(ctx plugin.GenerationContext, field model.OptField, inputVar string) plugin.ModifierResult {
	if hasModifier(field, "clone=eager") {
		snapshotVar := ctx.InputVar
		if snapshotVar == "" {
			snapshotVar = inputVar
		}

		if field.IsSlice {
			pkg := ctx.AddStdlibImport("slices", "slices")
			return plugin.ModifierResult{OutputVar: inputVar, PreClosureCode: "\t" + snapshotVar + " = " + pkg + ".Clone(" + snapshotVar + ")\n"}
		}

		pkg := ctx.AddStdlibImport("maps", "maps")

		return plugin.ModifierResult{OutputVar: inputVar, PreClosureCode: "\t" + snapshotVar + " = " + pkg + ".Clone(" + snapshotVar + ")\n"}
	}

	if ctx.InputIsNewCollection {
		return plugin.ModifierResult{OutputVar: inputVar}
	}

	if inputVar != cmp.Or(ctx.InputVar, "v") {
		return plugin.ModifierResult{OutputVar: inputVar}
	}

	used := plugin.BodyIdentifiers(ctx, field, field.Type, field.ElemType, field.MapKeyType, field.MapValueType, inputVar)
	cloneVar := plugin.UniqueIdentifier("cloned", used)

	if field.IsSlice {
		pkg := ctx.AddStdlibImport("slices", "slices")

		return plugin.ModifierResult{Code: []string{cloneVar + " := " + pkg + ".Clone(" + inputVar + ")"}, OutputVar: cloneVar}
	}

	pkg := ctx.AddStdlibImport("maps", "maps")

	return plugin.ModifierResult{Code: []string{cloneVar + " := " + pkg + ".Clone(" + inputVar + ")"}, OutputVar: cloneVar}
}

func hasModifier(field model.OptField, want string) bool {
	for _, modifier := range field.Modifiers {
		if strings.EqualFold(modifier, want) {
			return true
		}
	}

	return false
}
