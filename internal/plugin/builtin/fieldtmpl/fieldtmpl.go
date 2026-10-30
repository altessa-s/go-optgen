// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldtmpl

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// Common defines the shared "optionReturn" template that renders the
// trailing return of an option function.
const Common = `
{{- define "optionReturn" -}}
{{- if .OptionReturnsError}}
		return nil
{{- end}}
{{- end}}`

// Must parses a field template prefixed with [Common]. It panics on invalid
// syntax and is intended for package-level template variables.
func Must(name, body string) *template.Template {
	return template.Must(template.New(name).Parse(Common + body))
}

// Execute renders tmpl with data into the code of one option function.
func Execute(tmpl *template.Template, data any) (plugin.GeneratedCode, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return plugin.GeneratedCode{}, err
	}
	return plugin.GeneratedCode{Code: buf.String()}, nil
}

// BaseData carries the field and the option type identity shared by every
// field template.
type BaseData struct {
	model.OptField
	TypeName           string
	OptionType         string
	OptionReturnsError bool
	TypeParamsDecl     string // "[T any]" or "" for non-generic
	TypeParamsNames    string // "[T]" or "" for non-generic
	FuncPrefix         string // Option function prefix, "With" by default
	KeepEmptyStrings   bool   // Assign empty strings instead of skipping them
}

// NewBaseData collects the template data for field from ctx.
func NewBaseData(ctx plugin.GenerationContext, field model.OptField) BaseData {
	return BaseData{
		OptField:           field,
		TypeName:           ctx.TypeName,
		OptionType:         ctx.OptionType,
		OptionReturnsError: ctx.OptionReturnsError,
		TypeParamsDecl:     ctx.TypeParamsDecl(),
		TypeParamsNames:    ctx.TypeParamsNames(),
		FuncPrefix:         ctx.OptionFuncPrefix(),
		KeepEmptyStrings:   ctx.KeepEmptyStrings,
	}
}

// Data adds the value pipeline output to [BaseData]: the indented pipeline
// code, the final value expression, and whether the pipeline terminated.
type Data struct {
	BaseData
	PipelineCode string
	ValueExpr    string
	Final        bool
}

// NewData runs values for field starting from inputVar.
func NewData(ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline, inputVar string) Data {
	result := values.BuildPipeline(ctx, field, inputVar)
	code := ""
	if result.Code != "" {
		code = "\t\t" + strings.ReplaceAll(result.Code, "\n", "\n\t\t") + "\n"
	}
	return Data{BaseData: NewBaseData(ctx, field), PipelineCode: code, ValueExpr: result.ValueVar, Final: result.EarlyReturn}
}
