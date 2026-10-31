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
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/generator"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin"
	"github.com/altessa-s/go-optgen/internal/semantic"
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

	normalizationMethod := optparser.FindNormalizationMethod(pkg, "options")

	r := testRegistry()
	result.Fields, err = semantic.New(r).Prepare(result.Fields)
	require.NoError(tb, err)
	g := generator.New(r, plugin.NewPipeline(r))
	got, err := g.Generate(model.GenerateInput{
		RenderSettings: model.RenderSettings{
			TypeName:            "options",
			OptionType:          "Option",
			GenerateOptionType:  true,
			GenerateDefaultFunc: true,
			GenerateNewFunc:     true,
			OptionReturnsError:  optionReturnsError,
			DefaultFuncName:     "defaultOptions",
			NewFuncName:         "newOptions",
		},
		PackageName:         pkgName,
		Fields:              result.Fields,
		Imports:             result.Imports,
		NormalizationMethod: normalizationMethod,
		GenericInfo:         result.GenericInfo,
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
