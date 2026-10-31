// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package model

import (
	"cmp"
	"errors"
	"fmt"
	"go/token"
)

// StringSetterMode selects the signature of string option functions.
type StringSetterMode string

// String setter modes. The zero value is [StringSetterGeneric].
const (
	// StringSetterGeneric accepts string or *string: WithName[T string | *string](v T).
	StringSetterGeneric StringSetterMode = "generic"
	// StringSetterPlain accepts a string: WithName(v string).
	StringSetterPlain StringSetterMode = "plain"
)

// CheckFailureMode selects how an option without an error result reports a
// failed optcheck validation.
type CheckFailureMode string

// Check failure modes. The zero value is [CheckFailurePanic].
const (
	// CheckFailurePanic panics with the validation error.
	CheckFailurePanic CheckFailureMode = "panic"
	// CheckFailureSkip returns without assigning the value.
	CheckFailureSkip CheckFailureMode = "skip"
)

// SliceSetterMode selects the parameter of slice option functions.
type SliceSetterMode string

// Slice setter modes. The zero value is [SliceSetterVariadic].
const (
	// SliceSetterVariadic accepts elements: WithHosts(v ...string).
	SliceSetterVariadic SliceSetterMode = "variadic"
	// SliceSetterSlice accepts a slice: WithHosts(v []string).
	SliceSetterSlice SliceSetterMode = "slice"
)

// DefaultsSource selects where defaultOptions takes its starting value from.
type DefaultsSource string

// Defaults sources. The zero value is [DefaultsFromTags].
const (
	// DefaultsFromTags starts from the zero value and applies tag and type defaults.
	DefaultsFromTags DefaultsSource = "tag"
	// DefaultsFromVar starts from a copy of the variable named DefaultsName.
	DefaultsFromVar DefaultsSource = "var"
	// DefaultsFromFunc starts from the result of the function named DefaultsName.
	DefaultsFromFunc DefaultsSource = "func"
	// DefaultsFromNone starts from the zero value and ignores every default.
	DefaultsFromNone DefaultsSource = "none"
)

// Default option function prefixes used when the settings leave them empty.
const (
	DefaultFuncPrefix       = "With"
	DefaultInvertFuncPrefix = "Without"
)

// RenderSettings are the caller-chosen names and switches of a generated
// file. Fields that begin with "Generate" control which top-level declarations
// are included in the output. Zero values select the documented defaults.
type RenderSettings struct {
	TypeName            string
	OptionType          string
	GenerateOptionType  bool
	GenerateDefaultFunc bool
	GenerateNewFunc     bool
	OptionReturnsError  bool
	DefaultFuncName     string
	NewFuncName         string
	// FuncPrefix prefixes option function names; empty means [DefaultFuncPrefix].
	FuncPrefix string
	// InvertFuncPrefix prefixes inverted bool flag options; empty means
	// [DefaultInvertFuncPrefix].
	InvertFuncPrefix string
	// StringSetter selects the string option signature.
	StringSetter StringSetterMode
	// KeepEmptyStrings assigns empty strings instead of skipping them.
	KeepEmptyStrings bool
	// CheckFailure applies when OptionReturnsError is false.
	CheckFailure CheckFailureMode
	// SliceSetter selects the parameter of slice options.
	SliceSetter SliceSetterMode
	// DefaultsFrom selects the starting value of defaultOptions; var and
	// func sources name their declaration in DefaultsName. Tag and type
	// defaults are applied on top, except for DefaultsFromNone.
	DefaultsFrom DefaultsSource
	DefaultsName string
	// Validate checks every optcheck rule on the final value in newOptions,
	// so unset fields and defaults are validated too. ValidateFuncName names
	// the generated validation function.
	Validate         bool
	ValidateFuncName string
	// WithIsSet records applied options in the struct field of type
	// <TypeName>FieldSet and generates an isSet method.
	WithIsSet bool
}

// ErrInvalidSettings reports a [RenderSettings] value outside its documented set.
var ErrInvalidSettings = errors.New("invalid render settings")

// Verify reports whether every mode of s is a documented value and whether
// a var or func defaults source names a valid identifier. Errors wrap
// [ErrInvalidSettings].
func (s RenderSettings) Verify() error {
	switch s.StringSetter {
	case "", StringSetterGeneric, StringSetterPlain:
	default:
		return fmt.Errorf("%w: string setter %q (want %s or %s)", ErrInvalidSettings, s.StringSetter, StringSetterGeneric, StringSetterPlain)
	}
	switch s.CheckFailure {
	case "", CheckFailurePanic, CheckFailureSkip:
	default:
		return fmt.Errorf("%w: check failure mode %q (want %s or %s)", ErrInvalidSettings, s.CheckFailure, CheckFailurePanic, CheckFailureSkip)
	}
	switch s.SliceSetter {
	case "", SliceSetterVariadic, SliceSetterSlice:
	default:
		return fmt.Errorf("%w: slice setter %q (want %s or %s)", ErrInvalidSettings, s.SliceSetter, SliceSetterVariadic, SliceSetterSlice)
	}
	switch s.DefaultsFrom {
	case "", DefaultsFromTags, DefaultsFromNone:
		if s.DefaultsName != "" {
			return fmt.Errorf("%w: defaults source %s takes no name", ErrInvalidSettings, cmp.Or(s.DefaultsFrom, DefaultsFromTags))
		}
	case DefaultsFromVar, DefaultsFromFunc:
		if s.DefaultsName != "" && !token.IsIdentifier(s.DefaultsName) {
			return fmt.Errorf("%w: defaults name %q is not a valid Go identifier", ErrInvalidSettings, s.DefaultsName)
		}
	default:
		return fmt.Errorf("%w: defaults source %q (want %s, %s, %s or %s)", ErrInvalidSettings, s.DefaultsFrom,
			DefaultsFromTags, DefaultsFromVar, DefaultsFromFunc, DefaultsFromNone)
	}
	return nil
}

// OptionFuncPrefix returns FuncPrefix or [DefaultFuncPrefix].
func (s RenderSettings) OptionFuncPrefix() string { return cmp.Or(s.FuncPrefix, DefaultFuncPrefix) }

// OptionInvertFuncPrefix returns InvertFuncPrefix or [DefaultInvertFuncPrefix].
func (s RenderSettings) OptionInvertFuncPrefix() string {
	return cmp.Or(s.InvertFuncPrefix, DefaultInvertFuncPrefix)
}

// GenerateInput holds every parameter needed by a renderer to emit a complete
// Go source file: the caller's settings plus the prepared package declarations.
type GenerateInput struct {
	RenderSettings
	PackageName         string
	Fields              []OptField
	Imports             map[string]ImportInfo
	NormalizationMethod string
	GenericInfo         GenericInfo
	// IsSetField names the struct field of type <TypeName>FieldSet that
	// records applied options; empty when the struct declares none.
	IsSetField string
}

// SourcePackage is the syntax adapter's result, before semantic preparation.
type SourcePackage struct {
	Name                string
	Fields              []OptField
	Imports             map[string]ImportInfo
	GenericInfo         GenericInfo
	NormalizationMethod string
	IsSetField          string
}
