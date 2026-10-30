// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package semantic validates declarations and applies explicit plugin policies:
// the option name prefix, check applicability, type defaults, and the default
// modifiers derived from plugin metadata.
//
// # Usage
//
//	prepared, err := semantic.New(registry, semantic.WithPrefix("Validation")).Prepare(fields)
package semantic
