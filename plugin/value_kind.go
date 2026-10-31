// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package plugin

import (
	"slices"
	"strings"

	"github.com/altessa-s/go-optgen/model"
)

// ValueKind classifies the option value of a field for value plugins.
type ValueKind string

// Value kinds reported by [ValueKindOf].
const (
	ValueString    ValueKind = "string"
	ValueSlice     ValueKind = "slice"
	ValueMap       ValueKind = "map"
	ValuePointer   ValueKind = "pointer"
	ValueInterface ValueKind = "interface"
	ValueOther     ValueKind = "other"
)

// ValueKindOf classifies field by its syntactic type. Named types the parser
// cannot resolve (for example io.Writer or time.Duration) are [ValueOther].
func ValueKindOf(field model.OptField) ValueKind {
	switch {
	case field.IsSlice:
		return ValueSlice
	case field.Type == "string":
		return ValueString
	case strings.HasPrefix(field.Type, "*"):
		return ValuePointer
	case strings.HasPrefix(field.Type, "map["):
		return ValueMap
	case field.IsInterface:
		return ValueInterface
	default:
		return ValueOther
	}
}

// NumericTypes returns the well-known integer, floating-point and duration
// types that numeric value plugins accept without type information.
func NumericTypes() []string {
	return []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64",
		"time.Duration",
	}
}

// IsNumericField reports whether field holds a single integer or
// floating-point value: one of [NumericTypes], or a named type that the
// parser resolved as numeric, such as type Port int.
func IsNumericField(field model.OptField) bool {
	return !field.IsSlice && (field.IsNumeric || slices.Contains(NumericTypes(), field.Type))
}
