// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package parser_test

import (
	"testing"

	"github.com/altessa-s/go-optgen/internal/parser"
)

func BenchmarkParseOptGen(b *testing.B) {
	for b.Loop() {
		parser.ParseOptGen("default=DefaultValue,manual,notnil")
	}
}

func BenchmarkParseExprList(b *testing.B) {
	for b.Loop() {
		parser.ParseExprList("[trimspaces,lower,trimprefix=api/]")
	}
}
