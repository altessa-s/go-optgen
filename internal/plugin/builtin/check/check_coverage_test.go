// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package check_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin/check"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

func newCtx(returnsErr bool) plugin.GenerationContext {
	return plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option", OptionReturnsError: returnsErr}, nil, nil, model.GenericInfo{})
}

func importPaths(ctx plugin.GenerationContext) []string {
	var out []string
	for _, imp := range ctx.GetAdditionalImports() {
		out = append(out, imp.Path)
	}
	return out
}

func TestRequiredCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		field      model.OptField
		returnsErr bool
		want       []string
	}{
		{"pointer panics", model.OptField{FieldName: "name", Type: "*T"}, false, []string{
			"if v == nil {", `  panic(fmt.Errorf("name is required"))`, "}",
		}},
		{"string returns error", model.OptField{FieldName: "name", Type: "string"}, true, []string{
			`if v == "" {`, `  return fmt.Errorf("name is required")`, "}",
		}},
		{"slice", model.OptField{FieldName: "name", Type: "[]int", IsSlice: true}, true, []string{
			"if v == nil || len(v) == 0 {", `  return fmt.Errorf("name is required")`, "}",
		}},
		{"map", model.OptField{FieldName: "name", Type: "map[string]int"}, false, []string{
			"if v == nil || len(v) == 0 {", `  panic(fmt.Errorf("name is required"))`, "}",
		}},
		{"named interface", model.OptField{FieldName: "name", Type: "io.Writer"}, true, []string{
			"switch rv := reflect.ValueOf(v); rv.Kind() {",
			"case reflect.Invalid:",
			`  return fmt.Errorf("name is required")`,
			"case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:",
			"  if rv.IsNil() {",
			`    return fmt.Errorf("name is required")`,
			"  }",
			"}",
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := &check.RequiredCheck{}
			require.NoError(t, c.Validate(tc.field, ""))
			ctx := newCtx(tc.returnsErr)
			require.Equal(t, tc.want, c.Generate(ctx, tc.field, "v", ""))
			require.Contains(t, importPaths(ctx), "fmt")
		})
	}

	for _, typ := range []string{"int", "bool", "time.Duration", "[3]int", "struct{}"} {
		require.ErrorIs(t, (&check.RequiredCheck{}).Validate(model.OptField{Type: typ}, ""), plugin.ErrNotApplicable, typ)
	}
}

func TestLenChecks(t *testing.T) {
	t.Parallel()

	str := model.OptField{FieldName: "tags", Type: "string"}
	slice := model.OptField{FieldName: "tags", Type: "[]string", IsSlice: true}
	mp := model.OptField{FieldName: "tags", Type: "map[string]int"}

	t.Run("minlen string", func(t *testing.T) {
		t.Parallel()
		got := (&check.MinLenCheck{}).Generate(newCtx(true), str, "v", "2")
		require.Equal(t, []string{"if len(v) < (2) {", `  return fmt.Errorf("tags is too short (min 2)")`, "}"}, got)
	})
	t.Run("minlen slice", func(t *testing.T) {
		t.Parallel()
		got := (&check.MinLenCheck{}).Generate(newCtx(false), slice, "v", " 3 ")
		require.Equal(t, "if len(v) < (3) {", got[0])
		require.Equal(t, `  panic(fmt.Errorf("tags has too few elements (min 3)"))`, got[1])
	})
	t.Run("maxlen string", func(t *testing.T) {
		t.Parallel()
		got := (&check.MaxLenCheck{}).Generate(newCtx(true), str, "x", "5")
		require.Equal(t, []string{"if len(x) > (5) {", `  return fmt.Errorf("tags is too long (max 5)")`, "}"}, got)
	})
	t.Run("maxlen map", func(t *testing.T) {
		t.Parallel()
		got := (&check.MaxLenCheck{}).Generate(newCtx(true), mp, "x", "5")
		require.Equal(t, `  return fmt.Errorf("tags has too many elements (max 5)")`, got[1])
	})
	t.Run("supported kinds validate", func(t *testing.T) {
		t.Parallel()
		for _, f := range []model.OptField{str, slice, mp} {
			require.NoError(t, (&check.MinLenCheck{}).Validate(f, "1"))
			require.NoError(t, (&check.MaxLenCheck{}).Validate(f, "1"))
		}
	})
	t.Run("unsupported kind", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, (&check.MaxLenCheck{}).Validate(model.OptField{Type: "*T"}, "5"), plugin.ErrNotApplicable)
		require.ErrorIs(t, (&check.MinLenCheck{}).Validate(model.OptField{Type: "int"}, "5"), plugin.ErrNotApplicable)
	})
}

func TestOneOfCheck(t *testing.T) {
	t.Parallel()

	field := model.OptField{FieldName: "mode", Type: "string"}

	got := (&check.OneOfCheck{}).Generate(newCtx(true), field, "v", `["fast","slow"]`)
	require.Len(t, got, 5)
	require.Equal(t, "switch v {", got[0])
	require.Contains(t, got[1], `"fast"`)
	require.Contains(t, got[1], `"slow"`)
	require.True(t, strings.HasPrefix(got[1], "case "))
	require.Equal(t, "default:", got[2])
	require.Contains(t, got[3], "return fmt.Errorf(")

	unquoted := (&check.OneOfCheck{}).Generate(newCtx(false), field, "v", `[debug,info,warn]`)
	require.Equal(t, `case "debug", "info", "warn":`, unquoted[1])
	require.Contains(t, got[3], "mode has invalid value")

	require.NoError(t, (&check.OneOfCheck{}).Validate(field, `["a"]`))
	require.ErrorIs(t, (&check.OneOfCheck{}).Validate(model.OptField{Type: "[]string", IsSlice: true}, `["a"]`), plugin.ErrNotApplicable)
	require.ErrorIs(t, (&check.OneOfCheck{}).Validate(field, " "), plugin.ErrInvalidCheckValue)
	require.ErrorIs(t, (&check.OneOfCheck{}).Validate(field, "[]"), plugin.ErrInvalidCheckValue)
}

func TestNonZeroCheck(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{"int", "uint64", "float32", "time.Duration", "uintptr"} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			field := model.OptField{FieldName: "n", Type: typ}
			require.NoError(t, (&check.NonZeroCheck{}).Validate(field, ""))
			got := (&check.NonZeroCheck{}).Generate(newCtx(false), field, "v", "")
			require.Equal(t, []string{"if v == 0 {", `  panic(fmt.Errorf("n must be non-zero"))`, "}"}, got)
		})
	}

	for _, field := range []model.OptField{{Type: "bool"}, {Type: "string"}, {Type: "[]int", IsSlice: true, ElemType: "int"}} {
		require.ErrorIs(t, (&check.NonZeroCheck{}).Validate(field, ""), plugin.ErrNotApplicable, field.Type)
	}
}

func TestChecksRegistered(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"required", "nonempty", "minlen", "maxlen", "oneof", "nonzero"} {
		_, _, err := testRegistry().FindCheckPlugin(name)
		require.NoError(t, err, name)
	}
}

func testRegistry() *plugin.Registry {
	r := plugin.NewRegistry()
	if err := check.Register(r); err != nil {
		panic(err)
	}
	return r
}

func TestCheckFailureSkipMode(t *testing.T) {
	t.Parallel()
	ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{CheckFailure: model.CheckFailureSkip}, nil, nil, model.GenericInfo{})
	got := (&check.RequiredCheck{}).Generate(ctx, model.OptField{FieldName: "name", Type: "string"}, "v", "")
	require.Equal(t, []string{`if v == "" {`, "  return", "}"}, got)
	require.Empty(t, importPaths(ctx))

	errCtx := plugin.NewGenerationContext("pkg", model.RenderSettings{OptionReturnsError: true, CheckFailure: model.CheckFailureSkip}, nil, nil, model.GenericInfo{})
	require.Contains(t, (&check.RequiredCheck{}).Generate(errCtx, model.OptField{FieldName: "name", Type: "string"}, "v", "")[1], "return fmt.Errorf",
		"options with an error result always return the error")
}
