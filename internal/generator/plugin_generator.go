// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package generator

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// Generate produces a gofmt-formatted Go source file containing functional-option
// functions for every field in in.Fields. Each field is dispatched to the
// highest-priority [plugin.FieldPlugin] that can handle it; if no plugin matches,
// an error is returned. The output is deterministically ordered by option name.
func (g *Generator) Generate(in model.GenerateInput) ([]byte, error) {
	ctx := plugin.NewGenerationContext(in.PackageName, in.RenderSettings, in.Fields, in.Imports, in.GenericInfo)

	type funcOut struct {
		fieldName  string
		optionName string
		code       string
	}
	functionsOut := make([]funcOut, 0, len(in.Fields))
	importMap := make(map[string]model.GeneratedImport)
	var typeDefs []string
	var helpers []string
	needsTypeImports := make(map[string]bool, len(in.Fields))

	addImport := func(imp model.GeneratedImport) {
		if imp.Path == "" {
			return
		}
		if existing, ok := importMap[imp.Path]; ok && existing.Alias != "" && imp.Alias == "" {
			return
		}
		importMap[imp.Path] = imp
	}

	for _, field := range in.Fields {
		p, err := g.plugins.FindPlugin(field)
		if err != nil {
			return nil, err
		}

		result, err := p.Generate(ctx, field, g.pipeline)
		if err != nil {
			return nil, fmt.Errorf("plugin %s failed to generate code for field %s: %w", plugin.GetPluginName(p), field.FieldName, err)
		}

		functionsOut = append(functionsOut, funcOut{
			fieldName:  field.FieldName,
			optionName: field.OptionName,
			code:       result.Code,
		})
		needsTypeImports[field.FieldName] = strings.TrimSpace(result.Code) != ""
		typeDefs = append(typeDefs, result.TypeDefs...)
		helpers = append(helpers, result.Helpers...)
	}

	for _, imp := range ctx.GetAdditionalImports() {
		addImport(imp)
	}

	fieldsForImports := in.Fields
	if !in.GenerateDefaultFunc {
		fieldsForImports = slices.Clone(in.Fields)
		for i := range fieldsForImports {
			fieldsForImports[i].Default = ""
		}
	}
	requiredImports := sourceImports(fieldsForImports, in.Imports, needsTypeImports)
	for _, imp := range requiredImports {
		addImport(imp)
	}

	finalImports := make([]model.GeneratedImport, 0, len(importMap))
	for _, imp := range importMap {
		finalImports = append(finalImports, imp)
	}
	slices.SortFunc(finalImports, func(a, b model.GeneratedImport) int {
		if c := cmp.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		return cmp.Compare(a.Alias, b.Alias)
	})

	slices.Sort(typeDefs)
	typeDefs = slices.Compact(typeDefs)
	slices.Sort(helpers)
	helpers = slices.Compact(helpers)
	slices.SortFunc(functionsOut, func(a, b funcOut) int {
		if c := cmp.Compare(a.optionName, b.optionName); c != 0 {
			return c
		}
		return cmp.Compare(a.fieldName, b.fieldName)
	})
	generatedFunctions := make([]string, 0, len(functionsOut))
	for _, fn := range functionsOut {
		generatedFunctions = append(generatedFunctions, fn.code)
	}

	data := struct {
		Package             string
		TypeName            string
		OptionType          string
		GenerateOptionType  bool
		GenerateDefaultFunc bool
		GenerateNewFunc     bool
		OptionReturnsError  bool
		DefaultFuncName     string
		NewFuncName         string
		NormalizationMethod string
		Imports             []model.GeneratedImport
		TypeDefs            []string
		Functions           []string
		Helpers             []string
		Fields              []model.OptField
		GenericInfo         model.GenericInfo
	}{
		Package:             in.PackageName,
		TypeName:            in.TypeName,
		OptionType:          in.OptionType,
		GenerateOptionType:  in.GenerateOptionType,
		GenerateDefaultFunc: in.GenerateDefaultFunc,
		GenerateNewFunc:     in.GenerateNewFunc,
		OptionReturnsError:  in.OptionReturnsError,
		DefaultFuncName:     in.DefaultFuncName,
		NewFuncName:         in.NewFuncName,
		NormalizationMethod: in.NormalizationMethod,
		Imports:             finalImports,
		TypeDefs:            typeDefs,
		Functions:           generatedFunctions,
		Helpers:             helpers,
		Fields:              in.Fields,
		GenericInfo:         in.GenericInfo,
	}

	return g.renderPluginTemplate(data)
}
