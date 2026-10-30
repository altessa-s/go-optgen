// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package check

import (
	"fmt"
	"slices"
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// numericTypes lists the types supported by numeric checks.
var numericTypes = []string{
	"int", "int8", "int16", "int32", "int64",
	"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
	"float32", "float64",
	"time.Duration",
}

// valueTypes lists well-known types that can never hold nil.
var valueTypes = append([]string{"bool", "byte", "rune", "complex64", "complex128", "time.Time"}, numericTypes...)

// buildFail renders the statement reporting a failed check: an error result,
// or for options without one a panic or, in [model.CheckFailureSkip] mode, a
// return that keeps the previous value.
func buildFail(ctx plugin.GenerationContext, msg string) string {
	if !ctx.OptionReturnsError && ctx.CheckFailure == model.CheckFailureSkip {
		return "return"
	}
	ctx.AddImport("fmt")
	errExpr := fmt.Sprintf(`fmt.Errorf(%q)`, msg)
	if ctx.OptionReturnsError {
		return "return " + errExpr
	}
	return "panic(" + errExpr + ")"
}

// notApplicable reports that field's type is outside supported.
func notApplicable(field model.OptField, supported string) error {
	return fmt.Errorf("%w %s (supported: %s)", plugin.ErrNotApplicable, field.Type, supported)
}

// canBeNil reports whether a value of field may be nil. Without type
// information, named types are assumed to be nilable (most often interfaces).
func canBeNil(field model.OptField) bool {
	if field.TypeResolved {
		return field.IsNilable
	}
	return !slices.Contains(valueTypes, field.Type) && !strings.HasPrefix(field.Type, "[") && !strings.HasPrefix(field.Type, "struct")
}

// isNumeric reports whether field has an integer or floating-point type,
// including named types such as time.Duration or a type Port int.
func isNumeric(field model.OptField) bool {
	return !field.IsSlice && (field.IsNumeric || slices.Contains(numericTypes, field.Type))
}

func validateLenBound(field model.OptField) error {
	switch plugin.ValueKindOf(field) {
	case plugin.ValueString, plugin.ValueSlice, plugin.ValueMap:
		return nil
	default:
		return notApplicable(field, "string, slice, map")
	}
}

func buildLenBoundCheck(
	ctx plugin.GenerationContext,
	field model.OptField,
	valueVar string,
	rawValue string,
	op string,
	stringMsgFmt string,
	collectionMsgFmt string,
) []string {
	limit := strings.TrimSpace(rawValue)
	msg := fmt.Sprintf(stringMsgFmt, field.FieldName, limit)
	if plugin.ValueKindOf(field) != plugin.ValueString {
		msg = fmt.Sprintf(collectionMsgFmt, field.FieldName, limit)
	}

	return []string{
		"if len(" + valueVar + ") " + op + " (" + limit + ") {",
		"  " + buildFail(ctx, msg),
		"}",
	}
}
