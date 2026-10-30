// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package fields

import (
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/fieldtmpl"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// DefaultSetterPlugin generates inline setter for scalar fields.
//
// For non-string fields:
//
//	func WithTimeout(v time.Duration) Option {
//	    return func(o *options) {
//	        o.timeout = v
//	    }
//	}
//
// For string fields (accepts both string and *string, skips empty after transform):
//
//	func WithIssuer[T interface{string|*string}](v T) Option {
//	    return func(o *options) {
//	        switch t := any(v).(type) {
//	        case string:
//	            vv := strings.TrimSpace(t)
//	            if vv == "" { return }
//	            o.issuer = vv
//	        case *string:
//	            if t == nil { return }
//	            vv := strings.TrimSpace(*t)
//	            if vv == "" { return }
//	            o.issuer = vv
//	        }
//	    }
//	}
type DefaultSetterPlugin struct{}

// Meta returns plugin metadata.
func (p *DefaultSetterPlugin) Meta() plugin.Meta {
	return plugin.Meta{
		Kind:     plugin.KindField,
		Priority: 0, // fallback
	}
}

var setterTemplate = fieldtmpl.Must("setter", `// {{.FuncPrefix}}{{.OptionName}} sets the {{.FieldName}} option.
func {{.FuncPrefix}}{{.OptionName}}{{.TypeParamsDecl}}(v {{.Type}}) {{.OptionType}}{{.TypeParamsNames}} {
	return func(o *{{.TypeName}}{{.TypeParamsNames}}){{if .OptionReturnsError}} error{{end}} {
{{.PipelineCode}}{{if not .Final}}{{if .SkipEmpty}}		if {{.ValueExpr}} == "" { return{{if .OptionReturnsError}} nil{{end}} }
{{end}}		o.{{.FieldName}} = {{.ValueExpr}}
{{- end}}{{- template "optionReturn" . }}
	}
}`)

var stringSetterTemplate = fieldtmpl.Must("string_setter", `// {{.FuncPrefix}}{{.OptionName}} sets the {{.FieldName}} option.
func {{.FuncPrefix}}{{.OptionName}}[T interface{string|*string}](v T) {{.OptionType}} {
 return func(o *{{.TypeName}}){{if .OptionReturnsError}} error{{end}} {
 var value string
 switch t := any(v).(type) {
 case string: value = t
 case *string:
 if t == nil { return{{if .OptionReturnsError}} nil{{end}} }
 value = *t
 }
{{.PipelineCode}}{{if not .Final}}
{{if not .KeepEmptyStrings}} if {{.ValueExpr}} == "" { return{{if .OptionReturnsError}} nil{{end}} }
{{end}} o.{{.FieldName}} = {{.ValueExpr}}
{{- end}}{{- template "optionReturn" . }}
 }
}`)

// CanHandle reports whether the plugin applies to field.
func (p *DefaultSetterPlugin) CanHandle(field model.OptField) bool {
	// Handle scalar fields (transforms are applied automatically via TransformCode)
	return !field.IsSlice
}

// Generate renders the option function for field.
func (p *DefaultSetterPlugin) Generate(ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline) (plugin.GeneratedCode, error) {
	// The generic string setter cannot be combined with struct type parameters.
	if field.Type == "string" && !ctx.IsGeneric() && ctx.StringSetter != model.StringSetterPlain {
		return p.generateStringSetter(ctx, field, values)
	}

	data := struct {
		fieldtmpl.Data
		SkipEmpty bool
	}{
		Data: fieldtmpl.NewData(ctx, field, values, "v"),
		// A plain string setter skips empty values like the generic one.
		SkipEmpty: field.Type == "string" && ctx.StringSetter == model.StringSetterPlain && !ctx.KeepEmptyStrings,
	}
	return fieldtmpl.Execute(setterTemplate, data)
}

func (p *DefaultSetterPlugin) generateStringSetter(
	ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline,
) (plugin.GeneratedCode, error) {
	data := fieldtmpl.NewData(ctx, field, values, "value")
	return fieldtmpl.Execute(stringSetterTemplate, data)
}
