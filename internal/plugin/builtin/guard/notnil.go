// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package guard

import (
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const notNilGuardPriority = 100

// NotNilModifier implements optgen:"notnil" semantics as a pipeline guard.
//
// If the option input is nil, the option becomes a no-op and keeps defaults.
// This is useful for pointers/maps/interfaces and for variadic slice options invoked as xs... where xs is nil.
//
// Tag usage: optgen:"notnil"
//
// This modifier is applied by default to all pointer and interface types.
// Use optgen:"allownull" to disable this behavior for a specific field.
type NotNilModifier struct{}

// Meta returns plugin metadata.
func (m *NotNilModifier) Meta() plugin.Meta {
	return plugin.Meta{
		Kind:            plugin.KindModifier,
		Priority:        notNilGuardPriority, // run first in guards
		DefaultForTypes: []string{"*", "interface", "[]"},
		DisabledBy:      "allownull",
	}
}

// Key returns the modifier name used in tags.
func (m *NotNilModifier) Key() string { return "notnil" }

// Phase returns PhaseGuard - this runs first in the pipeline.
func (m *NotNilModifier) Phase() plugin.Phase { return plugin.PhaseGuard }

// CanHandle returns true for nilable types (pointers, slices, maps, interfaces, channels, functions).
func (m *NotNilModifier) CanHandle(field model.OptField) bool {
	// If user explicitly requested notnil via tag, trust them
	if _, explicit := field.Metadata[m.Key()]; explicit {
		return true
	}
	// Use IsNilable if set, otherwise fallback to type string checks
	if field.IsNilable || field.IsInterface {
		return true
	}
	// Fallback for backwards compatibility
	return strings.HasPrefix(field.Type, "*") || field.IsSlice || strings.HasPrefix(field.Type, "map[")
}

// Generate produces early-return code if input is nil (or empty for slices).
// Pointers, maps, channels and functions compare against nil directly. Other
// types, including interfaces and named types the parser cannot resolve, use a
// reflect-based check that also detects an interface wrapping a nil pointer (a
// common Go pitfall where v == nil is false but the underlying value is nil).
func (m *NotNilModifier) Generate(ctx plugin.GenerationContext, field model.OptField, inputVar string) plugin.ModifierResult {
	if inputVar == "" {
		inputVar = "v"
	}

	var code []string
	switch {
	case field.IsSlice:
		code = plugin.BuildEarlyReturn(ctx, "len("+inputVar+") == 0")
	case comparesWithNil(field):
		code = plugin.BuildEarlyReturn(ctx, inputVar+" == nil")
	default:
		code = plugin.BuildNilEarlyReturn(ctx, inputVar)
	}
	return plugin.ModifierResult{Code: code, OutputVar: inputVar}
}

// comparesWithNil reports whether the field type is a pointer, map, channel or
// function, whose nil value is detected by a direct comparison.
func comparesWithNil(field model.OptField) bool {
	if field.IsInterface {
		return false
	}
	return field.IsNilable || strings.HasPrefix(field.Type, "*") || strings.HasPrefix(field.Type, "map[")
}
