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

// Generator renders prepared fields using explicitly supplied plugins.
type Generator struct {
	plugins  FieldSelector
	pipeline plugin.Pipeline
}

// New creates a renderer with a required plugin catalog.
func New(plugins FieldSelector, pipeline plugin.Pipeline) *Generator {
	return &Generator{plugins: plugins, pipeline: pipeline}
}
