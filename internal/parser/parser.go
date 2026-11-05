// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package parser

import (
	"errors"
	"fmt"
	"go/ast"
	"maps"
	"slices"
	"strings"

	"github.com/altessa-s/go-optgen/internal/naming"
	"github.com/altessa-s/go-optgen/model"
)

// derivePackageName extracts the package name from an import path.
// Handles special cases:
//   - Versioned imports: github.com/redis/go-redis/v9 → go-redis
//   - .go suffix: github.com/nats-io/nats.go → nats
//   - Regular imports: github.com/foo/bar → bar
//
// Note that the result is the last meaningful path segment, which may differ
// from the actual Go package name (e.g. go-redis vs package redis); callers
// that need the real package name must handle that themselves.
func derivePackageName(path string) string {
	parts := strings.Split(path, "/")
	pkgName := parts[len(parts)-1]

	// Handle versioned imports (v2, v3, ..., v9, v10, etc.)
	// If last segment is a version, use the previous segment
	if isVersionSegment(pkgName) && len(parts) >= 2 {
		pkgName = parts[len(parts)-2]
	}

	// Handle special case: import path ending with ".go" suffix
	// e.g., github.com/nats-io/nats.go -> package name is "nats"
	if trimmed, found := strings.CutSuffix(pkgName, ".go"); found {
		pkgName = trimmed
	}

	return pkgName
}

// isVersionSegment checks if a path segment is a Go module version (v2, v3, etc.)
func isVersionSegment(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}

	for _, c := range s[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

// FindOptFieldsResult contains the result of parsing opt fields from a struct.
type FindOptFieldsResult struct {
	Fields      []model.OptField            // Parsed fields with opt tags
	Imports     map[string]model.ImportInfo // Import alias to ImportInfo mapping
	GenericInfo model.GenericInfo           // Generic type parameters if the struct is generic
	IsSetField  string                      // Field of type <TypeName>FieldSet recording applied options
}

// FindOptFields finds all fields with opt tags in the specified struct type.
// It scans the AST package for the target struct and extracts field information
// from opt tags. It also extracts generic type parameters if the struct is generic.
//
// Parameters:
//   - pkg: The AST package to scan
//   - typeName: The struct type name to find
//   - processAllFields: If true, process all fields (not just those with opt tags)
//   - types: Optional type facts for field types; nil keeps syntax-only inference
//
// Returns:
//   - FindOptFieldsResult containing fields, imports, and generic info
//   - error: Validation errors or parsing errors
//
//nolint:staticcheck // SA1019: ast.Package is deprecated; optgen intentionally operates on go/ast packages.
func FindOptFields(pkg *ast.Package, typeName string, processAllFields bool, types TypeResolver) (FindOptFieldsResult, error) {
	result := FindOptFieldsResult{
		Fields:  []model.OptField{},
		Imports: make(map[string]model.ImportInfo),
	}

	var parseErr error

	var declaredIn []string // files declaring typeName, in order

	fileNames := slices.Sorted(maps.Keys(pkg.Files))

	for _, name := range fileNames {
		file := pkg.Files[name]

		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok || ts.Name.Name != typeName {
				return true
			}

			if !slices.Contains(declaredIn, name) {
				declaredIn = append(declaredIn, name)
			}

			// Found the target struct! Collect imports from THIS file only.
			collectFileImports(file, result.Imports)

			// Extract generic type parameters if present
			if typeParams := ExtractTypeParams(ts); len(typeParams) > 0 {
				result.GenericInfo = model.GenericInfo{TypeParams: typeParams}
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}

			// Process each field in the struct
			for _, field := range st.Fields.List {
				// The field set records applied options; it is not an option.
				if TypeToString(field.Type) == typeName+"FieldSet" && len(field.Names) == 1 {
					result.IsSetField = field.Names[0].Name
					continue
				}

				optFields, err := processStructField(field, processAllFields, result.Imports, types)
				if err != nil {
					parseErr = err
					return false
				}

				result.Fields = append(result.Fields, optFields...)
			}

			return true
		})
	}

	if parseErr == nil && len(declaredIn) > 1 {
		// Files with different build constraints may declare the type once
		// per platform; merging them would duplicate fields.
		return result, fmt.Errorf("type %s is declared in several files (%s); declare it in one file", typeName, strings.Join(declaredIn, ", "))
	}

	sortOptFields(result.Fields)

	return result, parseErr
}

// collectFileImports extracts import information from a file's imports.
// This avoids cross-file import conflicts (e.g., "time" vs "custom/time").
func collectFileImports(file *ast.File, imports map[string]model.ImportInfo) {
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		pkgName := derivePackageName(path)

		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		}

		// Determine the key for lookup in ExtractImportsFromType.
		// The key must match what's used in the type expression:
		// - For aliased imports (vaultApi "pkg/api"), key = "vaultApi"
		// - For blank imports (_ "pkg"), key = package name, alias becomes ""
		// - For regular imports ("pkg"), key = package name
		key := pkgName
		if alias != "" && alias != "_" && alias != "." {
			key = alias
		}

		if alias == "_" {
			alias = "" // Convert blank import to regular import in generated code
		}

		imports[key] = model.ImportInfo{Path: path, Alias: alias}
	}
}

// fieldProcessingContext holds intermediate state during field processing.
type fieldProcessingContext struct {
	tagStr    string
	tag       string
	hasOptTag bool
	optgenTag string
	hasOptgen bool
	optfn     string
	hasOptfn  bool
}

// processStructField processes a single struct field and returns the OptFields if it should be included.
// imports resolves the packages referenced by the field type.
func processStructField(field *ast.Field, processAllFields bool, imports map[string]model.ImportInfo, types TypeResolver) ([]model.OptField, error) {
	if len(field.Names) == 0 {
		return nil, nil
	}

	ctx := extractFieldContext(field)

	// Check if field should be skipped
	if ctx.hasOptTag {
		_, skip := ParseOptName(ctx.tag)
		if skip {
			if ctx.hasOptfn {
				return nil, fmt.Errorf("field %s: opt:\"-\" cannot be combined with optfn", field.Names[0].Name)
			}

			return nil, nil
		}
	}

	// Determine if we should process this field
	if !shouldProcessField(ctx, processAllFields) {
		return nil, nil
	}

	// Build and return the OptFields
	return buildOptFields(field, ctx, imports, types)
}

// extractFieldContext extracts tag information from a field.
func extractFieldContext(field *ast.Field) fieldProcessingContext {
	tagStr := ""
	if field.Tag != nil {
		tagStr = field.Tag.Value
	}

	tag, hasOptTag := LookupTag(tagStr, "opt")
	optgenTag, hasOptgen := LookupTag(tagStr, "optgen")
	optfn, hasOptfn := LookupTag(tagStr, "optfn")

	return fieldProcessingContext{
		tagStr:    tagStr,
		tag:       tag,
		hasOptTag: hasOptTag,
		optgenTag: optgenTag,
		hasOptgen: hasOptgen,
		optfn:     optfn,
		hasOptfn:  hasOptfn,
	}
}

// shouldProcessField determines if a field should be processed based on its tags.
func shouldProcessField(ctx fieldProcessingContext, processAllFields bool) bool {
	switch {
	case processAllFields:
		return true
	case ctx.hasOptTag:
		return true
	default:
		_, hasOptval := LookupTag(ctx.tagStr, "optval")
		_, hasOptcheck := LookupTag(ctx.tagStr, "optcheck")
		_, hasOptfn := LookupTag(ctx.tagStr, "optfn")

		return ctx.hasOptgen || hasOptval || hasOptcheck || hasOptfn
	}
}

// buildOptFields constructs OptFields from an AST field and its context.
func buildOptFields(field *ast.Field, ctx fieldProcessingContext, imports map[string]model.ImportInfo, types TypeResolver) ([]model.OptField, error) {
	explicitOptName, _ := ParseOptName(ctx.tag)

	parsed := parseFieldTags(ctx.tagStr)
	if err := validateSetterTags(field, ctx, parsed); err != nil {
		return nil, err
	}

	// Extract type information
	typeStr := TypeToString(field.Type)
	// Only a slice gets the slice setters; a fixed-size array is a scalar.
	arrType, isSlice := field.Type.(*ast.ArrayType)
	isSlice = isSlice && arrType.Len == nil

	elemType := ""
	if isSlice {
		elemType = TypeToString(arrType.Elt)
	}

	// Extract documentation
	doc := extractFieldDoc(field)
	modifiers := parsed.Modifiers
	isInterface := IsInterfaceType(field.Type)
	isNilable := IsNilableType(field.Type)
	typeImports := ExtractImportsFromType(field.Type, imports)

	var (
		facts, elemFacts       TypeFacts
		resolved, elemResolved bool
	)
	if types != nil {
		if facts, resolved = types.TypeFacts(field.Type); resolved {
			isInterface, isNilable = facts.Interface, facts.Nilable
		}

		if isSlice {
			elemFacts, elemResolved = types.TypeFacts(arrType.Elt)
		}
	}

	out := make([]model.OptField, 0, len(field.Names)*max(1, len(parsed.SetterTargets)))
	for _, ident := range field.Names {
		fieldName := ident.Name

		optName := explicitOptName
		if optName == "" || len(field.Names) > 1 {
			optName = naming.OptionName(fieldName)
		}

		base := model.OptField{
			FieldName:    fieldName,
			OptionName:   optName,
			Type:         typeStr,
			TypeImports:  slices.Clone(typeImports),
			Default:      parsed.Default,
			Doc:          doc,
			IsSlice:      isSlice,
			IsInterface:  isInterface,
			IsNilable:    isNilable,
			ElemType:     elemType,
			Modifiers:    slices.Clone(modifiers),
			Metadata:     maps.Clone(parsed.Metadata),
			Checks:       maps.Clone(parsed.Checks),
			BareChecks:   maps.Clone(parsed.BareChecks),
			HasModifiers: len(modifiers) > 0,

			TypeResolved:   resolved,
			IsNumeric:      resolved && facts.Numeric,
			IsComparable:   resolved && facts.Comparable,
			ElemComparable: elemResolved && elemFacts.Comparable,
		}
		if len(parsed.SetterTargets) == 0 {
			out = append(out, base)
			continue
		}

		targetFields, err := resolveSetterFields(field, base, parsed.SetterTargets, imports, types)
		if err != nil {
			return nil, err
		}

		out = append(out, targetFields...)
	}

	return out, nil
}

func validateSetterTags(field *ast.Field, ctx fieldProcessingContext, parsed parsedFieldTags) error {
	fieldName := field.Names[0].Name
	if parsed.TagError != nil {
		return fmt.Errorf("field %s: %w", fieldName, parsed.TagError)
	}

	if !ctx.hasOptfn {
		return nil
	}

	if _, exists := LookupTag(ctx.tagStr, "optval"); exists {
		return fmt.Errorf("field %s: optfn cannot be combined with optval", fieldName)
	}

	if _, exists := LookupTag(ctx.tagStr, "optcheck"); exists {
		return fmt.Errorf("field %s: optfn cannot be combined with optcheck", fieldName)
	}

	for key := range parsed.Metadata {
		if key != "manual" {
			return fmt.Errorf("field %s: optfn cannot be combined with optgen modifier %q", fieldName, key)
		}
	}

	return nil
}

func resolveSetterFields(
	field *ast.Field,
	base model.OptField,
	targets map[string]string,
	imports map[string]model.ImportInfo,
	resolver TypeResolver,
) ([]model.OptField, error) {
	if resolver == nil {
		return nil, fmt.Errorf("field %s: optfn target resolution requires type information", base.FieldName)
	}

	fields := make([]model.OptField, 0, len(targets))
	for _, optionName := range slices.Sorted(maps.Keys(targets)) {
		expression := targets[optionName]

		target, err := resolver.ResolveSetterTarget(field.Type, field.Pos(), expression, strings.HasPrefix(expression, "."), imports)
		if err != nil {
			return nil, fmt.Errorf("field %s option %s target %s: %w", base.FieldName, optionName, expression, err)
		}

		setterField := base
		setterField.OptionName = optionName
		setterField.SetterTarget = &target

		if setterField.Metadata == nil {
			setterField.Metadata = make(map[string]string)
		}

		setterField.Metadata["optfn"] = "true"
		fields = append(fields, setterField)
	}

	return fields, nil
}

// extractFieldDoc extracts documentation from a field's Doc or Comment.
func extractFieldDoc(field *ast.Field) string {
	if field.Doc != nil {
		return strings.TrimSpace(field.Doc.Text())
	}

	if field.Comment != nil {
		return strings.TrimSpace(field.Comment.Text())
	}

	return ""
}

// sortOptFields sorts fields by OptionName, then FieldName.
func sortOptFields(fields []model.OptField) {
	slices.SortFunc(fields, func(a, b model.OptField) int {
		if a.OptionName != b.OptionName {
			return strings.Compare(a.OptionName, b.OptionName)
		}

		return strings.Compare(a.FieldName, b.FieldName)
	})
}

// ErrGenericNormalization reports a normalization method that declares its own
// type parameters: the generated call o.Normalization() cannot infer them.
var ErrGenericNormalization = errors.New("normalization method must not declare type parameters")

// FindNormalizationMethod searches for a Normalization or normalization method
// on the specified struct type. Returns the method name if found, or empty string otherwise.
// It returns [ErrGenericNormalization] when the method declares type parameters.
//
//nolint:staticcheck // SA1019: ast.Package is deprecated; optgen intentionally operates on go/ast packages.
func FindNormalizationMethod(pkg *ast.Package, typeName string) (string, error) {
	for _, name := range slices.Sorted(maps.Keys(pkg.Files)) {
		file := pkg.Files[name]
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
				continue
			}

			if receiverTypeName(funcDecl.Recv.List[0].Type) != typeName {
				continue
			}

			// Check for Normalization or normalization method
			methodName := funcDecl.Name.Name
			if methodName == "Normalization" || methodName == "normalization" {
				if funcDecl.Type.TypeParams != nil {
					return "", fmt.Errorf("%s.%s: %w", typeName, methodName, ErrGenericNormalization)
				}

				return methodName, nil
			}
		}
	}

	return "", nil
}

func receiverTypeName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.IndexListExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}

	return ""
}
