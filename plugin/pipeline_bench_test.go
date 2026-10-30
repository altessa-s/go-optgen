// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package plugin_test

import (
	"testing"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

func BenchmarkBuildPipeline(b *testing.B) {
	r := plugin.NewRegistry()
	if err := r.Register(&mockModifierPlugin{key: "audit", phase: plugin.PhaseTransform, canHandle: func(model.OptField) bool { return true }, generate: func(_ plugin.GenerationContext, _ model.OptField, in string) plugin.ModifierResult {
		return plugin.ModifierResult{OutputVar: in}
	}}); err != nil {
		b.Fatal(err)
	}
	field := model.OptField{Modifiers: []string{"audit"}}
	pipeline := plugin.NewPipeline(r)
	for b.Loop() {
		pipeline.BuildPipeline(plugin.GenerationContext{}, field, "v")
	}
}
