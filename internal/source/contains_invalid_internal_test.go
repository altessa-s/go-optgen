// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package source

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContainsInvalidThroughAliasAndNamedUnderlying(t *testing.T) {
	t.Parallel()

	pkg := types.NewPackage("example.com/sample", "sample")
	alias := types.NewAlias(types.NewTypeName(token.NoPos, pkg, "Handler", nil),
		types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", types.Typ[types.Invalid])), nil, false))
	named := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Broken", nil), types.Typ[types.Invalid], nil)

	require.True(t, containsInvalid(alias))
	require.True(t, containsInvalid(named))
}
