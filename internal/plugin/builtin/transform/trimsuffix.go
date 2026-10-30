// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package transform

import (
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// TrimSuffixTransformPlugin applies strings.TrimSuffix to string expressions.
// Tag usage: optval:"trimsuffix=/" or optval:"trimsuffix=suffix"
type TrimSuffixTransformPlugin struct {
	plugin.TransformBase
}

// Apply wraps expr in the transformation.
func (t *TrimSuffixTransformPlugin) Apply(ctx plugin.GenerationContext, field model.OptField, _ string, expr string) (string, bool) {
	// Find the suffix parameter from field.Modifiers
	suffix := ""
	const keyPrefix = "trimsuffix="
	for _, mod := range field.Modifiers {
		if len(mod) > len(keyPrefix) && strings.EqualFold(mod[:len(keyPrefix)], keyPrefix) {
			suffix = mod[len(keyPrefix):]
			break
		}
	}
	if suffix == "" {
		return expr, false
	}

	ctx.AddImport("strings")
	return "strings.TrimSuffix(" + expr + ", " + quote(suffix) + ")", true
}
