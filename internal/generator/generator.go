// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package generator

import (
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// FieldSelector supplies field selection for this renderer.
type FieldSelector interface {
	FindPlugin(model.OptField) (plugin.FieldPlugin, error)
}

// Pipeline is the value pipeline passed to field plugins plus the check
// rendering that the generated validation function needs.
type Pipeline interface {
	plugin.Pipeline
	// BuildChecks renders only the field's optcheck validations of valueVar.
	BuildChecks(ctx plugin.GenerationContext, field model.OptField, valueVar string) []string
}

// Generator renders prepared fields using explicitly supplied plugins.
type Generator struct {
	plugins  FieldSelector
	pipeline Pipeline
}

// New creates a renderer with a required plugin catalog.
func New(plugins FieldSelector, pipeline Pipeline) *Generator {
	return &Generator{plugins: plugins, pipeline: pipeline}
}
