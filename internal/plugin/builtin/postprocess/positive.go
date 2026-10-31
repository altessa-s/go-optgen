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

const positivePriority = 5

// PositiveModifier enforces positive value semantics.
//
// Tag usage: optval:"positive" or optval:"positive=allow_zero"
//
// Behavior:
// - Without allow_zero: skips assignment if v <= 0 (only positive values allowed)
// - With allow_zero: skips assignment if v < 0 (zero and positive values allowed)
//
// Works with numeric types: int, int8, int16, int32, int64,
// uint, uint8, uint16, uint32, uint64, float32, float64, time.Duration.
//
// For time.Duration, this modifier is applied by default (can be disabled with optval:"nonpositive").
type PositiveModifier struct{}

// Meta returns plugin metadata.
func (m *PositiveModifier) Meta() plugin.Meta {
	return plugin.Meta{
		Kind:            plugin.KindModifier,
		Priority:        positivePriority,
		DefaultForTypes: []string{durationType},
		DisabledBy:      "nonpositive",
	}
}

// Key returns the modifier name used in tags.
func (m *PositiveModifier) Key() string { return "positive" }

// Phase returns PhasePostProcess - runs after transforms.
func (m *PositiveModifier) Phase() plugin.Phase { return plugin.PhasePostProcess }

// CanHandle returns true for numeric types.
func (m *PositiveModifier) CanHandle(field model.OptField) bool {
	return plugin.IsNumericField(field)
}

// Generate produces positive value validation code.
func (m *PositiveModifier) Generate(ctx plugin.GenerationContext, field model.OptField, inputVar string) plugin.ModifierResult {
	if inputVar == "" {
		inputVar = "v"
	}

	// Check if allow_zero is specified
	allowZero := slices.ContainsFunc(field.Modifiers, func(mod string) bool {
		return strings.EqualFold(mod, "positive=allow_zero")
	})

	var op string
	if allowZero {
		op = "<" // skip if v < 0 (allow zero and positive)
	} else {
		op = "<=" // skip if v <= 0 (only positive)
	}

	return plugin.ModifierResult{
		Code:      plugin.BuildEarlyReturn(ctx, inputVar+" "+op+" 0"),
		OutputVar: inputVar,
	}
}
