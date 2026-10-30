// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package source

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/altessa-s/go-optgen/model"

	"golang.org/x/tools/go/packages"

	optparser "github.com/altessa-s/go-optgen/internal/parser"
)

// Reader reads Go declarations from disk.
//
// It loads the package with go/packages so field types are type-checked; type
// errors, such as references to options that are not generated yet, are
// tolerated. When the directory cannot be loaded as a package (for example
// outside a module), it falls back to parsing every file without type
// information.
//
// When File is set, only that file of the directory is searched for the
// struct; the other files still take part in type checking.
type Reader struct {
	File string
}

// Parse reads packages in deterministic order and extracts the target type.
func (r Reader) Parse(ctx context.Context, dir, typeName string, all bool) ([]model.SourcePackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	loaded, typed, err := load(ctx, dir)
	if err != nil {
		return nil, err
	}
	if len(loaded) == 0 {
		return nil, fmt.Errorf("no packages found in %s", dir)
	}
	var out []model.SourcePackage
	for _, pkg := range loaded {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Skip generated files, including previous optgen output, by their
		// "Code generated ... DO NOT EDIT." header rather than by file name.
		maps.DeleteFunc(pkg.ast.Files, func(name string, f *ast.File) bool {
			return ast.IsGenerated(f) || (r.File != "" && r.File != "." && filepath.Base(name) != r.File)
		})
		var resolver optparser.TypeResolver
		if typed {
			resolver = typeFacts{info: pkg.info}
		}
		result, err := optparser.FindOptFields(pkg.ast, typeName, all, resolver)
		if err != nil {
			return nil, err
		}
		out = append(out, model.SourcePackage{Name: pkg.ast.Name, Fields: result.Fields, Imports: result.Imports, GenericInfo: result.GenericInfo,
			NormalizationMethod: optparser.FindNormalizationMethod(pkg.ast, typeName), IsSetField: result.IsSetField})
	}
	return out, nil
}

// loadedPackage is a parsed package and, when type-checked, its type info.
type loadedPackage struct {
	ast  *ast.Package //nolint:staticcheck // The parser API intentionally operates on go/ast packages.
	info *types.Info
}

// load type-checks dir with go/packages, or parses it without types when it
// cannot be loaded as a package. It reports whether type info is available.
func load(ctx context.Context, dir string) ([]loadedPackage, bool, error) {
	fset := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{
		Context: ctx,
		Dir:     dir,
		Fset:    fset,
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
	}, ".")
	// Compile errors, such as references to options not generated yet, are
	// expected; fall back only when go list resolved no package files.
	if err != nil || len(pkgs) != 1 || len(pkgs[0].Syntax) == 0 {
		parsed, err := parseDir(dir)
		return parsed, false, err
	}
	pkg := pkgs[0]
	for _, e := range pkg.Errors {
		if e.Kind == packages.ParseError {
			return nil, false, fmt.Errorf("failed to parse directory: %w", errors.New(e.Error()))
		}
	}
	files := make(map[string]*ast.File, len(pkg.Syntax))
	for _, f := range pkg.Syntax {
		files[filepath.Base(fset.File(f.Pos()).Name())] = f
	}
	return []loadedPackage{{ast: &ast.Package{Name: pkg.Name, Files: files}, info: pkg.TypesInfo}}, true, nil //nolint:staticcheck // See loadedPackage.
}

// parseDir parses every non-test file of dir without type information.
func parseDir(dir string) ([]loadedPackage, error) {
	pkgs, err := parser.ParseDir(token.NewFileSet(), dir, func(fi os.FileInfo) bool { //nolint:staticcheck // Source scanning intentionally uses AST packages.
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory: %w", err)
	}
	out := make([]loadedPackage, 0, len(pkgs))
	for _, name := range slices.Sorted(maps.Keys(pkgs)) {
		out = append(out, loadedPackage{ast: pkgs[name]})
	}
	return out, nil
}

// typeFacts resolves field type facts from type-checked syntax.
type typeFacts struct{ info *types.Info }

// TypeFacts reports the facts of expr's type, or false for unknown types and
// type parameters, whose properties depend on the instantiation.
func (r typeFacts) TypeFacts(expr ast.Expr) (optparser.TypeFacts, bool) {
	t := r.info.TypeOf(expr)
	if t == nil {
		return optparser.TypeFacts{}, false
	}
	if _, ok := t.(*types.TypeParam); ok {
		return optparser.TypeFacts{}, false
	}
	facts := optparser.TypeFacts{Interface: types.IsInterface(t), Comparable: types.Comparable(t)}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		if u.Kind() == types.Invalid {
			return optparser.TypeFacts{}, false
		}
		facts.Numeric = u.Info()&types.IsNumeric != 0 && u.Info()&types.IsComplex == 0
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature, *types.Interface:
		facts.Nilable = true
	}
	return facts, true
}
