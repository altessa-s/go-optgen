// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package model

import "cmp"

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
}

// SourcePackage is the syntax adapter's result, before semantic preparation.
type SourcePackage struct {
	Name                string
	Fields              []OptField
	Imports             map[string]ImportInfo
	GenericInfo         GenericInfo
	NormalizationMethod string
}
