// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package generator_test

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/generator"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

type increment struct{}

func (increment) Meta() plugin.Meta             { return plugin.Meta{} }
func (increment) Key() string                   { return "increment" }
func (increment) Phase() plugin.Phase           { return plugin.PhaseTransform }
func (increment) CanHandle(model.OptField) bool { return true }
func (increment) Generate(_ plugin.GenerationContext, _ model.OptField, in string) plugin.ModifierResult {
	return plugin.ModifierResult{Code: []string{"next := " + in + " + 1"}, OutputVar: "next"}
}

type double struct{}

func (double) Meta() plugin.Meta             { return plugin.Meta{} }
func (double) Key() string                   { return "double" }
func (double) Phase() plugin.Phase           { return plugin.PhasePostProcess }
func (double) CanHandle(model.OptField) bool { return true }
func (double) Generate(_ plugin.GenerationContext, _ model.OptField, in string) plugin.ModifierResult {
	return plugin.ModifierResult{Code: []string{"result := " + in + " * 2"}, OutputVar: "result"}
}

func TestGeneratedPipelineCompilesAndUsesTransformedValue(t *testing.T) {
	t.Parallel()
	r := plugin.NewRegistry()
	require.NoError(t, builtin.Register(r))
	require.NoError(t, r.Register(increment{}, double{}))
	content, err := generator.New(r, plugin.NewPipeline(r)).Generate(model.GenerateInput{
		PackageName: "sample", TypeName: "options", OptionType: "Option", GenerateOptionType: true,
		Fields: []model.OptField{{FieldName: "value", OptionName: "Value", Type: "int", Modifiers: []string{"increment", "double"}, HasModifiers: true}},
	})
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "options_gen.go"), content, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "options.go"), []byte("package sample\ntype options struct{value int}\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "options_test.go"), []byte(`package sample
 import "testing"
 func TestValue(t *testing.T){o:=&options{};WithValue(3)(o);if o.value!=8{t.Fatalf("got %d, want 8",o.value)}}
 `), 0o600))
	// File arguments avoid resolving dependencies from the target's environment.
	command := exec.CommandContext(t.Context(), "go", "test", "options.go", "options_gen.go", "options_test.go")
	command.Dir = dir
	out, err := command.CombinedOutput()
	require.NoError(t, err, "%s", out)
}

func TestAllFieldHandlersUsePipeline(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, typ, elem, input string
		isSlice                bool
		mandatory              bool
		metadata               map[string]string
		mods                   []string
		settings               model.RenderSettings
		// skipped marks handlers that must not emit an option function.
		skipped bool
	}{
		{name: "scalar", typ: "int", input: "int"},
		{name: "string", typ: "string", input: "string"},
		{name: "string plain", typ: "string", input: "string", settings: model.RenderSettings{StringSetter: model.StringSetterPlain}},
		{name: "bool", typ: "bool", input: "bool"},
		{name: "bool invert", typ: "bool", input: "bool", mods: []string{"invert"}},
		{name: "bool param", typ: "bool", input: "bool", mods: []string{"param"}},
		{name: "slice", typ: "[]int", elem: "int", input: "[]int", isSlice: true},
		{name: "append", typ: "[]int", elem: "int", input: "[]int", isSlice: true, metadata: map[string]string{"append": "true"}},
		{name: "ip", typ: "[]net.IP", elem: "net.IP", input: "[]net.IP", isSlice: true, metadata: map[string]string{"parseip": "true"}},
		{name: "manual", typ: "int", input: "int", metadata: map[string]string{"manual": "true"}, skipped: true},
		{name: "mandatory", typ: "int", input: "int", mandatory: true, skipped: true},
	} {
		for _, returnsError := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/returns_error=%t", tc.name, returnsError), func(t *testing.T) {
				t.Parallel()
				r := plugin.NewRegistry()
				require.NoError(t, builtin.Register(r))
				require.NoError(t, r.Register(copyValue{}, terminal{}))
				settings := tc.settings
				settings.TypeName, settings.OptionType, settings.GenerateOptionType = "options", "Option", true
				settings.OptionReturnsError = returnsError
				field := model.OptField{FieldName: "value", OptionName: "Value", Type: tc.typ, ElemType: tc.elem, IsSlice: tc.isSlice,
					Metadata: tc.metadata, Mandatory: tc.mandatory, Modifiers: append(slices.Clone(tc.mods), "copyvalue"), HasModifiers: true}
				source := "package sample\n"
				if tc.name == "ip" {
					source += "import \"net\"\n"
				}
				source += "type options struct{value " + tc.input + "}"
				fset := token.NewFileSet()
				in, err := parser.ParseFile(fset, "options.go", source, 0)
				require.NoError(t, err)
				typeCheck := func(name string, field model.OptField) string {
					t.Helper()
					content, err := generator.New(r, plugin.NewPipeline(r)).Generate(model.GenerateInput{PackageName: "sample",
						RenderSettings: settings, Fields: []model.OptField{field}})
					require.NoError(t, err)
					out, err := parser.ParseFile(fset, name, content, 0)
					require.NoError(t, err)
					config := types.Config{Importer: importer.Default()}
					_, err = config.Check("sample", fset, []*ast.File{in, out}, nil)
					require.NoError(t, err, "%s", content)
					return string(content)
				}
				content := typeCheck("options_gen.go", field)
				if tc.skipped {
					require.NotContains(t, content, "o.value =", "skipped handlers must not assign the field")
					require.NotContains(t, content, "copied :=")
					return
				}
				require.Contains(t, content, "copied :=")
				require.Regexp(t, `o\.value = .*copied`, content, "the field is assigned from the pipeline output")

				field.Modifiers = append(slices.Clone(tc.mods), "terminal")
				content = typeCheck("terminal_gen.go", field)
				require.NotContains(t, content, "o.value =", "a Final modifier ends the option before the assignment")
			})
		}
	}
}

type copyValue struct{}

func (copyValue) Meta() plugin.Meta             { return plugin.Meta{} }
func (copyValue) Key() string                   { return "copyvalue" }
func (copyValue) Phase() plugin.Phase           { return plugin.PhasePostProcess }
func (copyValue) CanHandle(model.OptField) bool { return true }
func (copyValue) Generate(_ plugin.GenerationContext, _ model.OptField, in string) plugin.ModifierResult {
	return plugin.ModifierResult{Code: []string{"copied := " + in}, OutputVar: "copied"}
}

type terminal struct{}

func (terminal) Meta() plugin.Meta             { return plugin.Meta{} }
func (terminal) Key() string                   { return "terminal" }
func (terminal) Phase() plugin.Phase           { return plugin.PhaseCheck }
func (terminal) CanHandle(model.OptField) bool { return true }
func (terminal) Generate(ctx plugin.GenerationContext, _ model.OptField, _ string) plugin.ModifierResult {
	if ctx.OptionReturnsError {
		return plugin.ModifierResult{Code: []string{"return nil"}, Final: true}
	}
	return plugin.ModifierResult{Code: []string{"return"}, Final: true}
}
