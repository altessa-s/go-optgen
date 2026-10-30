// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fields_test

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/fields"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

func newCtx(returnsErr bool) plugin.GenerationContext {
	return plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option", OptionReturnsError: returnsErr}, nil, nil, model.GenericInfo{})
}

func genericCtx() plugin.GenerationContext {
	return plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "options", OptionType: "Option", OptionReturnsError: false}, nil, nil,
		model.GenericInfo{TypeParams: []model.TypeParam{{Name: "T", Constraint: "any"}}})
}

// requireParses asserts the generated code is syntactically valid Go.
func requireParses(tb testing.TB, code string) {
	tb.Helper()
	_, err := parser.ParseFile(token.NewFileSet(), "gen.go", "package pkg\n\n"+code, 0)
	require.NoError(tb, err, code)
}

func TestManualPlugin(t *testing.T) {
	t.Parallel()

	p := &fields.ManualPlugin{}
	require.Equal(t, 110, p.Meta().Priority)
	require.True(t, p.CanHandle(model.OptField{Metadata: map[string]string{"manual": "true"}}))
	require.True(t, p.CanHandle(model.OptField{DefaultOnly: true}))
	require.False(t, p.CanHandle(model.OptField{Type: "int"}))

	code, err := p.Generate(newCtx(false), model.OptField{}, plugin.NewPipeline(testRegistry()))
	require.NoError(t, err)
	require.Empty(t, code.Code)
}

func TestBoolFlagPlugin(t *testing.T) {
	t.Parallel()

	p := &fields.BoolFlagPlugin{}
	require.Equal(t, 20, p.Meta().Priority)
	require.True(t, p.CanHandle(model.OptField{Type: "bool"}))
	require.False(t, p.CanHandle(model.OptField{Type: "bool", IsSlice: true}))
	require.False(t, p.CanHandle(model.OptField{Type: "int"}))

	tests := []struct {
		name       string
		field      model.OptField
		returnsErr bool
		contains   []string
	}{
		{"enable", model.OptField{FieldName: "enabled", OptionName: "Enabled", Type: "bool"}, false,
			[]string{"func WithEnabled() Option", "o.enabled = value", "enables the enabled option"}},
		{"invert", model.OptField{
			FieldName: "skipValidation", OptionName: "Validation", Type: "bool",
			Modifiers: []string{"invert"}, HasModifiers: true,
		}, true, []string{"func WithoutValidation() Option", "o.skipValidation = value", "return nil"}},
		{"param", model.OptField{
			FieldName: "strict", OptionName: "Strict", Type: "bool",
			Modifiers: []string{"param"}, HasModifiers: true,
		}, false, []string{"func WithStrict(v bool) Option", "o.strict = v"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code, err := p.Generate(newCtx(tc.returnsErr), tc.field, plugin.NewPipeline(testRegistry()))
			require.NoError(t, err)
			for _, s := range tc.contains {
				require.Contains(t, code.Code, s)
			}
			requireParses(t, code.Code)
		})
	}
}

func TestSliceSetAndAppender(t *testing.T) {
	t.Parallel()

	set := &fields.SliceSetPlugin{}
	app := &fields.DefaultAppenderPlugin{}
	appendMD := map[string]string{"append": "true"}

	require.False(t, set.CanHandle(model.OptField{Type: "int"}))
	require.True(t, set.CanHandle(model.OptField{IsSlice: true, Type: "[]int", ElemType: "int"}))
	require.False(t, set.CanHandle(model.OptField{IsSlice: true, Metadata: appendMD}))

	require.False(t, app.CanHandle(model.OptField{Type: "int", Metadata: appendMD}))
	require.False(t, app.CanHandle(model.OptField{IsSlice: true}))
	require.True(t, app.CanHandle(model.OptField{IsSlice: true, Metadata: appendMD}))
	require.Zero(t, app.Meta().Priority)
	require.Equal(t, 35, set.Meta().Priority)

	field := model.OptField{FieldName: "ports", OptionName: "Ports", Type: "[]int", ElemType: "int", IsSlice: true}

	code, err := set.Generate(newCtx(false), field, plugin.NewPipeline(testRegistry()))
	require.NoError(t, err)
	require.Contains(t, code.Code, "func WithPorts(v ...int) Option")
	require.Contains(t, code.Code, "o.ports = ")
	require.NotContains(t, code.Code, "append(o.ports")
	requireParses(t, code.Code)

	field.Metadata = appendMD
	code, err = app.Generate(newCtx(true), field, plugin.NewPipeline(testRegistry()))
	require.NoError(t, err)
	require.Contains(t, code.Code, "appends to the ports option")
	require.Contains(t, code.Code, "o.ports = append(o.ports, ")
	requireParses(t, code.Code)
}

func TestNetIPParsePlugin(t *testing.T) {
	t.Parallel()

	p := &fields.NetIPParsePlugin{}
	md := map[string]string{"parseip": "true"}
	require.Equal(t, 40, p.Meta().Priority)
	require.False(t, p.CanHandle(model.OptField{IsSlice: true, ElemType: "string", Metadata: md}))
	require.False(t, p.CanHandle(model.OptField{IsSlice: true, ElemType: "net.IP"}))
	require.True(t, p.CanHandle(model.OptField{IsSlice: true, ElemType: "net.IP", Metadata: md}))

	base := model.OptField{FieldName: "ips", OptionName: "IPs", Type: "[]net.IP", ElemType: "net.IP", IsSlice: true}

	t.Run("replace, panic-free mode", func(t *testing.T) {
		t.Parallel()
		f := base
		f.Metadata = map[string]string{"parseip": "true"}
		ctx := newCtx(false)
		code, err := p.Generate(ctx, f, plugin.NewPipeline(testRegistry()))
		require.NoError(t, err)
		require.Contains(t, code.Code, "func WithIPs(v ...string) Option")
		require.Contains(t, code.Code, "sets the ips option")
		require.Contains(t, code.Code, "o.ips = netIPs")
		requireParses(t, code.Code)
		var paths []string
		for _, imp := range ctx.GetAdditionalImports() {
			paths = append(paths, imp.Path)
		}
		require.Contains(t, paths, "net")
		require.NotContains(t, paths, "fmt")
	})

	t.Run("append with default error", func(t *testing.T) {
		t.Parallel()
		f := base
		f.Metadata = map[string]string{"parseip": "true", "append": "true"}
		ctx := newCtx(true)
		code, err := p.Generate(ctx, f, plugin.NewPipeline(testRegistry()))
		require.NoError(t, err)
		require.Contains(t, code.Code, "o.ips = append(o.ips, netIPs...)")
		require.Contains(t, code.Code, `return fmt.Errorf("invalid IP %q", s)`)
		requireParses(t, code.Code)
		var paths []string
		for _, imp := range ctx.GetAdditionalImports() {
			paths = append(paths, imp.Path)
		}
		require.Contains(t, paths, "fmt")
	})

	t.Run("custom error expression", func(t *testing.T) {
		t.Parallel()
		f := base
		f.Metadata = map[string]string{"parseip": "true", "err": " ErrBadIP "}
		ctx := newCtx(true)
		code, err := p.Generate(ctx, f, plugin.NewPipeline(testRegistry()))
		require.NoError(t, err)
		require.Contains(t, code.Code, "return ErrBadIP")
		for _, imp := range ctx.GetAdditionalImports() {
			require.NotEqual(t, "fmt", imp.Path)
		}
	})
}

func TestDefaultSetterPlugin(t *testing.T) {
	t.Parallel()

	p := &fields.DefaultSetterPlugin{}
	require.True(t, p.CanHandle(model.OptField{Type: "int"}))
	require.False(t, p.CanHandle(model.OptField{IsSlice: true}))

	t.Run("scalar", func(t *testing.T) {
		t.Parallel()
		f := model.OptField{FieldName: "timeout", OptionName: "Timeout", Type: "int"}
		code, err := p.Generate(newCtx(false), f, plugin.NewPipeline(testRegistry()))
		require.NoError(t, err)
		require.Contains(t, code.Code, "func WithTimeout(v int) Option")
		require.Contains(t, code.Code, "o.timeout = ")
		requireParses(t, code.Code)
	})

	t.Run("string uses generic overload with transforms and checks", func(t *testing.T) {
		t.Parallel()
		f := model.OptField{
			FieldName: "name", OptionName: "Name", Type: "string",
			Modifiers: []string{"upper"}, HasModifiers: true,
			Checks: map[string]string{"maxlen": "5"},
		}
		code, err := p.Generate(newCtx(true), f, plugin.NewPipeline(testRegistry()))
		require.NoError(t, err)
		require.Contains(t, code.Code, "func WithName[T interface{string|*string}](v T) Option")
		require.Contains(t, code.Code, "strings.ToUpper(")
		require.Contains(t, code.Code, "*t")
		require.Contains(t, code.Code, "len(transformedValue) > (5)")
		require.Contains(t, code.Code, "return nil")
		requireParses(t, code.Code)
	})

	t.Run("string inside generic struct uses plain setter", func(t *testing.T) {
		t.Parallel()
		f := model.OptField{FieldName: "name", OptionName: "Name", Type: "string"}
		code, err := p.Generate(genericCtx(), f, plugin.NewPipeline(testRegistry()))
		require.NoError(t, err)
		require.Contains(t, code.Code, "func WithName[T any](v string) Option[T]")
		require.False(t, strings.Contains(code.Code, "interface{string|*string}"))
		requireParses(t, code.Code)
	})
}

func testRegistry() *plugin.Registry {
	r := plugin.NewRegistry()
	if err := builtin.Register(r); err != nil {
		panic(err)
	}
	return r
}

func TestRenderSettingsShapeOptions(t *testing.T) {
	t.Parallel()
	values := plugin.NewPipeline(testRegistry())
	ctxWith := func(settings model.RenderSettings) plugin.GenerationContext {
		settings.TypeName, settings.OptionType = "options", "Option"
		return plugin.NewGenerationContext("pkg", settings, nil, nil, model.GenericInfo{})
	}
	str := model.OptField{FieldName: "name", OptionName: "Name", Type: "string"}

	t.Run("custom prefixes", func(t *testing.T) {
		t.Parallel()
		ctx := ctxWith(model.RenderSettings{FuncPrefix: "Set", InvertFuncPrefix: "Unset"})
		code, err := (&fields.DefaultSetterPlugin{}).Generate(ctx, model.OptField{FieldName: "n", OptionName: "N", Type: "int"}, values)
		require.NoError(t, err)
		require.Contains(t, code.Code, "func SetN(v int) Option")
		code, err = (&fields.BoolFlagPlugin{}).Generate(ctx, model.OptField{FieldName: "debug", OptionName: "Debug", Type: "bool",
			Modifiers: []string{"invert"}, HasModifiers: true}, values)
		require.NoError(t, err)
		require.Contains(t, code.Code, "func UnsetDebug() Option")
		requireParses(t, code.Code)
	})

	t.Run("plain string setter skips empty values", func(t *testing.T) {
		t.Parallel()
		code, err := (&fields.DefaultSetterPlugin{}).Generate(ctxWith(model.RenderSettings{StringSetter: model.StringSetterPlain}), str, values)
		require.NoError(t, err)
		require.Contains(t, code.Code, "func WithName(v string) Option")
		require.Contains(t, code.Code, `if v == "" {`)
		requireParses(t, code.Code)
	})

	t.Run("empty strings kept", func(t *testing.T) {
		t.Parallel()
		for _, mode := range []model.StringSetterMode{model.StringSetterGeneric, model.StringSetterPlain} {
			code, err := (&fields.DefaultSetterPlugin{}).Generate(ctxWith(model.RenderSettings{StringSetter: mode, KeepEmptyStrings: true}), str, values)
			require.NoError(t, err)
			require.NotContains(t, code.Code, `== ""`, mode)
			requireParses(t, code.Code)
		}
	})
}
