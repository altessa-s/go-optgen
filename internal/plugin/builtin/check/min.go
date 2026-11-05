// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package check

import (
	"fmt"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const minPriority = 25

// MinCheck requires a numeric field to be at least the configured bound.
// Tag usage: optcheck:"min=1" or optcheck:"min=1s" for time.Duration.
type MinCheck struct{ plugin.CheckBase }

// Validate accepts numeric fields, including resolved named numeric types.
func (c *MinCheck) Validate(field model.OptField, value string) error {
	if !plugin.IsNumericField(field) {
		return notApplicable(field, "numeric")
	}

	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: min requires a value", plugin.ErrInvalidCheckValue)
	}

	if valid, ok := field.CheckValueValid[value]; ok && !valid {
		return fmt.Errorf("%w: min bound %q is not assignable to field type %s", plugin.ErrInvalidCheckValue, value, field.Type)
	}

	if _, err := parser.ParseExpr(value); err != nil {
		return fmt.Errorf("%w: min bound %q is not a Go expression", plugin.ErrInvalidCheckValue, value)
	}

	if evaluated, err := types.Eval(token.NewFileSet(), nil, token.NoPos, value); err == nil && evaluated.Value != nil {
		kind := evaluated.Value.Kind()
		if field.Type == "int" || strings.HasPrefix(field.Type, "int") || strings.HasPrefix(field.Type, "uint") {
			if kind != constant.Int {
				return fmt.Errorf("%w: min bound %q is not an integer", plugin.ErrInvalidCheckValue, value)
			}
		}
	}

	return nil
}

// Generate renders the lower-bound check for valueVar.
func (c *MinCheck) Generate(ctx plugin.GenerationContext, field model.OptField, valueVar, rawValue string) []string {
	limit := strings.TrimSpace(rawValue)

	return []string{
		"if " + valueVar + " < (" + limit + ") {",
		"  " + buildFail(ctx, fmt.Sprintf("%s is below minimum (%s)", field.FieldName, limit)),
		"}",
	}
}
