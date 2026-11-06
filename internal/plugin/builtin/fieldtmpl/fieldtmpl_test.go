// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldtmpl_test

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin/fieldtmpl"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

func TestParameterIdentifiersIncludesGeneratedBodyNames(t *testing.T) {
	t.Parallel()

	ctx := plugin.NewGenerationContext("sample", model.RenderSettings{TypeName: "options", WithIsSet: true}, nil, nil,
		model.GenericInfo{TypeParams: []model.TypeParam{{Name: "T", Constraint: "any"}}})
	ctx.StructFields = []model.StructField{{Name: "other", Type: "pkg.Policy"}}
	data := fieldtmpl.NewBaseData(ctx, model.OptField{OptionName: "Retry", Metadata: map[string]string{"enables": "active", "resets": "other"}})

	used := fieldtmpl.ParameterIdentifiers(ctx, model.OptField{OptionName: "Retry", Checks: map[string]string{"min": "v", "oneof": "[v]"}}, data.PostAssignment)
	for _, name := range []string{"options", "T", "error", "len", "make", "nil", "new", "true", "optionsFieldRetry", "active", "pkg", "Policy", "v", "cloned", "deduped"} {
		require.True(t, used[name], "%s must be reserved", name)
	}
}

// --- Option templates ---

func TestMustRendersOptionReturn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		returnErr bool
		want      string
	}{
		{"with error return", true, "return nil"},
		{"without error return", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpl := fieldtmpl.Must("t", `func set() {
{{template "optionReturn" .}}
}`)
			out, err := fieldtmpl.Execute(tmpl, struct{ OptionReturnsError bool }{OptionReturnsError: tt.returnErr})
			require.NoError(t, err)

			if tt.want == "" {
				require.NotContains(t, out.Code, "return nil")
			} else {
				require.Contains(t, out.Code, tt.want)
			}
		})
	}
}

func TestMustPanicsOnInvalidSyntax(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() {
		fieldtmpl.Must("bad", `{{.Unclosed`)
	})
}

func TestExecuteReturnsError(t *testing.T) {
	t.Parallel()

	tmpl := template.Must(template.New("strict").Option("missingkey=error").Parse(`{{.DoesNotExist}}`))
	_, err := fieldtmpl.Execute(tmpl, struct{ Other string }{Other: "x"})
	require.Error(t, err)
}

func TestExecuteSuccess(t *testing.T) {
	t.Parallel()

	tmpl := template.Must(template.New("ok").Parse(`hello {{.Name}}`))
	out, err := fieldtmpl.Execute(tmpl, struct{ Name string }{Name: "world"})
	require.NoError(t, err)
	require.Equal(t, "hello world", out.Code)
	require.Empty(t, out.TypeDefs)
	require.Empty(t, out.Helpers)
}

func TestExecuteUsesFullOptionDocumentation(t *testing.T) {
	t.Parallel()

	tmpl := template.Must(template.New("doc").Parse(`{{if .FullDocComment}}{{.FullDocComment}}{{else}}summary{{.DocComment}}{{end}}`))
	base := fieldtmpl.BaseData{FieldName: "name", OptionName: "ClientName", Doc: "SetClientName selects a client.\nDetails.", FuncPrefix: "Set"}
	out, err := fieldtmpl.Execute(tmpl, &base)
	require.NoError(t, err)
	require.Equal(t, "\n// SetClientName selects a client.\n// Details.", out.Code)

	ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option"}, nil, nil, model.GenericInfo{})
	base = fieldtmpl.NewBaseData(ctx, model.OptField{FieldName: "name", OptionName: "ClientName", Doc: "A field description."})
	base.FuncPrefix = "Set"
	out, err = fieldtmpl.Execute(tmpl, &base)
	require.NoError(t, err)
	require.Equal(t, "summary\n//\n// A field description.", out.Code)
}

func TestExecuteDoesNotReuseFullFieldDocAcrossMultiOptionField(t *testing.T) {
	t.Parallel()

	tmpl := template.Must(template.New("doc").Parse(fieldtmpl.Common + `{{template "optionDoc" .}}`))
	fields := []model.OptField{
		{FieldName: "mapping", OptionName: "First", Metadata: map[string]string{"multiopt": "true"}},
		{FieldName: "mapping", OptionName: "Second", Metadata: map[string]string{"multiopt": "true"}},
	}
	ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option"}, fields, nil, model.GenericInfo{})
	base := fieldtmpl.NewBaseData(ctx, model.OptField{FieldName: "mapping", OptionName: "Second", Doc: "WithFirst sets the first mapping.", Metadata: map[string]string{"multiopt": "true"}})
	base.FuncPrefix = "With"
	out, err := fieldtmpl.Execute(tmpl, &base)
	require.NoError(t, err)
	require.NotContains(t, out.Code, "WithFirst")
}

func TestNewBaseDataKeepsOrdinaryMultiOptionDocumentation(t *testing.T) {
	t.Parallel()

	ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option"}, nil, nil, model.GenericInfo{})
	base := fieldtmpl.NewBaseData(ctx, model.OptField{FieldName: "routes", OptionName: "Exact", Doc: "With multiple routes configured, each setter updates the table.", Metadata: map[string]string{"multiopt": "true"}})
	require.Contains(t, base.DocComment, "With multiple routes configured")
}

func TestNewBaseDataKeepsEnglishMultiOptionDocumentationStartingWithWithout(t *testing.T) {
	t.Parallel()

	ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option"}, nil, nil, model.GenericInfo{})
	base := fieldtmpl.NewBaseData(ctx, model.OptField{FieldName: "timeouts", OptionName: "ReadTimeout", Doc: "Without explicit timeouts, both fields use the default deadline.", Metadata: map[string]string{"multiopt": "true"}})
	require.Contains(t, base.DocComment, "Without explicit timeouts")
}

func TestNewBaseDataFiltersInvertOptionDoc(t *testing.T) {
	t.Parallel()

	fields := []model.OptField{
		{FieldName: "second", OptionName: "First", Metadata: map[string]string{"multiopt": "true"}, HasModifiers: true, Modifiers: []string{"invert"}},
		{FieldName: "second", OptionName: "Second", Metadata: map[string]string{"multiopt": "true"}, HasModifiers: true, Modifiers: []string{"invert"}},
	}
	ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option", InvertFuncPrefix: "Without"}, fields, nil, model.GenericInfo{})
	base := fieldtmpl.NewBaseData(ctx, fields[1])
	base.Doc = "WithoutFirst disables first."
	base.FuncPrefix = "Without"
	require.NotContains(t, base.DocComment, "WithoutFirst")
}

func TestResetImportsFollowEmittedPostAssignment(t *testing.T) {
	t.Parallel()

	field := model.OptField{FieldName: "mode", Metadata: map[string]string{"resets": "client"}}
	structFields := []model.StructField{{Name: "client", Type: "*client.Client", TypeImports: []model.GeneratedImport{{Path: "example.com/client", Alias: "client"}}}}
	ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option"}, nil, nil, model.GenericInfo{})
	ctx.StructFields = structFields
	base := fieldtmpl.NewBaseData(ctx, field)
	require.Contains(t, base.PostAssignment, "*new(*client.Client)")
	require.Equal(t, []model.GeneratedImport{{Path: "example.com/client", Alias: "client"}}, ctx.GetAdditionalImports())

	ctx = plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option"}, nil, nil, model.GenericInfo{})
	ctx.StructFields = structFields
	data := fieldtmpl.NewData(ctx, field, stubPipeline{ValueVar: "v", EarlyReturn: true}, "v")
	require.Empty(t, data.PostAssignment)
	require.Empty(t, ctx.GetAdditionalImports())
}

func TestExecuteRejectsOptionDocumentationWithWrongFunctionName(t *testing.T) {
	t.Parallel()

	tmpl := template.Must(template.New("doc").Parse(`{{.FullDocComment}}`))
	base := fieldtmpl.BaseData{FieldName: "name", OptionName: "ClientName", HasOptionDoc: true, OptionDoc: "WithClientName selects a client.", FuncPrefix: "Set"}
	_, err := fieldtmpl.Execute(tmpl, &base)
	require.ErrorContains(t, err, "field name option ClientName: optgen:doc must start with SetClientName ")
}

// stubPipeline returns a fixed pipeline result.
type stubPipeline plugin.PipelineResult

func (s stubPipeline) BuildPipeline(plugin.GenerationContext, model.OptField, string) plugin.PipelineResult {
	return plugin.PipelineResult(s)
}

func TestNewDataIndentsPipelineCode(t *testing.T) {
	t.Parallel()

	ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option", OptionReturnsError: true}, nil, nil,
		model.GenericInfo{TypeParams: []model.TypeParam{{Name: "T", Constraint: "any"}}})
	field := model.OptField{FieldName: "name"}

	data := fieldtmpl.NewData(ctx, field, stubPipeline{Code: "a\nb", ValueVar: "out", EarlyReturn: true}, "v")
	require.Equal(t, "\t\ta\n\t\tb\n", data.PipelineCode)
	require.Equal(t, "out", data.ValueExpr)
	require.True(t, data.Final)
	require.Equal(t, fieldtmpl.BaseData{OptField: field, TypeName: "options", OptionType: "Option", OptionReturnsError: true,
		TypeParamsDecl: "[T any]", TypeParamsNames: "[T]", FuncPrefix: "With", DocSummary: "sets the name option."}, data.BaseData)

	require.Empty(t, fieldtmpl.NewData(ctx, field, stubPipeline{ValueVar: "v"}, "v").PipelineCode)
}

func TestNewBaseDataDocComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doc  string
		want string
	}{
		{"empty", "", ""},
		{"single line", "name is the human-readable name.", "\n//\n// name is the human-readable name."},
		{"paragraphs", "first line\nsecond line\n\nnext paragraph", "\n//\n// first line\n// second line\n//\n// next paragraph"},
		{"code block", "example:\n\n\tWithName(\"x\")", "\n//\n// example:\n//\n//\tWithName(\"x\")"},
	}

	ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option"}, nil, nil, model.GenericInfo{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := fieldtmpl.NewBaseData(ctx, model.OptField{FieldName: "name", Doc: tt.doc})
			require.Equal(t, tt.want, data.DocComment)
		})
	}
}
