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

// stubPipeline returns a fixed pipeline result.
type stubPipeline plugin.PipelineResult

func (stubPipeline) BuildChecks(plugin.GenerationContext, model.OptField, string) []string {
	return nil
}

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
		TypeParamsDecl: "[T any]", TypeParamsNames: "[T]", FuncPrefix: "With"}, data.BaseData)

	require.Empty(t, fieldtmpl.NewData(ctx, field, stubPipeline{ValueVar: "v"}, "v").PipelineCode)
}
