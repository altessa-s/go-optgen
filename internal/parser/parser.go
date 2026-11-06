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

	if before, after, found := strings.Cut(pkgName, "."); found && isVersionSegment(after) {
		pkgName = before
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
	Fields       []model.OptField // Parsed fields with opt tags
	StructFields []model.StructField
	Imports      map[string]model.ImportInfo // Import alias to ImportInfo mapping
	GenericInfo  model.GenericInfo           // Generic type parameters if the struct is generic
	IsSetField   string                      // Field of type <TypeName>FieldSet recording applied options
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
				if recordFieldSet(&result, field, typeName) {
					appendStructFields(&result, field, types)

					continue
				}

				appendStructFields(&result, field, types)

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

func recordFieldSet(result *FindOptFieldsResult, field *ast.Field, typeName string) bool {
	if TypeToString(field.Type) != typeName+"FieldSet" || len(field.Names) != 1 {
		return false
	}

	result.IsSetField = field.Names[0].Name

	return true
}

func appendStructFields(result *FindOptFieldsResult, field *ast.Field, resolver TypeResolver) {
	names := field.Names
	if len(names) == 0 {
		if name := embeddedFieldName(field.Type); name != "" {
			names = []*ast.Ident{ast.NewIdent(name)}
		}
	}

	facts, resolved := TypeFacts{}, false
	if resolver != nil {
		facts, resolved = resolver.TypeFacts(field.Type)
	}

	for _, name := range names {
		result.StructFields = append(result.StructFields, model.StructField{
			Name: name.Name, Type: TypeToString(field.Type), TypeResolved: resolved,
			TypeImports: ExtractImportsFromType(field.Type, result.Imports),
			IsBool:      resolved && facts.Bool, IsNumeric: resolved && facts.Numeric,
			IsComparable: resolved && facts.Comparable, IsInterface: resolved && facts.Interface,
			IsNilable: resolved && facts.Nilable,
		})
	}
}

func embeddedFieldName(expr ast.Expr) string {
	switch typ := expr.(type) {
	case *ast.Ident:
		return typ.Name
	case *ast.SelectorExpr:
		return typ.Sel.Name
	case *ast.StarExpr:
		return embeddedFieldName(typ.X)
	default:
		return ""
	}
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
	mapType, isMap := field.Type.(*ast.MapType)

	elemType := ""
	if isSlice {
		elemType = TypeToString(arrType.Elt)
	}

	// Extract documentation
	doc := extractFieldDoc(field)

	optionDocs, err := extractOptionDocs(field)
	if err != nil {
		return nil, err
	}

	modifiers := parsed.Modifiers
	isInterface := IsInterfaceType(field.Type)
	isNilable := IsNilableType(field.Type)
	typeImports := ExtractImportsFromType(field.Type, imports)
	mapKeyType, mapValueType, mapValueNilable := mapEntryFacts(mapType, types)
	mapKeyString := mapType != nil && TypeToString(mapType.Key) == "string"

	var (
		facts, elemFacts       TypeFacts
		resolved, elemResolved bool
	)
	if types != nil {
		if facts, resolved = types.TypeFacts(field.Type); resolved {
			isInterface, isNilable = facts.Interface, facts.Nilable
			mapKeyString = mapKeyString || facts.MapKeyString
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
			FieldName:       fieldName,
			OptionName:      optName,
			Type:            typeStr,
			TypeImports:     slices.Clone(typeImports),
			Default:         parsed.Default,
			Doc:             doc,
			IsSlice:         isSlice,
			IsMap:           isMap,
			MapKeyType:      mapKeyType,
			MapKeyString:    mapKeyString,
			MapValueType:    mapValueType,
			MapValueNilable: mapValueNilable,
			IsInterface:     isInterface,
			IsNilable:       isNilable,
			ElemType:        elemType,
			Modifiers:       slices.Clone(modifiers),
			Metadata:        maps.Clone(parsed.Metadata),
			Checks:          maps.Clone(parsed.Checks),
			BareChecks:      maps.Clone(parsed.BareChecks),
			HasModifiers:    len(modifiers) > 0,

			TypeResolved:   resolved,
			IsNumeric:      resolved && facts.Numeric,
			IsComparable:   resolved && facts.Comparable,
			ElemComparable: elemResolved && elemFacts.Comparable,
		}
		base.Metadata = markMultipleOption(base.Metadata, len(field.Names) > 1)

		base.CheckValueValid = checkValueFacts(field, parsed, types)

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

	for optionName, optionDoc := range optionDocs {
		found := false

		for i := range out {
			if out[i].OptionName != optionName {
				continue
			}

			out[i].OptionDoc = optionDoc
			out[i].HasOptionDoc = true
			found = true
		}

		if !found {
			return nil, fmt.Errorf("field %s: optgen:doc names option %s that this field does not produce", field.Names[0].Name, optionName)
		}
	}

	return out, nil
}

func markMultipleOption(metadata map[string]string, multiple bool) map[string]string {
	if !multiple {
		return metadata
	}

	if metadata == nil {
		metadata = make(map[string]string)
	}

	metadata["multiopt"] = tagValueTrue

	return metadata
}

func checkValueFacts(field *ast.Field, parsed parsedFieldTags, resolver TypeResolver) map[string]bool {
	factsResolver, ok := resolver.(CheckValueResolver)
	if !ok {
		return nil
	}

	facts := make(map[string]bool)

	for _, key := range []string{"min", checkOneOfKey} {
		var value string

		var exists bool

		for checkKey, checkValue := range parsed.Checks {
			if strings.EqualFold(checkKey, key) {
				value, exists = checkValue, true
				break
			}
		}

		if !exists {
			continue
		}

		expressions := []string{value}
		if key == checkOneOfKey {
			expressions = ParseExprList(value)
		}

		for _, expression := range expressions {
			facts[expression] = factsResolver.CheckValueValid(field.Type, field.Pos(), expression, key == checkOneOfKey)
		}
	}

	return facts
}

func mapEntryFacts(mapType *ast.MapType, types TypeResolver) (keyType, valueType string, valueNilable bool) {
	if mapType == nil {
		return "", "", false
	}

	keyType, valueType = TypeToString(mapType.Key), TypeToString(mapType.Value)
	valueNilable = IsNilableType(mapType.Value)

	if types != nil {
		if facts, resolved := types.TypeFacts(mapType); resolved {
			valueNilable = facts.MapValueNilable
		}
	}

	return keyType, valueType, valueNilable
}

func extractOptionDocs(field *ast.Field) (map[string]string, error) {
	if field.Doc == nil {
		return map[string]string{}, nil
	}

	docs := make(map[string][]string)

	for _, comment := range field.Doc.List {
		text := strings.TrimPrefix(comment.Text, "//")

		if text != "optgen:doc" && !strings.HasPrefix(text, "optgen:doc ") {
			continue
		}

		optionAndText := strings.TrimPrefix(text, "optgen:doc")
		optionAndText = strings.TrimSpace(optionAndText)
		option, docLine, hasText := strings.Cut(optionAndText, " ")

		if option == "" {
			return nil, fmt.Errorf("field %s: optgen:doc requires an option name", field.Names[0].Name)
		}

		if !hasText {
			docLine = ""
		}

		docs[option] = append(docs[option], docLine)
	}

	if len(docs) == 0 {
		return map[string]string{}, nil
	}

	joined := make(map[string]string, len(docs))
	for option, lines := range docs {
		joined[option] = strings.Join(lines, "\n")
	}

	return joined, nil
}

const tagValueTrue = "true"
const tagValueSet = "set"
const tagValuePut = "put"
const checkOneOfKey = "oneof"

func validateSetterTags(field *ast.Field, ctx fieldProcessingContext, parsed parsedFieldTags) error {
	fieldName := field.Names[0].Name
	if parsed.TagError != nil {
		return fmt.Errorf("field %s: %w", fieldName, parsed.TagError)
	}

	_, isSet := parsed.Metadata[tagValueSet]

	_, isPut := parsed.Metadata[tagValuePut]

	if isSet && isPut {
		return fmt.Errorf("field %s: optgen set and put cannot be combined", fieldName)
	}

	if err := validateCollectionSetterTags(field, parsed, isSet, isPut); err != nil {
		return err
	}

	if ctx.hasOptfn && len(field.Names) > 1 {
		return fmt.Errorf("field %s: optfn cannot be used with multiple field names", fieldName)
	}

	if err := validateOptfnTags(fieldName, ctx, parsed); err != nil {
		return err
	}

	return nil
}

func validateCollectionSetterTags(field *ast.Field, parsed parsedFieldTags, isSet, isPut bool) error {
	fieldName := field.Names[0].Name

	if err := validateCollectionType(fieldName, field.Type, isSet, isPut); err != nil {
		return err
	}

	if err := validateCollectionOptions(fieldName, parsed, isSet, isPut); err != nil {
		return err
	}

	if err := validateCloneMode(fieldName, parsed.Modifiers); err != nil {
		return err
	}

	if parsed.Metadata["parseip"] == tagValueTrue && hasModifierFold(parsed.Modifiers, "clone=eager") {
		return fmt.Errorf("field %s: optgen parseip cannot be combined with optval clone=eager", fieldName)
	}

	return validateCloneType(fieldName, field.Type, parsed.Modifiers)
}

func validateCollectionType(fieldName string, fieldType ast.Expr, isSet, isPut bool) error {
	if isSet {
		mapType, ok := fieldType.(*ast.MapType)

		value, emptyStruct := mapTypeValue(mapType)
		if !ok || !emptyStruct || TypeToString(value) != "struct{}" {
			return fmt.Errorf("field %s: optgen set requires map[K]struct{}", fieldName)
		}
	}

	if isPut {
		if _, ok := fieldType.(*ast.MapType); !ok {
			return fmt.Errorf("field %s: optgen put requires a map field", fieldName)
		}
	}

	return nil
}

func validateCollectionOptions(fieldName string, parsed parsedFieldTags, isSet, isPut bool) error {
	if !isSet && !isPut {
		return nil
	}

	kind := tagValueSet
	if isPut {
		kind = tagValuePut
	}

	if len(parsed.Checks) > 0 {
		return fmt.Errorf("field %s: optgen %s cannot be combined with optcheck", fieldName, kind)
	}

	if len(parsed.Modifiers) > 0 {
		return fmt.Errorf("field %s: optgen %s cannot be combined with optval", fieldName, kind)
	}

	for key := range parsed.Metadata {
		switch key {
		case tagValueSet, tagValuePut, "append", "manual", "enables", "resets":
		default:
			return fmt.Errorf("field %s: optgen %s cannot be combined with optgen modifier %q", fieldName, kind, key)
		}
	}

	return nil
}

func validateCloneMode(fieldName string, modifiers []string) error {
	for _, modifier := range modifiers {
		if key, value, ok := strings.Cut(modifier, "="); ok && strings.EqualFold(key, "clone") && !strings.EqualFold(value, "eager") {
			return fmt.Errorf("field %s: unsupported clone mode %q", fieldName, value)
		}
	}

	return nil
}

func validateCloneType(fieldName string, fieldType ast.Expr, modifiers []string) error {
	for _, modifier := range modifiers {
		if !strings.EqualFold(modifier, "clone") && !strings.EqualFold(modifier, "clone=eager") {
			continue
		}

		if arr, slice := fieldType.(*ast.ArrayType); slice && arr.Len == nil {
			continue
		}

		if _, mapField := fieldType.(*ast.MapType); mapField {
			continue
		}

		return fmt.Errorf("field %s: optval %s requires a slice or map field", fieldName, modifier)
	}

	return nil
}

func hasModifierFold(modifiers []string, want string) bool {
	for _, modifier := range modifiers {
		if strings.EqualFold(modifier, want) {
			return true
		}
	}

	return false
}

func validateOptfnTags(fieldName string, ctx fieldProcessingContext, parsed parsedFieldTags) error {
	if !ctx.hasOptfn {
		return nil
	}

	if len(parsed.SetterTargets) > 0 && strings.TrimSpace(ctx.tag) != "" {
		return fmt.Errorf("field %s: optfn cannot be combined with opt", fieldName)
	}

	if _, exists := LookupTag(ctx.tagStr, "optval"); exists {
		return fmt.Errorf("field %s: optfn cannot be combined with optval", fieldName)
	}

	if _, exists := LookupTag(ctx.tagStr, "optcheck"); exists {
		return fmt.Errorf("field %s: optfn cannot be combined with optcheck", fieldName)
	}

	for key := range parsed.Metadata {
		if key != "manual" && key != "enables" && key != "resets" {
			return fmt.Errorf("field %s: optfn cannot be combined with optgen modifier %q", fieldName, key)
		}
	}

	return nil
}

func mapTypeValue(t *ast.MapType) (ast.Expr, bool) {
	if t == nil {
		return nil, false
	}

	value, ok := t.Value.(*ast.StructType)

	return t.Value, ok && len(value.Fields.List) == 0
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

		setterField.Metadata["optfn"] = tagValueTrue

		if len(targets) > 1 || len(field.Names) > 1 {
			setterField.Metadata["multiopt"] = "true"
		}

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
