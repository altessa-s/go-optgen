// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package check

import (
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const requiredPriority = 10

// RequiredCheck generates validation code ensuring a field value is non-empty/non-nil.
// Supports: strings (empty check), slices/maps (nil or empty check), pointers,
// interfaces and named types that may hold nil (nil check, including an
// interface wrapping a nil pointer). Types that can never be nil, such as int,
// bool or time.Duration, are rejected; use nonzero for numbers.
// Tag usage: optcheck:"required" or optcheck:"nonempty" (alias)
type RequiredCheck struct {
	plugin.CheckBase
}

// Validate rejects fields whose type can never be empty or nil.
func (c *RequiredCheck) Validate(field model.OptField, _ string) error {
	if plugin.ValueKindOf(field) == plugin.ValueOther && !canBeNil(field) {
		return notApplicable(field, "string, slice, map, pointer, interface")
	}
	return nil
}

// Generate renders the validation code for valueVar.
func (c *RequiredCheck) Generate(ctx plugin.GenerationContext, field model.OptField, valueVar, _ string) []string {
	fail := buildFail(ctx, field.FieldName+" is required")
	switch plugin.ValueKindOf(field) {
	case plugin.ValuePointer:
		return []string{"if " + valueVar + " == nil {", "  " + fail, "}"}
	case plugin.ValueString:
		return []string{`if ` + valueVar + ` == "" {`, "  " + fail, "}"}
	case plugin.ValueSlice, plugin.ValueMap:
		return []string{"if " + valueVar + " == nil || len(" + valueVar + ") == 0 {", "  " + fail, "}"}
	case plugin.ValueOther:
		if field.TypeResolved {
			// A resolved non-interface type that can be nil: named pointer,
			// map, slice, channel or function types.
			return []string{"if " + valueVar + " == nil {", "  " + fail, "}"}
		}
		return plugin.BuildNilCheck(ctx, valueVar, fail)
	default:
		return plugin.BuildNilCheck(ctx, valueVar, fail)
	}
}
