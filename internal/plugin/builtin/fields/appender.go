// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fields

import (
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/fieldtmpl"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// DefaultAppenderPlugin generates inline appender for slice fields without modifiers.
//
// Generated code example:
//
//	func WithPaths(v ...string) Option {
//	    return func(o *options) {
//	        o.paths = append(o.paths, v...)
//	    }
//	}
type DefaultAppenderPlugin struct{}

// Meta returns plugin metadata.
func (p *DefaultAppenderPlugin) Meta() plugin.Meta {
	return plugin.Meta{
		Kind:     plugin.KindField,
		Priority: 0, // fallback
	}
}

var appenderTemplate = fieldtmpl.Must("appender", `// {{.FuncPrefix}}{{.OptionName}} appends to the {{.FieldName}} option.
func {{.FuncPrefix}}{{.OptionName}}{{.TypeParamsDecl}}(v {{if .SliceParam}}{{.Type}}{{else}}...{{.ElemType}}{{end}}) {{.OptionType}}{{.TypeParamsNames}} {
	return func(o *{{.TypeName}}{{.TypeParamsNames}}){{if .OptionReturnsError}} error{{end}} {
{{.PipelineCode}}{{if not .Final}}		{{.MarkSet}}o.{{.FieldName}} = append(o.{{.FieldName}}, {{.ValueExpr}}...)
{{- end}}{{- template "optionReturn" . }}
	}
}`)

// CanHandle reports whether the plugin applies to field.
func (p *DefaultAppenderPlugin) CanHandle(field model.OptField) bool {
	// Handle slice fields when append semantics are explicitly requested.
	// Transforms are applied automatically via TransformCode.
	if !field.IsSlice {
		return false
	}
	return plugin.IsTruthyMetadata(field.Metadata, "append")
}

// Generate renders the option function for field.
func (p *DefaultAppenderPlugin) Generate(ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline) (plugin.GeneratedCode, error) {
	data := fieldtmpl.NewData(ctx, field, values, "v")
	return fieldtmpl.Execute(appenderTemplate, data)
}
