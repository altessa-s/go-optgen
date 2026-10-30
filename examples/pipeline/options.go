// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pipeline

import "time"

// Several types in one run: each gets its own option type, default and new
// functions, and output file (checkOptions -> CheckOption,
// check_options_gen.go), so options of one step cannot be passed to the
// other.
//go:generate optgen generate --type checkOptions,transformOptions

// checkOptions configure Check.
type checkOptions struct {
	// timeout gets 30s from the type-defaults of .optgen.yaml.
	timeout time.Duration
	// maxLen starts at the value of defaultCheckOptionsValue; a zero
	// argument fails nonzero and is skipped.
	maxLen int `optcheck:"nonzero"`
	// forbidden words are compared in lower case.
	forbidden []string `optval:"lower,dedup"`
}

// transformOptions configure Transform.
type transformOptions struct {
	// letterCase starts at "lower" from defaultTransformOptionsValue; the
	// default= tag of prefix applies on top of that variable.
	letterCase string `optval:"lower" optcheck:"oneof=[lower,upper,keep]"`
	// prefix keeps its surrounding spaces: notrim disables the default
	// trimspaces modifier.
	prefix string `optgen:"default=\"> \"" optval:"notrim"`
}

// DefaultMaxLen is the longest text Check accepts by default.
const DefaultMaxLen = 80

var defaultCheckOptionsValue = checkOptions{maxLen: DefaultMaxLen}

var defaultTransformOptionsValue = transformOptions{letterCase: caseLower}
