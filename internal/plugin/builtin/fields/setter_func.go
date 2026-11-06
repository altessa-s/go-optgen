// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fields

import (
	"fmt"
	"strings"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin/fieldtmpl"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// SetterFuncPlugin renders a setter whose parameter list comes from an optfn target.
type SetterFuncPlugin struct{}

const setterFuncPriority = 100
const setterParameterSuffix = "Arg"

// Meta returns the setter function plugin metadata.
func (p *SetterFuncPlugin) Meta() plugin.Meta {
	return plugin.Meta{Kind: plugin.KindField, Name: "SetterFunc", Priority: setterFuncPriority}
}

// CanHandle reports whether field has a resolved optfn target.
func (p *SetterFuncPlugin) CanHandle(field model.OptField) bool { return field.SetterTarget != nil }

var setterFuncTemplate = fieldtmpl.Must("setter_func", `{{template "optionDoc" .}}
func {{.FuncPrefix}}{{.OptionName}}{{.TypeParamsDecl}}({{.Parameters}}) {{.OptionType}}{{.TypeParamsNames}} {
	return func(o *{{.TypeName}}{{.TypeParamsNames}}){{if .OptionReturnsError}} error{{end}} {
{{- if .TargetReturnsError}}
		{{.ValueName}}, {{.ErrorName}} := {{.Call}}
		if {{.ErrorName}} != nil { return {{.ErrorName}} }
		{{.MarkSet}}o.{{.FieldName}} = {{.ValueName}}{{if .PostAssignment}}
		{{.PostAssignment}}{{end}}
{{- else}}
		{{.MarkSet}}o.{{.FieldName}} = {{.Call}}{{if .PostAssignment}}
		{{.PostAssignment}}{{end}}
{{- end}}
{{- template "optionReturn" . }}
	}
}`)

type setterFuncData struct {
	fieldtmpl.BaseData
	Parameters         string
	Call               string
	TargetReturnsError bool
	ValueName          string
	ErrorName          string
}

// Generate renders the function-backed option setter.
func (p *SetterFuncPlugin) Generate(ctx plugin.GenerationContext, field model.OptField, _ plugin.Pipeline) (plugin.GeneratedCode, error) {
	target := field.SetterTarget
	if target.ReturnsError && !ctx.OptionReturnsError {
		return plugin.GeneratedCode{}, fmt.Errorf(
			"field %s option %s target %s returns error; enable --option-error",
			field.FieldName, field.OptionName, target.Expression,
		)
	}

	for _, imp := range target.Imports {
		ctx.AddImport(imp.Path, imp.Alias)
	}

	baseData := fieldtmpl.NewBaseData(ctx, field)
	sources := []string{target.Expression, baseData.PostAssignment}

	for _, param := range target.Parameters {
		sources = append(sources, param.Type)
	}

	for _, imp := range target.Imports {
		sources = append(sources, imp.Alias)
	}

	used := fieldtmpl.ParameterIdentifiers(ctx, field, sources...)

	for _, imp := range target.Imports {
		if imp.Alias != "" {
			used[imp.Alias] = true
		}
	}

	params := make([]string, 0, len(target.Parameters))

	args := make([]string, 0, len(target.Parameters))
	for i, param := range target.Parameters {
		name := fieldtmpl.UniqueParameterName(parameterBaseName(param.Name, i), used)
		used[name] = true

		typ := param.Type
		if param.Variadic {
			typ = "..." + typ
		}

		params = append(params, name+" "+typ)

		if param.Variadic {
			name += "..."
		}

		args = append(args, name)
	}

	callTarget := target.Expression

	if target.Method {
		callTarget = "o." + field.FieldName + target.Expression
	}

	call := callTarget + "(" + strings.Join(args, ", ") + ")"

	data := setterFuncData{
		BaseData:           baseData,
		Parameters:         strings.Join(params, ", "),
		Call:               call,
		TargetReturnsError: target.ReturnsError,
	}
	if target.ReturnsError {
		data.ValueName = uniqueLocalName("optgenValue", used)
		used[data.ValueName] = true
		data.ErrorName = uniqueLocalName("optgenErr", used)
	}

	return fieldtmpl.Execute(setterFuncTemplate, &data)
}

func parameterBaseName(name string, index int) string {
	if name == "" || name == "_" {
		name = fmt.Sprintf("arg%d", index)
	}

	return name
}

func uniqueLocalName(name string, used map[string]bool) string {
	for used[name] {
		name += setterParameterSuffix
	}

	return name
}
