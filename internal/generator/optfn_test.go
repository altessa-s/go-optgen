// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license. See LICENSE.

package generator_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/generator"
	"github.com/altessa-s/go-optgen/internal/semantic"
	"github.com/altessa-s/go-optgen/internal/source"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

func TestGenerator_Optfn(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		optionError bool
		golden      string
	}{
		{name: "without option error", golden: "options_gen.golden"},
		{name: "option error", optionError: true, golden: "options_gen_error.golden"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := generateOptfn(t, tc.optionError)
			require.NoError(t, err)
			compareOptfnGolden(t, got, tc.golden)
			typeCheckOptfnGenerated(t, got)
		})
	}

	t.Run("error target requires option error", func(t *testing.T) {
		t.Parallel()
		_, err := generateOptfnErrorFixture(t, false)
		require.ErrorContains(t, err, "field result option Result target makeText returns error; enable --option-error")
	})
	t.Run("option error", func(t *testing.T) {
		t.Parallel()
		got, err := generateOptfnErrorFixture(t, true)
		require.NoError(t, err, "%s", got)
		compareOptfnGolden(t, got, "options_gen_target_error.golden")
		typeCheckGenerated(t, "testdata/optfn/testdata_error", got)
	})
}

func TestGeneratorOptfnParameterDoesNotShadowTargetImport(t *testing.T) {
	t.Parallel()

	r := testRegistry()
	field := model.OptField{FieldName: "value", OptionName: "Value", Type: "codec.Result", SetterTarget: &model.SetterTarget{
		Expression: "codec.Parse", Imports: []model.GeneratedImport{{Path: "example.com/codec", Alias: "codec"}},
		Parameters: []model.FuncParameter{{Name: "codec", Type: "string"}},
	}}
	prepared, err := semantic.New(r).Prepare([]model.OptField{field})
	require.NoError(t, err)
	generated, err := generator.New(r, plugin.NewPipeline(r)).Generate(model.GenerateInput{
		TypeName: "options", OptionType: "Option", PackageName: "sample", Fields: prepared,
	})
	require.NoError(t, err)
	require.Contains(t, string(generated), "codecArg string")
	require.Contains(t, string(generated), "codec.Parse(codecArg)")
}

func TestGeneratorOptfnParametersDoNotShadowUnaliasedImportsOrFunctions(t *testing.T) {
	t.Parallel()

	r := testRegistry()
	fields := []model.OptField{
		{FieldName: "value", OptionName: "Value", Type: "string", SetterTarget: &model.SetterTarget{
			Expression: "fmt.Sprint", Imports: []model.GeneratedImport{{Path: "fmt"}},
			Parameters: []model.FuncParameter{{Name: "fmt", Type: "string"}},
		}},
		{FieldName: "text", OptionName: "Text", Type: "string", SetterTarget: &model.SetterTarget{
			Expression: "makeText", Parameters: []model.FuncParameter{{Name: "makeText", Type: "string"}},
		}},
	}
	prepared, err := semantic.New(r).Prepare(fields)
	require.NoError(t, err)
	generated, err := generator.New(r, plugin.NewPipeline(r)).Generate(model.GenerateInput{
		TypeName: "options", OptionType: "Option", PackageName: "sample", Fields: prepared,
	})
	require.NoError(t, err)
	require.Contains(t, string(generated), "fmtArg string")
	require.Contains(t, string(generated), "fmt.Sprint(fmtArg)")
	require.Contains(t, string(generated), "makeTextArg string")
	require.Contains(t, string(generated), "makeText(makeTextArg)")

	fset := token.NewFileSet()
	input, err := parser.ParseFile(fset, "input.go", "package sample\ntype options struct { value string; text string }\ntype Option func(*options)\nfunc makeText(makeText string) string { return makeText }", 0)
	require.NoError(t, err)
	output, err := parser.ParseFile(fset, "options_gen.go", generated, 0)
	require.NoError(t, err)
	_, err = (&types.Config{Importer: importer.Default()}).Check("sample", fset, []*ast.File{input, output}, nil)
	require.NoError(t, err, "generated setters must type-check with unaliased import and function-name collisions")
}

func TestGeneratorOptfnParametersAvoidGeneratedBodyIdentifiers(t *testing.T) {
	t.Parallel()

	r := testRegistry()
	field := model.OptField{FieldName: "retry", OptionName: "Retry", Type: "RetryConfig", Metadata: map[string]string{
		"enables": "active", "resets": "other",
	}, SetterTarget: &model.SetterTarget{Expression: "configureRetry", ReturnsError: false, Parameters: []model.FuncParameter{
		{Name: "options", Type: "string"}, {Name: "error", Type: "string"}, {Name: "new", Type: "string"},
		{Name: "make", Type: "string"}, {Name: "len", Type: "string"},
	}}}
	prepared, err := semantic.New(r).Prepare([]model.OptField{field})
	require.NoError(t, err)

	settings := model.RenderSettings{TypeName: "options", OptionType: "Option", OptionReturnsError: true, WithIsSet: true}
	generated, err := generator.New(r, plugin.NewPipeline(r)).Generate(model.GenerateInput{
		RenderSettings: settings,
		PackageName:    "sample", Fields: prepared, IsSetField: "set",
		StructFields: []model.StructField{{Name: "active", Type: "bool"}, {Name: "other", Type: "int"}, {Name: "set", Type: "uint64"}},
	})
	require.NoError(t, err)

	fset := token.NewFileSet()
	input, err := parser.ParseFile(fset, "input.go", "package sample\ntype RetryConfig int\ntype options struct { retry RetryConfig; active bool; other int; set uint64 }\ntype Option func(*options) error\nfunc configureRetry(options, error, new, make, len string) RetryConfig { return 0 }", 0)
	require.NoError(t, err)
	output, err := parser.ParseFile(fset, "options_gen.go", generated, 0)
	require.NoError(t, err)
	_, err = (&types.Config{Importer: importer.Default()}).Check("sample", fset, []*ast.File{input, output}, nil)
	require.NoError(t, err, "generated setters must not shadow identifiers used in their body")

	generatedText := string(generated)
	require.Less(t, strings.Index(generatedText, "if optgenErr != nil"), strings.Index(generatedText, "o.set |= 1 <<"))
	require.Contains(t, string(generated), "optionsArg string")
	require.Contains(t, string(generated), "errorArg string")
	require.Contains(t, string(generated), "newArg string")
	require.Contains(t, string(generated), "makeArg string")
	require.Contains(t, string(generated), "lenArg string")
}

func TestGenerator_OptionDocUsesPreparedOptionPrefix(t *testing.T) {
	t.Parallel()

	r := testRegistry()
	fields, err := semantic.New(r, semantic.WithPrefix("Client")).Prepare([]model.OptField{{
		FieldName: "name", OptionName: "Name", Type: "string", Doc: "SetName selects the client name.",
	}})
	require.NoError(t, err)

	g := generator.New(r, plugin.NewPipeline(r))
	got, err := g.Generate(model.GenerateInput{
		TypeName:            "options",
		OptionType:          "Option",
		FuncPrefix:          "Set",
		GenerateOptionType:  true,
		GenerateDefaultFunc: true,
		GenerateNewFunc:     true,
		DefaultFuncName:     "defaultOptions",
		NewFuncName:         "newOptions",
		PackageName:         "sample",
		Fields:              fields,
	})
	require.NoError(t, err)
	compareOptionDocsGolden(t, got, "options_gen_prefix.golden")
	typeCheckGeneratedSource(t, got)
}

func TestGeneratorOptfnOptionDocsWithOptionAndFunctionPrefixes(t *testing.T) {
	t.Parallel()

	r := testRegistry()
	fields, err := semantic.New(r, semantic.WithPrefix("HTTP")).Prepare([]model.OptField{
		{FieldName: "header", OptionName: "Header", HasOptionDoc: true, OptionDoc: "SetHeader sets the HTTP header.", Type: "string", SetterTarget: &model.SetterTarget{Expression: "makeHeader", Parameters: []model.FuncParameter{{Name: "value", Type: "string"}}}, Metadata: map[string]string{"multiopt": "true"}},
		{FieldName: "header", OptionName: "Fallback", HasOptionDoc: true, OptionDoc: "SetFallback sets the fallback header.", Type: "string", SetterTarget: &model.SetterTarget{Expression: "makeHeader", Parameters: []model.FuncParameter{{Name: "value", Type: "string"}}}, Metadata: map[string]string{"multiopt": "true"}},
	})
	require.NoError(t, err)
	got, err := generator.New(r, plugin.NewPipeline(r)).Generate(model.GenerateInput{
		TypeName: "options", OptionType: "Option", PackageName: "sample", FuncPrefix: "Set", Fields: fields,
	})
	require.NoError(t, err)
	require.Contains(t, string(got), "// SetHTTPHeader sets the HTTP header.")
	require.Contains(t, string(got), "// SetHTTPFallback sets the fallback header.")
}

func typeCheckGeneratedSource(tb testing.TB, generated []byte) {
	tb.Helper()

	fset := token.NewFileSet()
	input, err := parser.ParseFile(fset, "options.go", "package sample\ntype options struct { name string }", 0)
	require.NoError(tb, err)
	file, err := parser.ParseFile(fset, "options_gen.go", generated, 0)
	require.NoError(tb, err)
	_, err = (&types.Config{Importer: importer.Default()}).Check("sample", fset, []*ast.File{input, file}, nil)
	require.NoError(tb, err, "generated source must type-check")
}

func TestGenerator_OptionDocRequiresFinalFunctionName(t *testing.T) {
	t.Parallel()

	r := testRegistry()
	fields, err := semantic.New(r, semantic.WithPrefix("Client")).Prepare([]model.OptField{{
		FieldName: "name", OptionName: "Name", Type: "string", HasOptionDoc: true, OptionDoc: "WithClientName selects the client name.",
	}})
	require.NoError(t, err)

	g := generator.New(r, plugin.NewPipeline(r))
	_, err = g.Generate(model.GenerateInput{
		TypeName:    "options",
		OptionType:  "Option",
		FuncPrefix:  "Set",
		PackageName: "sample",
		Fields:      fields,
	})
	require.ErrorContains(t, err, "field name option ClientName: optgen:doc must start with SetClientName ")
}

func TestGenerator_OptionDocs(t *testing.T) {
	t.Parallel()

	got, err := generateOptionDocs(t)
	require.NoError(t, err)
	compareOptionDocsGolden(t, got, "options_gen.golden")
	typeCheckOptionDocsGenerated(t, got)
}

func generateOptionDocs(tb testing.TB) ([]byte, error) {
	tb.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	dir := filepath.Join(filepath.Dir(thisFile), "testdata/option_docs")
	packages, err := (source.Reader{}).Parse(tb.Context(), dir, "options", false)
	require.NoError(tb, err)
	require.Len(tb, packages, 1)

	r := testRegistry()
	fields, err := semantic.New(r).Prepare(packages[0].Fields)
	require.NoError(tb, err)

	g := generator.New(r, plugin.NewPipeline(r))

	return g.Generate(model.GenerateInput{
		TypeName:            "options",
		OptionType:          "Option",
		FuncPrefix:          "Set",
		GenerateOptionType:  true,
		GenerateDefaultFunc: true,
		GenerateNewFunc:     true,
		DefaultFuncName:     "defaultOptions",
		NewFuncName:         "newOptions",
		PackageName:         packages[0].Name,
		Fields:              fields,
		Imports:             packages[0].Imports,
		GenericInfo:         packages[0].GenericInfo,
	})
}

func typeCheckOptionDocsGenerated(tb testing.TB, generated []byte) {
	tb.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	dir := filepath.Join(filepath.Dir(thisFile), "testdata/option_docs")
	fset := token.NewFileSet()
	input, err := parser.ParseFile(fset, filepath.Join(dir, "input.go"), nil, 0)
	require.NoError(tb, err)
	output, err := parser.ParseFile(fset, "options_gen.go", generated, 0)
	require.NoError(tb, err)
	_, err = (&types.Config{Importer: importer.Default()}).Check(input.Name.Name, fset, []*ast.File{input, output}, nil)
	require.NoError(tb, err, "generated option docs fixture must type-check")
}

func generateOptfnErrorFixture(tb testing.TB, optionError bool) ([]byte, error) {
	tb.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	dir := filepath.Join(filepath.Dir(thisFile), "testdata/optfn/testdata_error")

	return generateOptfnDir(tb, dir, optionError)
}

func generateOptfn(tb testing.TB, optionError bool) ([]byte, error) {
	tb.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	dir := filepath.Join(filepath.Dir(thisFile), "testdata/optfn")

	return generateOptfnDir(tb, dir, optionError)
}

func generateOptfnDir(tb testing.TB, dir string, optionError bool) ([]byte, error) {
	tb.Helper()
	sourcePackages, err := (source.Reader{}).Parse(tb.Context(), dir, "options", false)
	require.NoError(tb, err)
	require.Len(tb, sourcePackages, 1)
	pkg := sourcePackages[0]
	r := testRegistry()
	fields, err := semantic.New(r).Prepare(pkg.Fields)
	require.NoError(tb, err)

	settings := model.RenderSettings{
		TypeName: "options", OptionType: "Option", GenerateOptionType: true, GenerateDefaultFunc: true, GenerateNewFunc: true,
		OptionReturnsError: optionError, FuncPrefix: "Set", WithIsSet: true, DefaultFuncName: "defaultOptions", NewFuncName: "newOptions",
	}
	g := generator.New(r, plugin.NewPipeline(r))

	return g.Generate(model.GenerateInput{
		RenderSettings: settings, PackageName: pkg.Name, Fields: fields, Imports: pkg.Imports,
		GenericInfo: pkg.GenericInfo, IsSetField: pkg.IsSetField,
	})
}

func typeCheckOptfnGenerated(tb testing.TB, generated []byte) {
	tb.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	dir := filepath.Join(filepath.Dir(thisFile), "testdata/optfn")
	fset := token.NewFileSet()
	input, err := parser.ParseFile(fset, filepath.Join(dir, "input.go"), nil, 0)
	require.NoError(tb, err)
	output, err := parser.ParseFile(fset, "options_gen.go", generated, 0)
	require.NoError(tb, err)
	supportFile, err := parser.ParseFile(fset, filepath.Join(dir, "support/support.go"), nil, 0)
	require.NoError(tb, err)
	support, err := (&types.Config{Importer: importer.Default()}).Check("optfn.test/support", fset, []*ast.File{supportFile}, nil)
	require.NoError(tb, err)
	_, err = (&types.Config{Importer: optfnImporter{support: support, fallback: importer.Default()}}).Check(input.Name.Name, fset, []*ast.File{input, output}, nil)
	require.NoError(tb, err, "generated options must type-check")
}

type optfnImporter struct {
	support  *types.Package
	fallback types.Importer
}

func (i optfnImporter) Import(path string) (*types.Package, error) {
	if path == "optfn.test/support" {
		return i.support, nil
	}

	return i.fallback.Import(path)
}

func compareOptfnGolden(tb testing.TB, got []byte, golden string) {
	tb.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	dir := filepath.Join(filepath.Dir(thisFile), "testdata/optfn")

	goldenPath := filepath.Join(dir, golden)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		require.NoError(tb, os.WriteFile(goldenPath, got, 0o644))
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(tb, err)
	require.Equal(tb, string(want), string(got))
}

func compareOptionDocsGolden(tb testing.TB, got []byte, golden string) {
	tb.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	dir := filepath.Join(filepath.Dir(thisFile), "testdata/option_docs")
	path := filepath.Join(dir, golden)

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		require.NoError(tb, os.WriteFile(path, got, 0o644))
	}

	want, err := os.ReadFile(path)
	require.NoError(tb, err)
	require.Equal(tb, string(want), string(got))
}
