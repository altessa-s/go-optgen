// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package model_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/model"
)

func TestOptFieldHasModifier(t *testing.T) {
	t.Parallel()

	f := model.OptField{Modifiers: []string{"upper", "trimspaces"}, HasModifiers: true}
	require.True(t, f.HasModifier("upper"))
	require.False(t, f.HasModifier("lower"))

	f.HasModifiers = false
	require.False(t, f.HasModifier("upper"), "HasModifiers=false short-circuits")
}

func TestGenericInfo(t *testing.T) {
	t.Parallel()

	var none model.GenericInfo
	require.False(t, none.IsGeneric())
	require.Empty(t, none.TypeParamsDecl())
	require.Empty(t, none.TypeParamsNames())

	one := model.GenericInfo{TypeParams: []model.TypeParam{{Name: "T", Constraint: "any"}}}
	require.True(t, one.IsGeneric())
	require.Equal(t, "[T any]", one.TypeParamsDecl())
	require.Equal(t, "[T]", one.TypeParamsNames())

	two := model.GenericInfo{TypeParams: []model.TypeParam{
		{Name: "T", Constraint: "any"},
		{Name: "U", Constraint: "comparable"},
	}}
	require.Equal(t, "[T any, U comparable]", two.TypeParamsDecl())
	require.Equal(t, "[T, U]", two.TypeParamsNames())
}
