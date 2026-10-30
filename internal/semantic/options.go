// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package semantic

//go:generate optgen generate

// DefaultDefaultModifiers enables the modifiers that plugins apply by default.
const DefaultDefaultModifiers = true

type options struct {
	// defaultModifiers applies the modifiers plugins declare for a type, such
	// as trimspaces for strings. Disabler keys are removed either way.
	defaultModifiers bool `optgen:"default=DefaultDefaultModifiers" optval:"param"`
	// prefix is prepended to every option name before validation.
	prefix string `optval:"notrim"`
	// warningSink receives diagnostics, such as a check skipped because its
	// plugin is disabled.
	warningSink func(message string)
}
