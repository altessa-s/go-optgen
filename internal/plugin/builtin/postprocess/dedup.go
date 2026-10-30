// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package postprocess

import (
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const (
	dedupPriority = 5
	kindString    = "string"
	durationType  = "time.Duration"
)

// DedupModifier applies deduplication to slice values.
// It is applied by default to slices of comparable types (primitives, pointers).
//
// Tag usage: optval:"dedup" (explicit) or automatic for comparable slices
// To disable: optval:"nodup"
//
// Generated code keeps the first occurrence of each element:
//
//	{
//		seen := make(map[string]struct{}, len(v))
//		deduped := make([]string, 0, len(v))
//		for _, item := range v {
//			if _, ok := seen[item]; ok {
//				continue
//			}
//			seen[item] = struct{}{}
//			deduped = append(deduped, item)
//		}
//		v = deduped
//	}
type DedupModifier struct{}

// Meta returns plugin metadata including default application for slices.
func (m *DedupModifier) Meta() plugin.Meta {
	return plugin.Meta{
		Kind:            plugin.KindModifier,
		Priority:        dedupPriority,
		DefaultForTypes: []string{"[]"}, // Apply to all slice types (filtered by CanHandle)
		DisabledBy:      "nodup",
	}
}

// Key returns the modifier name used in tags.
func (m *DedupModifier) Key() string { return "dedup" }

// Phase returns PhasePostProcess - runs after transforms.
func (m *DedupModifier) Phase() plugin.Phase { return plugin.PhasePostProcess }

// CanHandle returns true for slice fields with comparable element types.
// Deduplication requires comparable types to use them as map keys.
func (m *DedupModifier) CanHandle(field model.OptField) bool {
	if !field.IsSlice {
		return false
	}

	// Only handle slices with comparable element types
	return field.ElemComparable || isComparableType(field.ElemType)
}

// isComparableType checks if a type is known to be comparable.
// Returns true for primitive types, pointers, and common comparable types.
func isComparableType(typeStr string) bool {
	// Pointer types are always comparable
	if strings.HasPrefix(typeStr, "*") {
		return true
	}

	// Check for known comparable primitive types
	switch typeStr {
	case kindString, "bool", "byte", "rune", "error",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64",
		"complex64", "complex128":
		return true
	}

	// Common comparable types from standard library
	switch typeStr {
	case "time.Time", durationType:
		return true
	}

	return false
}

// Generate produces deduplication code that allocates a new slice, so the
// caller's variadic argument is never modified.
func (m *DedupModifier) Generate(_ plugin.GenerationContext, field model.OptField, inputVar string) plugin.ModifierResult {
	return plugin.ModifierResult{
		Code: []string{
			"{",
			"  seen := make(map[" + field.ElemType + "]struct{}, len(" + inputVar + "))",
			"  deduped := make([]" + field.ElemType + ", 0, len(" + inputVar + "))",
			"  for _, item := range " + inputVar + " {",
			"    if _, ok := seen[item]; ok {",
			"      continue",
			"    }",
			"    seen[item] = struct{}{}",
			"    deduped = append(deduped, item)",
			"  }",
			"  " + inputVar + " = deduped",
			"}",
		},
		OutputVar: inputVar,
	}
}
