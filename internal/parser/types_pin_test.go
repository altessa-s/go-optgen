// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package parser_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/parser"
	"github.com/altessa-s/go-optgen/model"
)

func TestExtractImportsFromType_StdLibPin(t *testing.T) {
	t.Parallel()
	for _, expr := range []string{"context.Context", "*context.Context", "[]context.Context", "[]*context.Context"} {
		t.Run(expr, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, []string{"context"}, importSpecs(parser.ExtractImportsFromType(mustParseExpr(t, expr), map[string]model.ImportInfo{})))
			require.Equal(t, []string{"context"}, importSpecs(parser.ExtractImportsFromType(mustParseExpr(t, expr),
				map[string]model.ImportInfo{"context": {Path: "context"}})), "listed once")
		})
	}
}
