// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package postprocess

import (
	"slices"
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
	if field.IsSlice && ((hasModifier(field, "dedup") && (field.ElemComparable || isComparableType(field.ElemType))) ||
		field.ElemType == "string" && (hasModifier(field, "nonempty") || hasSliceTransform(field))) {
		return plugin.ModifierResult{OutputVar: inputVar}
	}

	if field.IsSlice {
		ctx.AddImport("slices")

		if hasModifier(field, "clone=eager") {
			return plugin.ModifierResult{OutputVar: inputVar}
		}

		return plugin.ModifierResult{Code: []string{"cloned := slices.Clone(" + inputVar + ")"}, OutputVar: "cloned"}
	}

	ctx.AddImport("maps")

	if hasModifier(field, "clone=eager") {
		return plugin.ModifierResult{OutputVar: inputVar}
	}

	return plugin.ModifierResult{Code: []string{"cloned := maps.Clone(" + inputVar + ")"}, OutputVar: "cloned"}
}

func hasSliceTransform(field model.OptField) bool {
	if field.ElemType != "string" {
		return false
	}

	for _, modifier := range field.Modifiers {
		key, _, _ := strings.Cut(modifier, "=")
		switch key {
		case "trimspaces", "lower", "upper", "trimprefix", "trimsuffix":
			return true
		}
	}

	return false
}

func hasModifier(field model.OptField, want string) bool {
	return slices.Contains(field.Modifiers, want)
}
