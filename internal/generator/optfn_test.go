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
