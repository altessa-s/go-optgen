// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package check

import (
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const maxlenPriority = 30

// MaxLenCheck generates validation code ensuring a string/slice/map has at most N elements.
// Tag usage: optcheck:"maxlen=N" where N is a positive integer.
type MaxLenCheck struct {
	plugin.CheckBase
}

// Validate accepts string, slice and map fields.
func (c *MaxLenCheck) Validate(field model.OptField, _ string) error {
	return validateLenBound(field)
}

// Generate renders the validation code for valueVar.
func (c *MaxLenCheck) Generate(ctx plugin.GenerationContext, field model.OptField, valueVar, rawValue string) []string {
	return buildLenBoundCheck(ctx, field, valueVar, rawValue, ">",
		"%s is too long (max %s)",
		"%s has too many elements (max %s)",
	)
}
