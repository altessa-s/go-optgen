// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fields

import (
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/fieldtmpl"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const sliceSetPriority = 35

// SliceSetPlugin generates "setter" semantics for slice fields (replace, not append).
//
// Default behavior for slices is "set" (no extra tag parameters required).
// To get append semantics, use:
//
//	opt:"FieldName" optgen:"append"
type SliceSetPlugin struct{}

// Meta returns plugin metadata.
func (p *SliceSetPlugin) Meta() plugin.Meta {
	return plugin.Meta{
		Kind:     plugin.KindField,
		Priority: sliceSetPriority, // higher than default appender
	}
}

// CanHandle reports whether the plugin applies to field.
func (p *SliceSetPlugin) CanHandle(field model.OptField) bool {
	if !field.IsSlice {
		return false
	}
	// If append is requested, do not handle (let appender plugin take it).
	if plugin.IsTruthyMetadata(field.Metadata, "append") {
		return false
	}
	// Transforms are applied automatically via TransformCode.
	return true
}

var sliceSetTemplate = fieldtmpl.Must("slice-set", `// {{.FuncPrefix}}{{.OptionName}} sets the {{.FieldName}} option.
func {{.FuncPrefix}}{{.OptionName}}{{.TypeParamsDecl}}(v {{if .SliceParam}}{{.Type}}{{else}}...{{.ElemType}}{{end}}) {{.OptionType}}{{.TypeParamsNames}} {
	return func(o *{{.TypeName}}{{.TypeParamsNames}}){{if .OptionReturnsError}} error{{end}} {
{{.PipelineCode}}{{if not .Final}}		{{.MarkSet}}o.{{.FieldName}} = {{.ValueExpr}}
{{- end}}{{- template "optionReturn" . }}
	}
}`)

// Generate renders the option function for field.
func (p *SliceSetPlugin) Generate(ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline) (plugin.GeneratedCode, error) {
	data := fieldtmpl.NewData(ctx, field, values, "v")
	return fieldtmpl.Execute(sliceSetTemplate, data)
}
