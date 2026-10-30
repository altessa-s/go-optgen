// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package plugin

import "github.com/altessa-s/go-optgen/model"

// CheckSpec describes a registered optcheck key, including whether the check
// requires a value (e.g. "minlen=3" vs bare "required") and its execution
// priority relative to other checks.
type CheckSpec struct {
	Key           string
	RequiresValue bool
	// Priority controls the ordering of generated checks (lower first).
	// Use small integers (10, 20, 30) to leave gaps for custom checks.
	Priority int
}

// CheckPlugin generates validation code for a single optcheck key
// (e.g. "required", "minlen", "oneof"). Use [CheckBase] as an embedding
// helper to supply default Meta, Spec and Aliases implementations.
type CheckPlugin interface {
	PluginMeta
	Spec() CheckSpec
	Aliases() []string

	// Validate reports whether the check can be generated for field with
	// rawValue. It returns an error wrapping [ErrNotApplicable] when the field
	// type is unsupported and [ErrInvalidCheckValue] when rawValue is unusable.
	// Declarations are validated before generation, so Generate must emit code
	// for every field and value that Validate accepts.
	Validate(field model.OptField, rawValue string) error

	// Generate returns code lines (without indentation) to be inserted inside the option body.
	// Plugins may call ctx.AddImport(...) if needed.
	Generate(ctx GenerationContext, field model.OptField, valueVar, rawValue string) []string
}
