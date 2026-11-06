// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fields

import (
	"strings"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin/fieldtmpl"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const netIPParsePriority = 40

// NetIPParsePlugin generates a With* function that accepts IPs as strings and parses them into []net.IP.
//
// Enable by setting metadata in the tag:
//
//	opt:"FieldName" optgen:"parseip"
//
// Optional metadata:
//   - append: append parsed IPs instead of replacing (default: replace)
//   - notnil: if v is nil (nil slice expansion), do nothing
//   - err=ErrIdent: error expression to return on invalid IP in --option-error mode
//
// The generated option replaces the slice by default.
type NetIPParsePlugin struct{}

// Meta returns plugin metadata.
func (p *NetIPParsePlugin) Meta() plugin.Meta {
	return plugin.Meta{
		Kind:     plugin.KindField,
		Priority: netIPParsePriority, // higher than slice set/appender
	}
}

// CanHandle reports whether the plugin applies to field.
func (p *NetIPParsePlugin) CanHandle(field model.OptField) bool {
	if !field.IsSlice || field.ElemType != "net.IP" {
		return false
	}

	return plugin.IsTruthyMetadata(field.Metadata, "parseip")
}

var netIPParseTemplate = fieldtmpl.Must("net-ip-parse",
	`{{template "optionDoc" .}}
func {{.FuncPrefix}}{{.OptionName}}{{.TypeParamsDecl}}({{.ParamVar}} ...string) {{.OptionType}}{{.TypeParamsNames}} {
	return func(o *{{.TypeName}}{{.TypeParamsNames}}){{if .OptionReturnsError}} error{{end}} {
		{{.InputVar}} := make([]net.IP, 0, len({{.ParamVar}}))
		for _, {{.RangeVar}} := range {{.ParamVar}} {
			{{.IPVar}} := net.ParseIP({{.RangeVar}})
			if {{.IPVar}} == nil {
				{{- if .OptionReturnsError}}
				return {{.ErrExpr}}
				{{- else}}
				return
				{{- end}}
			}
			{{.InputVar}} = append({{.InputVar}}, {{.IPVar}})
		}
{{.PipelineCode}}{{if not .Final}}		{{- if .Append}}
		{{.MarkSet}}o.{{.FieldName}} = append(o.{{.FieldName}}, {{.ValueExpr}}...)
		{{- else}}
		{{.MarkSet}}o.{{.FieldName}} = {{.ValueExpr}}
		{{- end}}
		{{- if .PostAssignment}}{{.PostAssignment}}{{- end}}{{- end}}{{- template "optionReturn" . }}
	}
}`)

// Generate renders the option function for field.
func (p *NetIPParsePlugin) Generate(ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline) (plugin.GeneratedCode, error) {
	ctx.AddImport("net")

	appendMode := plugin.IsTruthyMetadata(field.Metadata, "append")

	errExpr := ""

	if v, ok := field.Metadata["err"]; ok {
		v = strings.TrimSpace(v)
		if v != "" {
			errExpr = v
		}
	}

	if ctx.OptionReturnsError && (errExpr == "" || strings.HasPrefix(errExpr, "fmt.")) {
		ctx.AddImport("fmt")
	}

	data := struct {
		fieldtmpl.Data
		Append   bool
		ErrExpr  string
		ParamVar string
		RangeVar string
		IPVar    string
	}{
		Data:   fieldtmpl.NewData(ctx, field, values, "netIPs"),
		Append: appendMode,
	}
	used := fieldtmpl.ParameterIdentifiers(ctx, field, data.PostAssignment, field.Type, field.ElemType)
	data.ParamVar = fieldtmpl.UniqueParameterName("v", used)
	used[data.ParamVar] = true
	data.RangeVar = fieldtmpl.UniqueParameterName("s", used)
	used[data.RangeVar] = true
	data.IPVar = fieldtmpl.UniqueParameterName("ip", used)

	if errExpr == "" {
		errExpr = `fmt.Errorf("invalid IP %q", ` + data.RangeVar + ")"
	}

	data.ErrExpr = errExpr
	if appendMode {
		data.DocSummary = "appends to the " + field.FieldName + " option."
	}

	return fieldtmpl.Execute(netIPParseTemplate, &data)
}
