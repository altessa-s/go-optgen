// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package check

import (
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const minlenPriority = 20

// MinLenCheck generates validation code ensuring a string/slice/map has at least N elements.
// Tag usage: optcheck:"minlen=N" where N is a positive integer.
type MinLenCheck struct {
	plugin.CheckBase
}

// Validate accepts string, slice and map fields.
func (c *MinLenCheck) Validate(field model.OptField, _ string) error {
	return validateLenBound(field)
}

// Generate renders the validation code for valueVar.
func (c *MinLenCheck) Generate(ctx plugin.GenerationContext, field model.OptField, valueVar, rawValue string) []string {
	return buildLenBoundCheck(ctx, field, valueVar, rawValue, "<",
		"%s is too short (min %s)",
		"%s has too few elements (min %s)",
	)
}
