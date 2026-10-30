// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package generator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/model"
)

func TestSourceImports_Pin(t *testing.T) {
	t.Parallel()

	typeExpr := func(_ testing.TB, src string) model.OptField {
		return model.OptField{FieldName: src}
	}

	t.Run("aliased type import", func(t *testing.T) {
		t.Parallel()
		f := typeExpr(t, "mo.ClientOptions")
		f.TypeImports = []model.GeneratedImport{{Path: "go.mongodb.org/mongo-driver/v2/mongo/options", Alias: "mo"}}
		local := map[string]model.ImportInfo{"mo": {Path: "go.mongodb.org/mongo-driver/v2/mongo/options", Alias: "mo"}}
		got := sourceImports([]model.OptField{f}, local, map[string]bool{f.FieldName: true})
		require.Equal(t, []model.GeneratedImport{{Path: "go.mongodb.org/mongo-driver/v2/mongo/options", Alias: "mo"}}, got)
	})

	t.Run("default expression by package name", func(t *testing.T) {
		t.Parallel()
		f := model.OptField{FieldName: "cfg", Default: "api.DefaultConfig()"}
		local := map[string]model.ImportInfo{"vaultApi": {Path: "github.com/hashicorp/vault/api", Alias: "vaultApi"}}
		got := sourceImports([]model.OptField{f}, local, nil)
		require.Equal(t, []model.GeneratedImport{{Path: "github.com/hashicorp/vault/api", Alias: "vaultApi"}}, got)
	})

	t.Run("an alias found later replaces a plain import", func(t *testing.T) {
		t.Parallel()
		f1 := typeExpr(t, "time.Duration")
		f1.TypeImports = []model.GeneratedImport{{Path: "time"}}
		f2 := model.OptField{FieldName: "at", Default: "t.Second"}
		local := map[string]model.ImportInfo{"time": {Path: "time"}, "t": {Path: "time", Alias: "t"}}
		got := sourceImports([]model.OptField{f1, f2}, local, map[string]bool{f1.FieldName: true})
		require.Equal(t, []model.GeneratedImport{{Path: "time", Alias: "t"}}, got)
	})

	t.Run("string and rune literals in defaults name no package", func(t *testing.T) {
		t.Parallel()
		f := model.OptField{FieldName: "addr", Default: `"a.b" + x.Y + ` + "`c.d` + string('e') + \"f\\\".g\""}
		local := map[string]model.ImportInfo{"a": {Path: "example.com/a"}, "c": {Path: "example.com/c"}, "g": {Path: "example.com/g"},
			"x": {Path: "example.com/x"}}
		got := sourceImports([]model.OptField{f}, local, nil)
		require.Equal(t, []model.GeneratedImport{{Path: "example.com/x"}}, got)
	})

	t.Run("an import without a path is dropped and the result is sorted", func(t *testing.T) {
		t.Parallel()
		f1 := model.OptField{FieldName: "x", Default: "x.Y + z.W"}
		f2 := typeExpr(t, "b.T")
		f2.TypeImports = []model.GeneratedImport{{Path: "example.com/b"}}
		local := map[string]model.ImportInfo{"x": {}, "z": {Path: "example.com/z"}, "b": {Path: "example.com/b"}}
		got := sourceImports([]model.OptField{f1, f2}, local, map[string]bool{f2.FieldName: true})
		require.Equal(t, []model.GeneratedImport{{Path: "example.com/b"}, {Path: "example.com/z"}}, got)
	})
}
