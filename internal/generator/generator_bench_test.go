// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package generator_test

import (
	"testing"

	"github.com/altessa-s/go-optgen/internal/generator"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

func BenchmarkGenerate(b *testing.B) {
	r := plugin.NewRegistry()
	if err := builtin.Register(r); err != nil {
		b.Fatal(err)
	}
	g := generator.New(r, plugin.NewPipeline(r))
	in := model.GenerateInput{
		PackageName:         "sample",
		TypeName:            "options",
		OptionType:          "Option",
		GenerateOptionType:  true,
		GenerateDefaultFunc: true,
		GenerateNewFunc:     true,
		DefaultFuncName:     "defaultOptions",
		NewFuncName:         "newOptions",
		Fields: []model.OptField{
			{FieldName: "name", OptionName: "Name", Type: "string", Modifiers: []string{"trimspaces"}, HasModifiers: true},
			{FieldName: "retries", OptionName: "Retries", Type: "int", Default: "3"},
		},
	}

	for b.Loop() {
		_, _ = g.Generate(in)
	}
}
