// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package postprocess

import (
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// NonEmptyModifier enforces/normalizes "nonempty" semantics.
//
// Tag usage: optval:"nonempty"
//
// Behavior:
//   - string: skips assignment if empty (early return, keeps default/current value)
//   - []string: drops empty elements into a new slice (after any prior transforms),
//     so the caller's variadic argument is never modified
//
// Rationale: this runs after transformations so trimspaces/lower/etc happen first.
type NonEmptyModifier struct{}

// Meta returns plugin metadata.
func (m *NonEmptyModifier) Meta() plugin.Meta {
	return plugin.Meta{Kind: plugin.KindModifier}
}

// Key returns the modifier name used in tags.
func (m *NonEmptyModifier) Key() string { return "nonempty" }

// Phase returns PhasePostProcess - runs after transforms.
func (m *NonEmptyModifier) Phase() plugin.Phase { return plugin.PhasePostProcess }

// CanHandle returns true for string and []string fields.
func (m *NonEmptyModifier) CanHandle(field model.OptField) bool {
	return field.Type == kindString || (field.IsSlice && field.ElemType == kindString)
}

// Generate produces non-empty validation/filtering code.
func (m *NonEmptyModifier) Generate(ctx plugin.GenerationContext, field model.OptField, inputVar string) plugin.ModifierResult {
	if inputVar == "" {
		inputVar = "v"
	}

	// Scalar string: skip assignment if empty (early return).
	if field.Type == kindString {
		return plugin.ModifierResult{
			Code:      plugin.BuildEarlyReturn(ctx, inputVar+` == ""`),
			OutputVar: inputVar,
		}
	}

	// []string: drop empty entries into a new slice, which becomes the output.
	// The input is never reassigned: it is the option's captured argument, and
	// an Option value may be applied many times, concurrently.
	if field.IsSlice && field.ElemType == kindString {
		tmp := "nonEmpty"
		if strings.EqualFold(inputVar, tmp) {
			tmp = "nonEmpty2"
		}
		code := []string{
			tmp + " := make([]string, 0, len(" + inputVar + "))",
			"for _, x := range " + inputVar + " {",
			`  if x != "" {`,
			"    " + tmp + " = append(" + tmp + ", x)",
			"  }",
			"}",
		}
		return plugin.ModifierResult{Code: code, OutputVar: tmp}
	}

	return plugin.ModifierResult{OutputVar: inputVar}
}
