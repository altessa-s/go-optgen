// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package plugin

import (
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
