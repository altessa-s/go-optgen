// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fields

import (
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/fieldtmpl"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const boolFlagPriority = 20

// BoolFlagPlugin generates WithXxx() or WithoutXxx() for bool fields.
//
// Default behavior (no modifiers):
//
//	func WithEnabled() Option {
//	    return func(o *options) {
//	        o.enabled = true
//	    }
//	}
//
// With optval:"invert":
//
//	func WithoutValidation() Option {
//	    return func(o *options) {
//	        o.skipValidation = true
//	    }
//	}
//
// With optval:"param":
//
//	func WithEnforceMandatory(v bool) Option {
//	    return func(o *options) {
//	        o.enforceMandatory = v
//	    }
//	}
type BoolFlagPlugin struct{}

// Meta returns plugin metadata.
func (p *BoolFlagPlugin) Meta() plugin.Meta {
	return plugin.Meta{
		Kind:     plugin.KindField,
		Priority: boolFlagPriority,
	}
}

type boolFlagData struct {
	fieldtmpl.Data
	FuncPrefix  string
	CommentVerb string
}

var boolFlagTemplate = fieldtmpl.Must("bool_flag", `// {{.FuncPrefix}}{{.OptionName}} {{.CommentVerb}} the {{.FieldName}} option.
func {{.FuncPrefix}}{{.OptionName}}{{.TypeParamsDecl}}() {{.OptionType}}{{.TypeParamsNames}} {
	return func(o *{{.TypeName}}{{.TypeParamsNames}}){{if .OptionReturnsError}} error{{end}} {
		value := true
{{.PipelineCode}}{{if not .Final}}{{.MarkSet}}o.{{.FieldName}} = {{.ValueExpr}}{{end}}
{{- template "optionReturn" . }}
	}
}`)

var boolSetterTemplate = fieldtmpl.Must("bool_setter", `// {{.FuncPrefix}}{{.OptionName}} sets the {{.FieldName}} option.
func {{.FuncPrefix}}{{.OptionName}}{{.TypeParamsDecl}}(v bool) {{.OptionType}}{{.TypeParamsNames}} {
	return func(o *{{.TypeName}}{{.TypeParamsNames}}){{if .OptionReturnsError}} error{{end}} {
		{{.PipelineCode}}{{if not .Final}}{{.MarkSet}}o.{{.FieldName}} = {{.ValueExpr}}{{end}}
{{- template "optionReturn" . }}
	}
}`)

// CanHandle reports whether the plugin applies to field.
func (p *BoolFlagPlugin) CanHandle(field model.OptField) bool {
	return field.Type == "bool" && !field.IsSlice
}

// Generate renders the option function for field.
func (p *BoolFlagPlugin) Generate(ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline) (plugin.GeneratedCode, error) {
	if field.HasModifier("param") {
		data := fieldtmpl.NewData(ctx, field, values, "v")
		return fieldtmpl.Execute(boolSetterTemplate, data)
	}

	// Check for invert modifier
	invert := field.HasModifier("invert")

	data := boolFlagData{
		Data: fieldtmpl.NewData(ctx, field, values, "value"),
	}

	if invert {
		data.FuncPrefix = ctx.OptionInvertFuncPrefix()
		data.CommentVerb = "disables"
	} else {
		data.FuncPrefix = ctx.OptionFuncPrefix()
		data.CommentVerb = "enables"
	}

	return fieldtmpl.Execute(boolFlagTemplate, data)
}
