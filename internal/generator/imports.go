// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package generator

import (
	"cmp"
	"slices"
	"strings"

	"github.com/altessa-s/go-optgen/model"
)

// extractPackagePrefixes extracts package prefixes from a Go expression.
// For "base64.NewKeyDecoder()" returns ["base64"].
// For "json.NewValueDecoder[T]()" returns ["json"].
func extractPackagePrefixes(expr string) []string {
	var prefixes []string
	// Simple regex-like matching: find "identifier." patterns
	// This handles cases like "pkg.Func()" or "pkg.Type{}"
	i := 0
	for i < len(expr) {
		// Skip non-identifier characters and string or rune literals, whose
		// contents (such as "a.b") never name a package.
		for i < len(expr) && !isIdentStart(expr[i]) {
			if q := expr[i]; q == '"' || q == '`' || q == '\'' {
				i = skipLiteral(expr, i)
				continue
			}
			i++
		}
		if i >= len(expr) {
			break
		}
		// Collect identifier
		start := i
		for i < len(expr) && isIdentContinue(expr[i]) {
			i++
		}
		ident := expr[start:i]
		// Check if followed by '.'
		if i < len(expr) && expr[i] == '.' {
			// Check next char is also identifier (not "...")
			if i+1 < len(expr) && isIdentStart(expr[i+1]) {
				prefixes = append(prefixes, ident)
			}
		}
	}
	return prefixes
}

// skipLiteral returns the index after the literal that starts at expr[start].
// Backslash escapes apply to interpreted literals; raw literals end at the
// next backquote. An unterminated literal extends to the end of expr.
func skipLiteral(expr string, start int) int {
	quote := expr[start]
	for i := start + 1; i < len(expr); i++ {
		switch expr[i] {
		case '\\':
			if quote != '`' {
				i++
			}
		case quote:
			return i + 1
		}
	}
	return len(expr)
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentContinue(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// sourceImports returns the source-file imports referenced by generated code:
// the packages of field types used in option signatures or defaults, and the
// packages named in default expressions.
func sourceImports(
	fields []model.OptField,
	localImports map[string]model.ImportInfo,
	needsTypeImports map[string]bool,
) []model.GeneratedImport {
	imports := make(importSet)
	importByName := indexImportsByName(localImports)

	for _, field := range fields {
		// Only include imports for the field type if the generated code references the type
		// (e.g., WithXxx parameter types) or if defaultOptions will reference a non-empty default.
		if needsTypeImports[field.FieldName] || field.Default != "" {
			for _, imp := range field.TypeImports {
				imports.add(imp.Path, imp.Alias)
			}
		}

		// Extract package prefixes from default expression and add their imports
		if field.Default != "" {
			for _, prefix := range extractPackagePrefixes(field.Default) {
				if imp, ok := importByName[prefix]; ok {
					imports.add(imp.Path, imp.Alias)
				}
			}
		}
	}

	return imports.sorted()
}

// importSet collects generated imports by path.
type importSet map[string]model.GeneratedImport

// add records the import of path; an explicit alias is preferred over none.
func (s importSet) add(path, alias string) {
	if path == "" {
		return
	}
	if existing, ok := s[path]; ok {
		// Prefer explicit alias when present.
		if existing.Alias == "" && alias != "" {
			s[path] = model.GeneratedImport{Path: path, Alias: alias}
		}
		return
	}
	s[path] = model.GeneratedImport{Path: path, Alias: alias}
}

// sorted returns the imports ordered by path, then alias.
func (s importSet) sorted() []model.GeneratedImport {
	result := make([]model.GeneratedImport, 0, len(s))
	for _, imp := range s {
		result = append(result, imp)
	}
	slices.SortFunc(result, func(a, b model.GeneratedImport) int {
		if c := cmp.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		return cmp.Compare(a.Alias, b.Alias)
	})
	return result
}

// indexImportsByName indexes the source file's imports for default
// expressions. localImports is already keyed by the identifier name used in
// code:
//   - For aliased imports (vaultApi "pkg/api"), key = "vaultApi"
//   - For regular imports ("pkg" or _ "pkg"), key = package name (last path segment)
//
// The index also holds each import by package name (last path segment), for
// default expressions that may use package names directly.
func indexImportsByName(localImports map[string]model.ImportInfo) map[string]model.ImportInfo {
	importByName := make(map[string]model.ImportInfo, len(localImports)*2)
	for key, imp := range localImports {
		importByName[key] = imp
		// Also index by package name (last path segment) if different from key
		pathName := imp.Path
		if idx := strings.LastIndex(imp.Path, "/"); idx >= 0 {
			pathName = imp.Path[idx+1:]
		}
		if pathName != key {
			if _, exists := importByName[pathName]; !exists {
				importByName[pathName] = imp
			}
		}
	}
	return importByName
}
