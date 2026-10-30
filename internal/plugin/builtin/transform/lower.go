// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package transform

import (
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// LowerTransformPlugin applies strings.ToLower to string expressions.
// Tag usage: optval:"lower"
type LowerTransformPlugin struct {
	plugin.TransformBase
}

// Apply wraps expr in the transformation.
func (t *LowerTransformPlugin) Apply(ctx plugin.GenerationContext, _ model.OptField, _ string, expr string) (string, bool) {
	ctx.AddImport("strings")
	return "strings.ToLower(" + expr + ")", true
}
