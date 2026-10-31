// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package check

import (
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const nonzeroPriority = 90

// NonZeroCheck generates validation code ensuring a numeric value is non-zero.
// Supports integer and floating-point types, including named ones such as
// time.Duration or a type Port int when type information is available.
// Tag usage: optcheck:"nonzero"
type NonZeroCheck struct {
	plugin.CheckBase
}

// Validate accepts the numeric types listed on [NonZeroCheck].
func (c *NonZeroCheck) Validate(field model.OptField, _ string) error {
	if !plugin.IsNumericField(field) {
		return notApplicable(field, strings.Join(plugin.NumericTypes(), ", "))
	}
	return nil
}

// Generate renders the validation code for valueVar.
func (c *NonZeroCheck) Generate(ctx plugin.GenerationContext, field model.OptField, valueVar, _ string) []string {
	return []string{
		"if " + valueVar + " == 0 {",
		"  " + buildFail(ctx, field.FieldName+" must be non-zero"),
		"}",
	}
}
