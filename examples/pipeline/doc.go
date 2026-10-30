// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package pipeline shows several option types generated in one run and a
// package-local .optgen.yaml.
//
// # Generation
//
//	//go:generate optgen generate --type checkOptions,transformOptions
//
// Each type gets its own option type and file: CheckOption in
// check_options_gen.go and TransformOption in transform_options_gen.go, so a
// CheckOption passed to Transform does not compile.
//
// # Configuration
//
// The .optgen.yaml next to this file replaces the repository one. It starts
// defaults from the defaultCheckOptionsValue and defaultTransformOptionsValue
// variables, skips arguments that fail an optcheck rule instead of
// panicking, makes slice options take a slice, and sets a type default for
// every time.Duration field.
package pipeline
