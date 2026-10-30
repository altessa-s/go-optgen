// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package parser_test

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/parser"
)

func FuzzStructTagRoundTrip(f *testing.F) {
	for _, seed := range []string{"Name", "-", "a,b", "quoted \"value\"", "", "\n"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		// Backticks cannot occur in a Go raw-string struct tag.
		if strings.ContainsRune(value, '`') {
			t.Skip()
		}
		tag := "opt:" + strconv.Quote(value)
		got, ok := parser.LookupTag(tag, "opt")
		expected, exists := reflect.StructTag(tag).Lookup("opt")
		require.Equal(t, exists, ok)
		require.Equal(t, expected, got)
		if exists {
			require.Equal(t, value, got)
		}
	})
}

func FuzzExpressionList(f *testing.F) {
	for _, seed := range []string{"a,b", "[a,b]", "map[string]int{a:1,b:2},x", "", "f(a,b),x", `"a,b",c`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		got := parser.ParseExprList(input)
		require.Equal(t, got, parser.ParseExprList(input))
		for _, part := range got {
			require.NotEmpty(t, part)
			require.Equal(t, strings.TrimSpace(part), part)
		}
	})
}

func TestSkippedFieldsNeverRetainDefaults(t *testing.T) {
	t.Parallel()
	pkg := parsePackage(t, map[string]string{"options.go": `package sample
 type options struct {
 skipped int ~opt:"-" optgen:"default=42"~
 manual int ~optgen:"manual,default=7"~
 }`})
	for _, all := range []bool{false, true} {
		result, err := parser.FindOptFields(pkg, "options", all, nil)
		require.NoError(t, err)
		require.Len(t, result.Fields, 1)
		require.Equal(t, "manual", result.Fields[0].FieldName)
		require.Equal(t, "7", result.Fields[0].Default)
	}
}
