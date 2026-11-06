// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package plugin

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/altessa-s/go-optgen/model"
)

// GenerationContext is passed to every plugin during code generation. It carries
// package-level metadata (package name, type name, generic params) and allows
// plugins to register additional imports via [GenerationContext.AddImport].
type GenerationContext struct {
	// RenderSettings are the caller's names and switches, such as TypeName,
	// OptionType, OptionReturnsError, and the option function prefixes.
	model.RenderSettings

	// PackageName is the name of the package being generated.
	PackageName string

	// IsSetField names the struct field recording applied options when
	// WithIsSet is enabled.
	IsSetField string

	// AllFields contains all fields being generated.
	AllFields []model.OptField
	// StructFields contains every source struct field, including fields without generated options.
	StructFields []model.StructField
	// ValidationOnly identifies checks emitted in the generated validate method.
	ValidationOnly bool
	// OnFailSkip selects a no-op return for a failed option-body check.
	OnFailSkip bool
	// InputVar names the original value variable before the modifier pipeline.
	InputVar             string
	InputIsNewCollection bool

	// Imports contains all imports available in the source file.
	// Map key is the package alias, value is import info.
	Imports map[string]model.ImportInfo

	// GenericInfo contains generic type parameters if the struct is generic.
	GenericInfo model.GenericInfo

	// additionalImports tracks imports added by plugins.
	additionalImports map[string]model.GeneratedImport
}

// NewGenerationContext creates a GenerationContext with an empty additional-imports map.
// Callers should populate it once per generation run.
func NewGenerationContext(
	pkgName string,
	settings model.RenderSettings,
	fields []model.OptField,
	imports map[string]model.ImportInfo,
	genericInfo model.GenericInfo,
) GenerationContext {
	return GenerationContext{
		RenderSettings:    settings,
		PackageName:       pkgName,
		AllFields:         fields,
		Imports:           imports,
		GenericInfo:       genericInfo,
		additionalImports: make(map[string]model.GeneratedImport),
	}
}

// IsSetConst returns the name of the field-set constant of field.
func (ctx *GenerationContext) IsSetConst(field model.OptField) string {
	return ctx.TypeName + "Field" + field.OptionName
}

// IsGeneric returns true if generating for a generic struct.
func (ctx *GenerationContext) IsGeneric() bool {
	return ctx.GenericInfo.IsGeneric()
}

// TypeParamsDecl returns the type parameters declaration string.
// Example: "[T any]" or "" if not generic.
func (ctx *GenerationContext) TypeParamsDecl() string {
	return ctx.GenericInfo.TypeParamsDecl()
}

// TypeParamsNames returns just the type parameter names.
// Example: "[T]" or "" if not generic.
func (ctx *GenerationContext) TypeParamsNames() string {
	return ctx.GenericInfo.TypeParamsNames()
}

// AddImport adds an import to the generation context.
// The import will be included in the generated file.
//
// If alias is provided, it will be used as the import alias.
// Otherwise, the last segment of the path is used.
func (ctx *GenerationContext) AddImport(path string, alias ...string) {
	imp := model.GeneratedImport{Path: path}
	if len(alias) > 0 && alias[0] != "" {
		imp.Alias = alias[0]
	}
	// Deduplicate by path. Prefer explicit alias when provided.
	if existing, ok := ctx.additionalImports[path]; ok {
		if existing.Alias != "" && imp.Alias == "" {
			return
		}
	}

	ctx.additionalImports[path] = imp
}

// AddStdlibImport adds a standard-library import using a free package name and
// returns the identifier plugins should use in generated code.
func (ctx *GenerationContext) AddStdlibImport(path, preferred string) string {
	used := make(map[string]bool, len(ctx.Imports)+len(ctx.additionalImports))
	for alias, imp := range ctx.Imports {
		if alias != "" && alias != "_" && alias != "." && imp.Path != path {
			used[alias] = true
		}
	}

	for impPath, imp := range ctx.additionalImports {
		if imp.Alias != "" && impPath != path {
			used[imp.Alias] = true
		}
	}

	alias := preferred
	for suffix := 2; used[alias]; suffix++ {
		alias = preferred + strconv.Itoa(suffix)
	}

	if alias == preferred {
		ctx.AddImport(path)
	} else {
		ctx.AddImport(path, alias)
	}

	return alias
}

// BodyIdentifiers returns source identifiers referenced by a generated option
// body, including check expressions and cross-field reset types.
func BodyIdentifiers(ctx GenerationContext, field model.OptField, sources ...string) map[string]bool {
	used := make(map[string]bool)
	used[ctx.TypeName] = true

	used["o"] = true
	for _, param := range ctx.GenericInfo.TypeParams {
		used[param.Name] = true
	}

	if ctx.WithIsSet {
		used[ctx.IsSetConst(field)] = true
	}

	for alias := range ctx.Imports {
		if alias != "_" && alias != "." {
			used[alias] = true
		}
	}

	for _, expression := range field.Checks {
		addIdentifiers(used, expression)
	}

	for _, source := range sources {
		addIdentifiers(used, source)
	}

	addIdentifiers(used, field.Metadata["err"])

	for target := range strings.SplitSeq(field.Metadata["resets"], "|") {
		target = strings.TrimSpace(target)
		for _, sourceField := range ctx.StructFields {
			if sourceField.Name != target {
				continue
			}

			addIdentifiers(used, sourceField.Type)

			for _, imp := range sourceField.TypeImports {
				addIdentifiers(used, imp.Alias)
			}

			break
		}
	}

	return used
}

func addIdentifiers(used map[string]bool, source string) {
	for i := 0; i < len(source); {
		if !identifierStart(source[i]) {
			i++
			continue
		}

		start := i
		for i < len(source) && identifierContinue(source[i]) {
			i++
		}

		used[source[start:i]] = true
	}
}

func identifierStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func identifierContinue(c byte) bool { return identifierStart(c) || c >= '0' && c <= '9' }

// UniqueIdentifier appends Arg until name is absent from used identifiers.
func UniqueIdentifier(name string, used map[string]bool) string {
	for used[name] {
		name += "Arg"
	}

	return name
}

// GetAdditionalImports returns a deterministically-sorted slice of all imports
// added by plugins via [GenerationContext.AddImport]. The slice is sorted by
// import path, then alias.
func (ctx *GenerationContext) GetAdditionalImports() []model.GeneratedImport {
	imports := make([]model.GeneratedImport, 0, len(ctx.additionalImports))
	for _, imp := range ctx.additionalImports {
		imports = append(imports, imp)
	}

	slices.SortFunc(imports, func(a, b model.GeneratedImport) int {
		if c := cmp.Compare(a.Path, b.Path); c != 0 {
			return c
		}

		return cmp.Compare(a.Alias, b.Alias)
	})

	return imports
}
