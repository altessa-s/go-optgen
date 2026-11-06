// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fields

import (
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/fieldtmpl"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// MapPutPlugin generates key/value setters for map fields.
type MapPutPlugin struct{}

// Meta returns field plugin metadata.
func (p *MapPutPlugin) Meta() plugin.Meta {
	return plugin.Meta{Kind: plugin.KindField, Name: "MapPut", Priority: mapFieldPriority}
}

// CanHandle reports whether the field uses optgen:"put" on a map.
func (p *MapPutPlugin) CanHandle(field model.OptField) bool {
	return field.IsMap && field.Metadata["put"] == "true"
}

var mapPutTemplate = fieldtmpl.Must("map-put", `{{template "optionDoc" .}}
func {{.FuncPrefix}}{{.OptionName}}`+
	`{{.TypeParamsDecl}}({{.KeyParam}} {{.MapKeyType}}, `+
	`{{.ValueParam}} {{.MapValueType}}) {{.OptionType}}{{.TypeParamsNames}} {
	return func(o *{{.TypeName}}{{.TypeParamsNames}}){{if .OptionReturnsError}} error{{end}} {
{{- if .MapKeyString}}
		var zeroKey {{.MapKeyType}}
		if {{.KeyParam}} == zeroKey { return{{if .OptionReturnsError}} nil{{end}} }
{{- end}}
{{- if .MapValueNilable}}
		if {{.ValueParam}} == nil { return{{if .OptionReturnsError}} nil{{end}} }
{{- end}}
		if o.{{.FieldName}} == nil { o.{{.FieldName}} = make({{.Type}}) }
		{{.MarkSet}}o.{{.FieldName}}[{{.KeyParam}}] = {{.ValueParam}}{{if .PostAssignment}}
		{{.PostAssignment}}{{end}}
		{{template "optionReturn" .}}
	}
}`)

// Generate renders the map key/value setter.
func (p *MapPutPlugin) Generate(ctx plugin.GenerationContext, field model.OptField, _ plugin.Pipeline) (plugin.GeneratedCode, error) {
	data := fieldtmpl.NewBaseData(ctx, field)
	data.DocSummary = "adds a " + field.FieldName + " entry."
	keyParam := uniqueCollectionParameter("key", ctx, field, data.PostAssignment, field.MapKeyType, field.MapValueType, field.Type)
	valueParam := uniqueCollectionParameter("value", ctx, field, data.PostAssignment, field.MapKeyType, field.MapValueType, field.Type, keyParam)

	return fieldtmpl.Execute(mapPutTemplate, &struct {
		fieldtmpl.BaseData
		KeyParam   string
		ValueParam string
	}{BaseData: data, KeyParam: keyParam, ValueParam: valueParam})
}

func uniqueCollectionParameter(name string, ctx plugin.GenerationContext, field model.OptField, sources ...string) string {
	used := fieldtmpl.ParameterIdentifiers(ctx, field, sources...)

	for used[name] {
		name += setterParameterSuffix
	}

	return name
}
