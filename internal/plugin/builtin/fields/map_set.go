// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fields

import (
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/fieldtmpl"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const mapFieldPriority = 45

// MapSetPlugin generates replacement or accumulating setters for map[K]struct{} fields.
type MapSetPlugin struct{}

// Meta returns field plugin metadata.
func (p *MapSetPlugin) Meta() plugin.Meta {
	return plugin.Meta{Kind: plugin.KindField, Name: "MapSet", Priority: mapFieldPriority}
}

// CanHandle reports whether the field uses optgen:"set" on a map.
func (p *MapSetPlugin) CanHandle(field model.OptField) bool {
	return field.IsMap && field.Metadata["set"] == "true"
}

var mapSetTemplate = fieldtmpl.Must("map-set", `{{template "optionDoc" .}}
func {{.FuncPrefix}}{{.OptionName}}{{.TypeParamsDecl}}(
	{{.ValuesParam}} {{if .SliceParam}}[]{{.MapKeyType}}{{else}}...{{.MapKeyType}}{{end}},
) {{.OptionType}}{{.TypeParamsNames}} {
	return func(o *{{.TypeName}}{{.TypeParamsNames}}){{if .OptionReturnsError}} error{{end}} {
{{if .Append}}        if len({{.ValuesParam}}) == 0 { return{{if .OptionReturnsError}} nil{{end}} }
        {{.MarkSet}}
        if o.{{.FieldName}} == nil { o.{{.FieldName}} = make({{.Type}}) }
{{else}}        {{.MarkSet}}o.{{.FieldName}} = make({{.Type}}, len({{.ValuesParam}}))
{{end}}        for _, key := range {{.ValuesParam}} { o.{{.FieldName}}[key] = struct{}{} }
        {{.PostAssignment}}{{template "optionReturn" .}}
	}
}`)

// Generate renders the map key setter.
func (p *MapSetPlugin) Generate(ctx plugin.GenerationContext, field model.OptField, _ plugin.Pipeline) (plugin.GeneratedCode, error) {
	data := fieldtmpl.NewBaseData(ctx, field)
	data.DocSummary = "sets the " + field.FieldName + " option."

	return fieldtmpl.Execute(mapSetTemplate, &struct {
		fieldtmpl.BaseData
		Append      bool
		ValuesParam string
	}{BaseData: data, Append: plugin.IsTruthyMetadata(field.Metadata, "append"), ValuesParam: uniqueCollectionParameter("v", field.MapKeyType, field.Type)})
}
