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

	"golang.org/x/tools/go/packages"

	"github.com/altessa-s/go-optgen/model"

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
			resolver = typeFacts{info: pkg.info, fset: pkg.fset, types: pkg.types}
		}

		result, err := optparser.FindOptFields(pkg.ast, typeName, all, resolver)
		if err != nil {
			return nil, err
		}

		normalization, err := optparser.FindNormalizationMethod(pkg.ast, typeName)
		if err != nil {
			return nil, err
		}

		out = append(out, model.SourcePackage{Name: pkg.ast.Name, Fields: result.Fields, Imports: result.Imports, GenericInfo: result.GenericInfo,
			NormalizationMethod: normalization, IsSetField: result.IsSetField, Fset: pkg.fset, Types: pkg.types})
	}

	return out, nil
}

// loadedPackage is a parsed package and, when type-checked, its type info.
type loadedPackage struct {
	ast   *ast.Package //nolint:staticcheck // The parser API intentionally operates on go/ast packages.
	info  *types.Info
	fset  *token.FileSet
	types *types.Package
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

	return []loadedPackage{{
		ast:  &ast.Package{Name: pkg.Name, Files: files}, //nolint:staticcheck // See loadedPackage.
		info: pkg.TypesInfo, fset: fset, types: pkg.Types,
	}}, true, nil //nolint:staticcheck // See loadedPackage.
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
type typeFacts struct {
	info  *types.Info
	fset  *token.FileSet
	types *types.Package
}

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

const maxSetterResults = 2

// ResolveSetterTarget resolves an optfn function or method and renders its parameter signature.
func (r typeFacts) ResolveSetterTarget(
	field ast.Expr,
	pos token.Pos,
	expression string,
	method bool,
	imports map[string]model.ImportInfo,
) (model.SetterTarget, error) {
	if r.types == nil || r.info == nil || r.fset == nil {
		return model.SetterTarget{}, fmt.Errorf("type information is unavailable")
	}

	signature, targetPackage, err := r.setterSignature(field, pos, expression, method)
	if err != nil {
		return model.SetterTarget{}, err
	}

	if signature.TypeParams() != nil && signature.TypeParams().Len() > 0 {
		return model.SetterTarget{}, fmt.Errorf("generic target must be explicitly instantiated")
	}

	if signature.Results().Len() < 1 || signature.Results().Len() > maxSetterResults {
		return model.SetterTarget{}, fmt.Errorf("target must return a value and optionally an error")
	}

	fieldType := r.info.TypeOf(field)
	if fieldType == nil || hasInvalidType(signature) {
		return model.SetterTarget{}, fmt.Errorf("target signature contains an unresolved type")
	}

	resultType := signature.Results().At(0).Type()
	if !types.AssignableTo(resultType, fieldType) {
		return model.SetterTarget{}, fmt.Errorf("result is not assignable to field type")
	}

	returnsError := signature.Results().Len() == maxSetterResults && types.Identical(signature.Results().At(1).Type(), types.Universe.Lookup("error").Type())
	if signature.Results().Len() == maxSetterResults && !returnsError {
		return model.SetterTarget{}, fmt.Errorf("second result must be error")
	}

	target := model.SetterTarget{Expression: expression, Method: method, ReturnsError: returnsError}
	importsForTarget := newTargetImports(r.types, imports, expression, method, targetPackage)
	target.Parameters = setterParameters(signature, importsForTarget.qualifier)
	target.Imports = importsForTarget.imports

	return target, nil
}

func (r typeFacts) setterSignature(field ast.Expr, pos token.Pos, expression string, method bool) (*types.Signature, *types.Package, error) {
	if method {
		fieldType := r.info.TypeOf(field)
		if fieldType == nil {
			return nil, nil, fmt.Errorf("field type is unresolved")
		}

		name := strings.TrimPrefix(expression, ".")
		obj, _, _ := types.LookupFieldOrMethod(fieldType, true, r.types, name)

		fn, ok := obj.(*types.Func)
		if !ok {
			return nil, nil, fmt.Errorf("method %q was not found", name)
		}

		sig, _ := fn.Type().(*types.Signature)

		return sig, fn.Pkg(), nil
	}

	value, err := types.Eval(r.fset, r.types, pos, expression)
	if err != nil {
		return nil, nil, err
	}

	sig, ok := value.Type.(*types.Signature)
	if !ok {
		return nil, nil, fmt.Errorf("target is not a function")
	}

	return sig, nil, nil
}

type targetImportResolver struct {
	imports []model.GeneratedImport
	aliases map[string]string
	current string
}

func newTargetImports(
	current *types.Package,
	imports map[string]model.ImportInfo,
	expression string,
	method bool,
	targetPackage *types.Package,
) targetImportResolver {
	resolver := targetImportResolver{aliases: make(map[string]string), current: current.Path()}

	for _, imp := range imports {
		if imp.Alias != "" && imp.Alias != "_" && imp.Alias != "." {
			resolver.aliases[imp.Path] = imp.Alias
		}
	}

	if targetPackage != nil && !method {
		resolver.qualifier(targetPackage)
	}

	if targetPackage == nil && !method {
		prefix, _, ok := strings.Cut(expression, ".")
		if ok {
			if imp, found := imports[prefix]; found && imp.Path != "" {
				alias := imp.Alias
				if alias == "" {
					alias = prefix
				}

				resolver.aliases[imp.Path] = alias
				resolver.imports = append(resolver.imports, model.GeneratedImport{Path: imp.Path, Alias: alias})
			}
		}
	}

	return resolver
}

func (r *targetImportResolver) qualifier(pkg *types.Package) string {
	if pkg == nil || pkg.Path() == r.current {
		return ""
	}

	if alias, ok := r.aliases[pkg.Path()]; ok {
		if !slices.ContainsFunc(r.imports, func(imp model.GeneratedImport) bool { return imp.Path == pkg.Path() }) {
			r.imports = append(r.imports, model.GeneratedImport{Path: pkg.Path(), Alias: alias})
		}

		return alias
	}

	alias := pkg.Name()
	if !slices.ContainsFunc(r.imports, func(imp model.GeneratedImport) bool { return imp.Path == pkg.Path() }) {
		r.imports = append(r.imports, model.GeneratedImport{Path: pkg.Path(), Alias: alias})
	}

	r.aliases[pkg.Path()] = alias

	return alias
}

func setterParameters(signature *types.Signature, qualifier types.Qualifier) []model.FuncParameter {
	params := slices.Collect(signature.Params().Variables())

	out := make([]model.FuncParameter, 0, len(params))
	for i, param := range params {
		name := param.Name()
		if name == "" || name == "_" {
			name = fmt.Sprintf("arg%d", i)
		}

		variadic := signature.Variadic() && i == len(params)-1

		paramType := param.Type()
		if variadic {
			if sliceType, ok := paramType.(*types.Slice); ok {
				paramType = sliceType.Elem()
			}
		}

		out = append(out, model.FuncParameter{Name: name, Type: types.TypeString(paramType, qualifier), Variadic: variadic})
	}

	return out
}

func hasInvalidType(sig *types.Signature) bool {
	for param := range sig.Params().Variables() {
		if containsInvalid(param.Type()) {
			return true
		}
	}

	for result := range sig.Results().Variables() {
		if containsInvalid(result.Type()) {
			return true
		}
	}

	return false
}

func containsInvalid(t types.Type) bool {
	return types.TypeString(t, func(*types.Package) string { return "" }) == "invalid type"
}
