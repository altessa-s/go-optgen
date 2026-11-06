// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldtmpl

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// Common defines the shared "optionReturn" template that renders the
// trailing return of an option function.
const Common = `
{{- define "optionDoc" -}}
{{- if .FullDocComment}}{{.FullDocComment}}{{else}}// {{.FuncPrefix}}{{.OptionName}} {{.DocSummary}}{{.DocComment}}{{end}}
{{- end -}}
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
// Option templates must receive a pointer to [BaseData], [Data] or a struct
// embedding one of them: only then does Execute resolve and validate the
// option documentation (the field doc used as the full doc, or optgen:doc
// directives) before rendering.
func Execute(tmpl *template.Template, data any) (plugin.GeneratedCode, error) {
	if docData, ok := data.(optionDocData); ok {
		base := docData.optionDocBase()

		functionName := base.FuncPrefix + base.OptionName

		if base.HasOptionDoc {
			if !strings.HasPrefix(base.OptionDoc, functionName+" ") {
				return plugin.GeneratedCode{}, fmt.Errorf("field %s option %s: optgen:doc must start with %s ",
					base.FieldName, base.OptionName, functionName)
			}

			base.FullDocComment = fullDocComment(base.OptionDoc)
		} else if base.Metadata["multiopt"] != "true" && strings.HasPrefix(base.Doc, functionName+" ") {
			base.FullDocComment = fullDocComment(base.Doc)
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return plugin.GeneratedCode{}, err
	}

	return plugin.GeneratedCode{Code: buf.String()}, nil
}

type optionDocData interface {
	optionDocBase() *BaseData
}

func (b *BaseData) optionDocBase() *BaseData { return b }

func fullDocComment(doc string) string {
	var b strings.Builder

	for line := range strings.Lines(doc) {
		line = strings.TrimRight(line, " \t\r\n")

		b.WriteString("\n//")

		if line != "" {
			if !strings.HasPrefix(line, "\t") {
				b.WriteByte(' ')
			}

			b.WriteString(line)
		}
	}

	return b.String()
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
	SliceParam         bool   // Slice options take a slice instead of variadic elements
	PreClosureCode     string // Code that snapshots option inputs before returning the closure
	MarkSet            string // Statement recording the applied option, or empty
	PostAssignment     string // Enables and resets rendered after the assigned value
	DocComment         string // Field documentation as comment lines following the summary, or empty
	FullDocComment     string // Complete option documentation, or empty for the generated summary
	DocSummary         string // Generated summary after the function name
}

// NewBaseData collects the template data for field from ctx.
func NewBaseData(ctx plugin.GenerationContext, field model.OptField) BaseData {
	mark := markSet(ctx, field)

	post := postAssignment(ctx, field)

	doc := field.Doc

	if field.Metadata["multiopt"] == "true" && startsWithGeneratedOptionFunction(doc, ctx, field) {
		doc = ""
	}

	return BaseData{
		OptField:           field,
		TypeName:           ctx.TypeName,
		OptionType:         ctx.OptionType,
		OptionReturnsError: ctx.OptionReturnsError,
		TypeParamsDecl:     ctx.TypeParamsDecl(),
		TypeParamsNames:    ctx.TypeParamsNames(),
		FuncPrefix:         ctx.OptionFuncPrefix(),
		KeepEmptyStrings:   ctx.KeepEmptyStrings,
		SliceParam:         ctx.SliceSetter == model.SliceSetterSlice,
		MarkSet:            mark,
		PostAssignment:     post,
		DocComment:         docComment(doc),
		DocSummary:         "sets the " + field.FieldName + " option.",
	}
}

// ParameterIdentifiers returns identifiers that an option parameter must not
// shadow in the generated function body.
func ParameterIdentifiers(ctx plugin.GenerationContext, field model.OptField, sources ...string) map[string]bool {
	used := make(map[string]bool)
	used[ctx.TypeName] = true
	used["o"] = true
	used["error"] = true
	used["false"] = true
	used["append"] = true
	used["any"] = true
	used["len"] = true
	used["make"] = true
	used["nil"] = true
	used["new"] = true
	used["true"] = true

	for _, param := range ctx.GenericInfo.TypeParams {
		used[param.Name] = true
	}

	if ctx.WithIsSet {
		used[ctx.IsSetConst(field)] = true
	}

	for target := range strings.SplitSeq(field.Metadata["resets"], "|") {
		target = strings.TrimSpace(target)

		for _, sourceField := range ctx.StructFields {
			if sourceField.Name != target {
				continue
			}

			addIdentifiers(used, sourceField.Type)

			for _, imp := range sourceField.TypeImports {
				addIdentifiers(used, imp.Alias)
			}

			break
		}
	}

	for _, source := range sources {
		addIdentifiers(used, source)
	}

	addIdentifiers(used, field.Metadata["err"])

	return used
}

func addIdentifiers(used map[string]bool, source string) {
	for i := 0; i < len(source); {
		if !isIdentifierStart(source[i]) {
			i++
			continue
		}

		start := i

		for i < len(source) && isIdentifierContinue(source[i]) {
			i++
		}

		used[source[start:i]] = true
	}
}

// UniqueParameterName returns preferred with Arg suffixes until it avoids every
// identifier referenced by the generated function body.
func UniqueParameterName(preferred string, used map[string]bool) string {
	for used[preferred] {
		preferred += "Arg"
	}

	return preferred
}

func isIdentifierStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isIdentifierContinue(c byte) bool {
	return isIdentifierStart(c) || c >= '0' && c <= '9'
}

func startsWithGeneratedOptionFunction(doc string, ctx plugin.GenerationContext, field model.OptField) bool {
	for _, candidate := range ctx.AllFields {
		if candidate.FieldName != field.FieldName {
			continue
		}

		prefix := ctx.OptionFuncPrefix()
		if candidate.HasModifier("invert") {
			prefix = ctx.OptionInvertFuncPrefix()
		}

		functionName := prefix + candidate.OptionName
		if strings.HasPrefix(doc, functionName+" ") || strings.HasPrefix(doc, functionName+"\n") {
			return true
		}
	}

	return false
}

func postAssignment(ctx plugin.GenerationContext, field model.OptField) string {
	var b strings.Builder

	fieldsByName := make(map[string]model.StructField, len(ctx.StructFields))
	for _, sourceField := range ctx.StructFields {
		fieldsByName[sourceField.Name] = sourceField
	}

	if target := strings.TrimSpace(field.Metadata["enables"]); target != "" {
		fmt.Fprintf(&b, "\t\to.%s = true\n", target)
	}

	for target := range strings.SplitSeq(field.Metadata["resets"], "|") {
		target = strings.TrimSpace(target)
		if target != "" {
			if resetField, ok := fieldsByName[target]; ok {
				for _, imp := range resetField.TypeImports {
					ctx.AddImport(imp.Path, imp.Alias)
				}
			}

			fmt.Fprintf(&b, "\t\to.%s = *new(%s)\n", target, structFieldType(ctx, target))
		}
	}

	if b.Len() == 0 {
		return ""
	}

	return b.String()
}

func structFieldType(ctx plugin.GenerationContext, name string) string {
	for _, target := range ctx.StructFields {
		if target.Name == name {
			return target.Type
		}
	}

	return "any"
}

// docComment renders doc as a separate paragraph of the option function's
// doc comment: an empty comment line followed by one comment line per line
// of doc. It returns "" when doc is empty.
func docComment(doc string) string {
	if doc == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n//")

	for line := range strings.Lines(doc) {
		line = strings.TrimRight(line, " \t\r\n")

		b.WriteString("\n//")

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "\t") {
			b.WriteByte(' ')
		}

		b.WriteString(line)
	}

	return b.String()
}

// Data adds the value pipeline output to [BaseData]: the indented pipeline
// code, the final value expression, and whether the pipeline terminated.
type Data struct {
	BaseData
	PipelineCode string
	ValueExpr    string
	InputVar     string
	Final        bool
}

// NewData runs values for field starting from inputVar.
func NewData(ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline, inputVar string) Data {
	used := ParameterIdentifiers(ctx, field, field.Type, field.ElemType, field.MapKeyType, field.MapValueType)
	inputVar = UniqueParameterName(inputVar, used)

	result := plugin.PipelineResult{ValueVar: inputVar}
	if values != nil {
		result = values.BuildPipeline(ctx, field, inputVar)
	}

	code := ""
	if result.Code != "" {
		code = "\t\t" + strings.ReplaceAll(result.Code, "\n", "\n\t\t") + "\n"
	}

	baseField := field
	if result.EarlyReturn && field.Metadata["resets"] != "" {
		baseField.Metadata = make(map[string]string, len(field.Metadata))
		for key, value := range field.Metadata {
			if key != "resets" {
				baseField.Metadata[key] = value
			}
		}
	}

	base := NewBaseData(ctx, baseField)
	base.PreClosureCode = result.PreClosureCode

	return Data{BaseData: base, PipelineCode: code, ValueExpr: result.ValueVar, InputVar: inputVar, Final: result.EarlyReturn}
}

// markSet returns the statement recording that the option of field was
// applied, followed by a separator, or "" when WithIsSet is off.
func markSet(ctx plugin.GenerationContext, field model.OptField) string {
	if !ctx.WithIsSet || ctx.IsSetField == "" {
		return ""
	}

	return "o." + ctx.IsSetField + " |= 1 << " + ctx.IsSetConst(field) + "; "
}
