// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package generator_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/generator"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin"
	"github.com/altessa-s/go-optgen/internal/semantic"
	"github.com/altessa-s/go-optgen/internal/source"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"

	optparser "github.com/altessa-s/go-optgen/internal/parser"
)

func TestGenerator_Golden_Default(t *testing.T) {
	t.Parallel()
	runGeneratorGolden(t, false, "testdata/basic", "basic", "options_gen.golden")
}

func TestGenerator_Golden_OptionError(t *testing.T) {
	t.Parallel()
	runGeneratorGolden(t, true, "testdata/basic", "basic", "options_gen_error.golden")
}

func TestGenerator_Golden_Stages45(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		settings model.RenderSettings
		golden   string
	}{
		{name: "default", golden: "options_gen.golden"},
		{name: "option error", settings: model.RenderSettings{OptionReturnsError: true}, golden: "options_gen_error.golden"},
		{name: "check failure skip", settings: model.RenderSettings{CheckFailure: model.CheckFailureSkip}, golden: "options_gen_skip.golden"},
		{name: "with isset", settings: model.RenderSettings{WithIsSet: true}, golden: "options_gen_isset.golden"},
		{name: "validate", settings: model.RenderSettings{Validate: true}, golden: "options_gen_validate.golden"},
		{name: "validate option error", settings: model.RenderSettings{Validate: true, OptionReturnsError: true}, golden: "options_gen_validate_error.golden"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := generateStages45(t, tc.settings)
			typeCheckGenerated(t, "testdata/stages45", got)

			goldenPath := filepath.Join(filepath.Dir(testFile(t)), "testdata/stages45", tc.golden)
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				require.NoError(t, os.WriteFile(goldenPath, got, 0o644))
			}

			want, err := os.ReadFile(goldenPath)
			require.NoError(t, err)
			require.Equal(t, string(want), string(got))
		})
	}
}

func generateStages45(tb testing.TB, settings model.RenderSettings) []byte {
	tb.Helper()
	dir := filepath.Join(filepath.Dir(testFile(tb)), "testdata/stages45")
	packages, err := (source.Reader{}).Parse(tb.Context(), dir, "options", false)
	require.NoError(tb, err)
	require.Len(tb, packages, 1)
	pkg := packages[0]
	settings.TypeName = "options"
	r := testRegistry()
	prepared, err := semantic.New(r).PrepareInput(model.GenerateInput{
		RenderSettings: settings, Fields: pkg.Fields, StructFields: pkg.StructFields,
		IsSetField: pkg.IsSetField, GenericInfo: pkg.GenericInfo,
	})
	require.NoError(tb, err)

	in := model.GenerateInput{
		RenderSettings: settings, PackageName: pkg.Name, Fields: prepared.Fields, StructFields: prepared.StructFields,
		Imports: pkg.Imports, GenericInfo: pkg.GenericInfo, IsSetField: pkg.IsSetField,
	}
	in.OptionType = "Option"
	in.GenerateOptionType, in.GenerateDefaultFunc, in.GenerateNewFunc = true, true, true
	in.DefaultFuncName, in.NewFuncName = "defaultOptions", "newOptions"
	got, err := generator.New(r, plugin.NewPipeline(r)).Generate(in)
	require.NoError(tb, err)

	return got
}

func testFile(tb testing.TB) string {
	tb.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	return file
}

func TestGenerator_Golden_Collections(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		settings model.RenderSettings
		golden   string
	}{
		{name: "default", golden: "options_gen.golden"},
		{name: "option error", settings: model.RenderSettings{OptionReturnsError: true}, golden: "options_gen_error.golden"},
		{name: "prefix and isset slice parameters", settings: model.RenderSettings{FuncPrefix: "Set", WithIsSet: true, SliceSetter: model.SliceSetterSlice}, golden: "options_gen_prefix.golden"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := runGeneratorGoldenSettings(t, "testdata/collections", "collections", tc.golden, tc.settings)
			typeCheckGenerated(t, "testdata/collections", got)
		})
	}
}

func TestGenerator_CollectionsCloneTiming(t *testing.T) {
	t.Parallel()

	generated := runGeneratorGoldenSettings(t, "testdata/collections", "collections", "options_gen.golden", model.RenderSettings{})
	dir := t.TempDir()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	fixture := filepath.Join(filepath.Dir(thisFile), "testdata/collections")

	input, err := os.ReadFile(filepath.Join(fixture, "input.go"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module collectiontest\n\ngo 1.26\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "input.go"), input, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "options_gen.go"), generated, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "clone_test.go"), []byte(`package collections

import "testing"

func TestCloneTiming(t *testing.T) {
	input := []string{"before"}
	clone := WithReplace[string](input...)
	eager := WithEager[string](input...)
	input[0] = "after"

	var cloneOptions options[string]
	clone(&cloneOptions)
	if cloneOptions.Replace[0] != "after" {
		t.Fatalf("clone saw %q, want mutation made before application", cloneOptions.Replace[0])
	}

	var eagerOptions options[string]
	eager(&eagerOptions)
	if eagerOptions.Eager[0] != "before" {
		t.Fatalf("clone=eager saw %q, want creation-time snapshot", eagerOptions.Eager[0])
	}

	var values options[string]
	WithStringValue[string](" ", 0)(&values)
	if value, ok := values.StringValue[" "]; !ok || value != 0 {
		t.Fatalf("put trimmed or skipped a valid zero value: %v", values.StringValue)
	}
	WithStringValue[string]("", 1)(&values)
	if len(values.StringValue) != 1 {
		t.Fatalf("put accepted a zero key: %v", values.StringValue)
	}
}
`), 0o644))

	cmd := exec.Command("go", "test", "-count=1", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "compiled clone behavior: %s", output)
}

// The typed fixture covers field types that must be rendered verbatim (fixed
// arrays, func signatures, anonymous structs with tags, interface literals)
// and is type-checked together with the generated options.
func TestGenerator_Golden_Typed(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name               string
		optionReturnsError bool
		golden             string
	}{
		{name: "default", golden: "options_gen.golden"},
		{name: "option error", optionReturnsError: true, golden: "options_gen_error.golden"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := runGeneratorGolden(t, tc.optionReturnsError, "testdata/typed", "typed", tc.golden)
			typeCheckGenerated(t, "testdata/typed", got)
		})
	}
}

// typeCheckGenerated type-checks the fixture sources together with the
// generated file, so the generated code must compile against them.
func typeCheckGenerated(tb testing.TB, relDir string, generated []byte) {
	tb.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(tb, ok, "runtime.Caller failed")

	dir := filepath.Join(filepath.Dir(thisFile), relDir)

	fset := token.NewFileSet()
	input, err := parser.ParseFile(fset, filepath.Join(dir, "input.go"), nil, 0)
	require.NoError(tb, err, "parse input")

	if strings.Contains(string(generated), "type optionsFieldSet uint64") {
		inputBytes, readErr := os.ReadFile(filepath.Join(dir, "input.go"))
		require.NoError(tb, readErr, "read fixture input")

		inputSource := strings.Replace(string(inputBytes), "type optionsFieldSet uint64\n\n", "", 1)
		input, err = parser.ParseFile(fset, filepath.Join(dir, "input.go"), inputSource, 0)
		require.NoError(tb, err, "parse input without generated field set type")
	}

	output, err := parser.ParseFile(fset, "options_gen.go", generated, 0)
	require.NoError(tb, err, "parse generated output")

	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	_, err = conf.Check(input.Name.Name, fset, []*ast.File{input, output}, nil)
	require.NoError(tb, err, "generated options must type-check")
}

// requireParamsNotReassigned fails when the closure returned by a generated
// option function assigns to one of that function's parameters. The closure
// captures them, and an Option value may be applied many times, concurrently:
// reassigning a parameter is a data race and makes a second application see
// the first one's result.
func requireParamsNotReassigned(tb testing.TB, generated []byte) {
	tb.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "options_gen.go", generated, 0)
	require.NoError(tb, err, "parse generated output")

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		params := map[string]bool{}

		for _, field := range fn.Type.Params.List {
			for _, name := range field.Names {
				params[name.Name] = true
			}
		}

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			lit, ok := n.(*ast.FuncLit)
			if !ok {
				return true
			}

			ast.Inspect(lit.Body, func(n ast.Node) bool {
				if assign, ok := n.(*ast.AssignStmt); ok && assign.Tok == token.ASSIGN {
					for _, lhs := range assign.Lhs {
						if id, ok := lhs.(*ast.Ident); ok && params[id.Name] {
							tb.Errorf("%s: the option closure reassigns the captured parameter %s", fn.Name.Name, id.Name)
						}
					}
				}

				return true
			})

			return false
		})
	}
}

func runGeneratorGolden(tb testing.TB, optionReturnsError bool, relDir, pkgName, goldenName string) []byte {
	tb.Helper()
	return runGeneratorGoldenSettings(tb, relDir, pkgName, goldenName, model.RenderSettings{OptionReturnsError: optionReturnsError})
}

func runGeneratorGoldenSettings(tb testing.TB, relDir, pkgName, goldenName string, settings model.RenderSettings) []byte {
	tb.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(tb, ok, "runtime.Caller failed")

	dir := filepath.Join(filepath.Dir(thisFile), relDir)

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	require.NoError(tb, err, "ParseDir")

	pkg, ok := pkgs[pkgName]
	require.True(tb, ok, "expected package %q in %s", pkgName, dir)

	result, err := optparser.FindOptFields(pkg, "options", false, nil)
	require.NoError(tb, err, "FindOptFields")

	normalizationMethod, err := optparser.FindNormalizationMethod(pkg, "options")
	require.NoError(tb, err, "FindNormalizationMethod")

	r := testRegistry()

	settings.TypeName = "options"

	isSetField := result.IsSetField

	prepared, err := semantic.New(r).PrepareInput(model.GenerateInput{RenderSettings: settings, Fields: result.Fields, IsSetField: isSetField})
	require.NoError(tb, err)

	g := generator.New(r, plugin.NewPipeline(r))
	got, err := g.Generate(model.GenerateInput{
		TypeName: "options", OptionType: "Option", GenerateOptionType: true, GenerateDefaultFunc: true, GenerateNewFunc: true,
		OptionReturnsError: settings.OptionReturnsError, FuncPrefix: settings.FuncPrefix, WithIsSet: settings.WithIsSet,
		SliceSetter: settings.SliceSetter, DefaultFuncName: "defaultOptions", NewFuncName: "newOptions",
		PackageName:         pkgName,
		Fields:              prepared.Fields,
		Imports:             result.Imports,
		NormalizationMethod: normalizationMethod,
		GenericInfo:         result.GenericInfo,
		IsSetField:          isSetField,
	})
	require.NoError(tb, err, "Generate")
	require.NotContains(tb, string(got), `"github.com/altessa-s/`, "generated code must not import altessa-s modules")
	requireParamsNotReassigned(tb, got)

	goldenPath := filepath.Join(dir, goldenName)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		require.NoError(tb, os.WriteFile(goldenPath, got, 0o644), "write golden")
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(tb, err, "read golden %s (set UPDATE_GOLDEN=1 to create)", goldenPath)

	require.Equal(tb, string(want), string(got), "golden mismatch: %s (set UPDATE_GOLDEN=1 to update)", goldenPath)

	return got
}

func testRegistry() *plugin.Registry {
	r := plugin.NewRegistry()
	if err := builtin.Register(r); err != nil {
		panic(err)
	}

	return r
}
