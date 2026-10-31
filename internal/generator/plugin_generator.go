// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
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
//
// in must be prepared: settings policies such as dropping defaults for
// [model.DefaultsFromNone] and the WithIsSet limits are applied before rendering.
func (g *Generator) Generate(in model.GenerateInput) ([]byte, error) {
	ctx := plugin.NewGenerationContext(in.PackageName, in.RenderSettings, in.Fields, in.Imports, in.GenericInfo)
	ctx.IsSetField = in.IsSetField

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
		// Mandatory fields have no option but appear in the constructor signature.
		needsTypeImports[field.FieldName] = strings.TrimSpace(result.Code) != "" || field.Mandatory
		typeDefs = append(typeDefs, result.TypeDefs...)
		helpers = append(helpers, result.Helpers...)
	}

	validation := g.validationCode(ctx, in)

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
		Mandatory           []mandatoryParam
		DefaultsFrom        model.DefaultsSource
		DefaultsName        string
		ValidateFuncName    string
		Validation          string
		ValidateFailure     string
		IsSet               *isSetData
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
		Mandatory:           mandatoryParams(in.Fields),
		DefaultsFrom:        in.DefaultsFrom,
		DefaultsName:        in.DefaultsName,
		ValidateFuncName:    validateFuncName(in.RenderSettings),
		Validation:          validation,
		ValidateFailure:     validateFailure(in.RenderSettings),
		IsSet:               newIsSetData(ctx, in),
	}

	return g.renderPluginTemplate(data)
}

// mandatoryParam is a constructor parameter assigned to a mandatory field.
type mandatoryParam struct {
	Field, Param, Type string
}

// mandatoryParams lists the mandatory fields in declaration order. Parameter
// names avoid the identifiers used by the generated constructor.
func mandatoryParams(fields []model.OptField) []mandatoryParam {
	var out []mandatoryParam
	for _, f := range fields {
		if !f.Mandatory {
			continue
		}
		param := f.FieldName
		if slices.Contains([]string{"o", "opt", "opts", "err"}, param) {
			param += "Value"
		}
		out = append(out, mandatoryParam{Field: f.FieldName, Param: param, Type: f.Type})
	}
	return out
}

func validateFuncName(s model.RenderSettings) string {
	return cmp.Or(s.ValidateFuncName, "validate"+strings.ToUpper(s.TypeName[:1])+s.TypeName[1:])
}

// validateFailure returns how newOptions reports a validation error: "return"
// for error results, "panic", or "" when final validation is skipped.
func validateFailure(s model.RenderSettings) string {
	switch {
	case !s.Validate:
		return ""
	case s.OptionReturnsError:
		return "return"
	case s.CheckFailure == model.CheckFailureSkip:
		return ""
	default:
		return "panic"
	}
}

// validationCode renders every field's checks of its final value for the
// validation function; validation always reports an error.
func (g *Generator) validationCode(ctx plugin.GenerationContext, in model.GenerateInput) string {
	if !in.Validate {
		return ""
	}
	checkCtx := ctx
	checkCtx.OptionReturnsError = true
	var lines []string
	for _, f := range in.Fields {
		lines = append(lines, g.pipeline.BuildChecks(checkCtx, f, "o."+f.FieldName)...)
	}
	if len(lines) == 0 {
		return ""
	}
	return "\t" + strings.Join(lines, "\n\t") + "\n"
}

// isSetData describes the generated field-set constants and isSet method.
type isSetData struct {
	FieldType, SetType, Field string
	Consts                    []string
}

func newIsSetData(ctx plugin.GenerationContext, in model.GenerateInput) *isSetData {
	if !in.WithIsSet {
		return nil
	}
	d := &isSetData{FieldType: in.TypeName + "Field", SetType: in.TypeName + "FieldSet", Field: in.IsSetField}
	for _, f := range in.Fields {
		d.Consts = append(d.Consts, ctx.IsSetConst(f))
	}
	return d
}
