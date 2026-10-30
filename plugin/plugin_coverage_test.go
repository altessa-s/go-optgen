// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package plugin_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// --- mock plugin types shared across this file ---

type namedMetaPlugin struct{ name string }

func (p *namedMetaPlugin) Meta() plugin.Meta { return plugin.Meta{Name: p.name} }

type noNameMetaPlugin struct{}

func (p *noNameMetaPlugin) Meta() plugin.Meta { return plugin.Meta{} }

type valueMetaPlugin struct{}

func (valueMetaPlugin) Meta() plugin.Meta { return plugin.Meta{} }

// mockFieldPlugin is a configurable plugin.FieldPlugin for registry tests.
type mockFieldPlugin struct {
	name       string
	priority   int
	applicable []string
	canHandle  func(model.OptField) bool
}

func (m *mockFieldPlugin) Meta() plugin.Meta {
	return plugin.Meta{Name: m.name, Priority: m.priority, ApplicableTypes: m.applicable}
}

func (m *mockFieldPlugin) CanHandle(f model.OptField) bool { return m.canHandle(f) }

func (m *mockFieldPlugin) Generate(_ plugin.GenerationContext, _ model.OptField, _ plugin.Pipeline) (plugin.GeneratedCode, error) {
	return plugin.GeneratedCode{Code: m.name}, nil
}

// mockTransformPlugin is a configurable plugin.TransformPlugin for registry tests.
type mockTransformPlugin struct {
	name       string
	key        string
	applicable []string
	disabledBy string
}

func (m *mockTransformPlugin) Meta() plugin.Meta {
	return plugin.Meta{Name: m.name, ApplicableTypes: m.applicable, DisabledBy: m.disabledBy}
}

func (m *mockTransformPlugin) Key() string { return m.key }

func (m *mockTransformPlugin) Apply(_ plugin.GenerationContext, _ model.OptField, _, expr string) (string, bool) {
	return expr + "!", true
}

// mockCheckPlugin is a configurable plugin.CheckPlugin for registry tests.
// Its name is the spec key.
type mockCheckPlugin struct {
	spec    plugin.CheckSpec
	aliases []string
}

func (m *mockCheckPlugin) Meta() plugin.Meta                     { return plugin.Meta{Name: m.spec.Key} }
func (m *mockCheckPlugin) Spec() plugin.CheckSpec                { return m.spec }
func (m *mockCheckPlugin) Aliases() []string                     { return m.aliases }
func (m *mockCheckPlugin) Validate(model.OptField, string) error { return nil }

func (m *mockCheckPlugin) Generate(_ plugin.GenerationContext, _ model.OptField, valueVar, rawValue string) []string {
	return []string{m.spec.Key, valueVar, rawValue}
}

// mockTypeDefaultPlugin is a configurable plugin.TypeDefaultPlugin for registry tests.
type mockTypeDefaultPlugin struct {
	name     string
	priority int
	types    []string
	value    string
}

func (m *mockTypeDefaultPlugin) Meta() plugin.Meta {
	return plugin.Meta{Name: m.name, Priority: m.priority}
}

func (m *mockTypeDefaultPlugin) CanProvideDefault(typeStr string) bool {
	for _, t := range m.types {
		if t == typeStr {
			return true
		}
	}
	return false
}

func (m *mockTypeDefaultPlugin) GetDefault(string) string { return m.value }

// mockModifierPlugin is a configurable plugin.ModifierPlugin for registry tests.
// Its name is the key.
type mockModifierPlugin struct {
	key       string
	phase     plugin.Phase
	canHandle func(model.OptField) bool
	generate  func(plugin.GenerationContext, model.OptField, string) plugin.ModifierResult
}

func (m *mockModifierPlugin) Meta() plugin.Meta   { return plugin.Meta{Name: m.key} }
func (m *mockModifierPlugin) Key() string         { return m.key }
func (m *mockModifierPlugin) Phase() plugin.Phase { return m.phase }

func (m *mockModifierPlugin) CanHandle(f model.OptField) bool { return m.canHandle(f) }

func (m *mockModifierPlugin) Generate(ctx plugin.GenerationContext, f model.OptField, in string) plugin.ModifierResult {
	return m.generate(ctx, f, in)
}

// --- GetPluginName ---

func TestGetPluginName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    plugin.PluginMeta
		want string
	}{
		{name: "explicit name wins", p: &namedMetaPlugin{name: "Custom"}, want: "Custom"},
		{name: "empty name falls back to reflection", p: &noNameMetaPlugin{}, want: "noNameMetaPlugin"},
		{name: "non-pointer value", p: valueMetaPlugin{}, want: "valueMetaPlugin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, plugin.GetPluginName(tt.p))
		})
	}
}

// --- IsTruthyString / IsTruthyMetadata / HasModifier ---

func TestIsTruthyString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"one", "1", true},
		{"true", "true", true},
		{"TRUE upper", "TRUE", true},
		{"yes", "yes", true},
		{"y short", "y", true},
		{"on", "on", true},
		{"padded", "  yes  ", true},
		{"false", "false", false},
		{"no", "no", false},
		{"empty", "", false},
		{"garbage", "maybe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, plugin.IsTruthyString(tt.in))
		})
	}
}

func TestIsTruthyMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		md   map[string]string
		key  string
		want bool
	}{
		{"nil map", nil, "x", false},
		{"missing key", map[string]string{"a": "1"}, "x", false},
		{"present truthy", map[string]string{"x": "yes"}, "x", true},
		{"present falsy", map[string]string{"x": "0"}, "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, plugin.IsTruthyMetadata(tt.md, tt.key))
		})
	}
}

func TestHasModifier(t *testing.T) {
	t.Parallel()

	require.True(t, plugin.HasModifier([]string{"a", "b"}, "a"))
	require.False(t, plugin.HasModifier([]string{"a", "b"}, "c"))
	require.False(t, plugin.HasModifier(nil, "a"))
}

// --- BuildEarlyReturn ---

func TestBuildEarlyReturn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		returnErr bool
		cond      string
		want      []string
	}{
		{
			name:      "error-returning option",
			returnErr: true,
			cond:      "v == nil",
			want:      []string{"if v == nil {", "  return nil", "}"},
		},
		{
			name:      "bare-return option",
			returnErr: false,
			cond:      "len(v) == 0",
			want:      []string{"if len(v) == 0 {", "  return", "}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "T", OptionType: "Option", OptionReturnsError: tt.returnErr}, nil, nil, model.GenericInfo{})
			require.Equal(t, tt.want, plugin.BuildEarlyReturn(ctx, tt.cond))
		})
	}
}

// --- CheckBase / TransformBase ---

func TestCheckBase(t *testing.T) {
	t.Parallel()

	cb := plugin.NewCheckBase("required", 10, plugin.WithRequiresValue(true), plugin.WithAliases("nonempty", "req"))
	require.Equal(t, plugin.CheckSpec{Key: "required", Priority: 10, RequiresValue: true}, cb.Spec())
	require.Equal(t, []string{"nonempty", "req"}, cb.Aliases())

	plain := plugin.NewCheckBase("plain", 5)
	require.Equal(t, plugin.CheckSpec{Key: "plain", Priority: 5, RequiresValue: false}, plain.Spec())
	require.Nil(t, plain.Aliases())
}

func TestTransformBase(t *testing.T) {
	t.Parallel()

	tb := plugin.NewTransformBase("custom",
		plugin.WithApplicableTypes("string", "int"),
		plugin.WithDefaultForTypes("string"),
		plugin.WithDisabledBy("nocustom"),
	)
	meta := tb.Meta()
	require.Equal(t, plugin.KindTransform, meta.Kind)
	require.Equal(t, []string{"string", "int"}, meta.ApplicableTypes)
	require.Equal(t, plugin.DefaultPriority, meta.Priority)
	require.Equal(t, []string{"string"}, meta.DefaultForTypes)
	require.Equal(t, "nocustom", meta.DisabledBy)
	require.Equal(t, "custom", tb.Key())

	defaultPrio := plugin.NewTransformBase("bare")
	require.Equal(t, plugin.DefaultPriority, defaultPrio.Meta().Priority)
	require.Empty(t, defaultPrio.Meta().ApplicableTypes)
}

// --- GenerationContext ---

func TestGenerationContextGenerics(t *testing.T) {
	t.Parallel()

	generic := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "T", OptionType: "Option", OptionReturnsError: true},
		[]model.OptField{{FieldName: "A"}},
		map[string]model.ImportInfo{"a": {Path: "a"}},
		model.GenericInfo{TypeParams: []model.TypeParam{{Name: "T", Constraint: "any"}}},
	)
	require.True(t, generic.IsGeneric())
	require.Equal(t, "[T any]", generic.TypeParamsDecl())
	require.Equal(t, "[T]", generic.TypeParamsNames())

	nonGeneric := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "T", OptionType: "Option", OptionReturnsError: false}, nil, nil, model.GenericInfo{})
	require.False(t, nonGeneric.IsGeneric())
	require.Equal(t, "", nonGeneric.TypeParamsDecl())
	require.Equal(t, "", nonGeneric.TypeParamsNames())
}

func TestGenerationContextAddImport(t *testing.T) {
	t.Parallel()

	ctx := plugin.NewGenerationContext("pkg", model.RenderSettings{TypeName: "T", OptionType: "Option", OptionReturnsError: false}, nil, nil, model.GenericInfo{})

	ctx.AddImport("strings")        // no alias yet
	ctx.AddImport("strings", "str") // adds alias, should overwrite
	ctx.AddImport("strings")        // no alias given, existing has alias -> keep existing
	ctx.AddImport("fmt")            // distinct path, no alias
	ctx.AddImport("fmt", "")        // explicit empty alias behaves like no alias

	imports := ctx.GetAdditionalImports()
	require.Equal(t, []model.GeneratedImport{
		{Path: "fmt"},
		{Path: "strings", Alias: "str"},
	}, imports)
}

// --- warning sink (exercised through FindPlugin/FindTransformPlugin) ---

// newCapturingRegistry returns a registry whose warnings are collected.
func newCapturingRegistry(tb testing.TB) (*plugin.Registry, *[]string) {
	tb.Helper()
	var msgs []string
	return plugin.NewRegistry(plugin.WithWarningSink(func(msg string) { msgs = append(msgs, msg) })), &msgs
}

// mustRegister registers plugins and fails the test on error.
func mustRegister(tb testing.TB, r *plugin.Registry, plugins ...any) {
	tb.Helper()
	require.NoError(tb, r.Register(plugins...))
}

// --- Registry: field plugins ---
//

func TestRegistryFieldPlugins(t *testing.T) {
	t.Parallel()
	r := plugin.NewRegistry()

	low := &mockFieldPlugin{name: "Low", priority: 1, canHandle: func(model.OptField) bool { return true }}
	high := &mockFieldPlugin{name: "High", priority: 10, canHandle: func(model.OptField) bool { return true }}
	mustRegister(t, r, low, high)

	found, err := r.FindPlugin(model.OptField{FieldName: "X", Type: "string"})
	require.NoError(t, err)
	require.Equal(t, "High", plugin.GetPluginName(found))

	require.NoError(t, r.DisablePlugins("High"))
	found, err = r.FindPlugin(model.OptField{FieldName: "X", Type: "string"})
	require.NoError(t, err)
	require.Equal(t, "Low", plugin.GetPluginName(found))

	require.NoError(t, r.DisablePlugins("Low"))
	_, err = r.FindPlugin(model.OptField{FieldName: "X", Type: "string"})
	require.Error(t, err)

	// Applicable-types mismatch: plugin says CanHandle but is restricted away
	// from this type, so it should be skipped and a warning emitted.
	restricted, warnings := newCapturingRegistry(t)
	mustRegister(t, restricted, &mockFieldPlugin{
		name:       "Restricted",
		applicable: []string{"int"},
		canHandle:  func(model.OptField) bool { return true },
	})
	_, err = restricted.FindPlugin(model.OptField{FieldName: "X", Type: "string"})
	require.Error(t, err)
	require.NotEmpty(t, *warnings, "expected warning for applicable-types mismatch")
}

// --- Registry: check plugins ---

func TestRegistryCheckPlugins(t *testing.T) {
	t.Parallel()

	r := plugin.NewRegistry()
	require.ErrorIs(t, r.Register(nil), plugin.ErrUnsupportedPlugin)
	require.ErrorIs(t, r.Register(&mockCheckPlugin{spec: plugin.CheckSpec{Key: "  "}}), plugin.ErrUnsupportedPlugin)
	mustRegister(t, r, &mockCheckPlugin{spec: plugin.CheckSpec{Key: "foo", Priority: 1}, aliases: []string{"bar", " ", "foo"}})

	h, spec, err := r.FindCheckPlugin("foo")
	require.NoError(t, err)
	require.Equal(t, 1, spec.Priority)
	require.Equal(t, []string{"foo", "v", "1"}, h.Generate(plugin.GenerationContext{}, model.OptField{}, "v", "1"))

	h2, spec2, err := r.FindCheckPlugin(" BAR ")
	require.NoError(t, err)
	require.Equal(t, "bar", spec2.Key) // alias spec reports the queried alias key
	require.Equal(t, spec.Priority, spec2.Priority)
	require.Equal(t, plugin.GetPluginName(h), plugin.GetPluginName(h2))

	_, _, err = r.FindCheckPlugin("missing")
	require.ErrorIs(t, err, plugin.ErrUnknownCheck)
	_, _, err = r.FindCheckPlugin("")
	require.ErrorIs(t, err, plugin.ErrUnknownCheck)

	require.NoError(t, r.DisablePlugins("FOO"))
	for _, key := range []string{"foo", "bar"} {
		_, _, err = r.FindCheckPlugin(key)
		require.ErrorIs(t, err, plugin.ErrPluginDisabled)
	}

	descriptors := r.Descriptors()
	require.Len(t, descriptors, 1)
	require.Equal(t, plugin.Descriptor{
		Name: "foo", Kind: plugin.KindOptCheck, Key: "foo", Aliases: []string{"bar"}, Phase: plugin.PhaseCheck,
		Meta: plugin.Meta{Name: "foo"}, Disabled: true,
	}, descriptors[0])
}

// --- Registry: transform plugins ---

func TestRegistryTransformPlugins(t *testing.T) {
	t.Parallel()
	r, warnings := newCapturingRegistry(t)
	mustRegister(t, r, &mockTransformPlugin{name: "Foo", key: "Foo", applicable: []string{"string"}})

	tp, ok := r.FindTransformPlugin("foo", "string")
	require.True(t, ok)
	out, applied := tp.Apply(plugin.GenerationContext{}, model.OptField{}, "string", "v")
	require.True(t, applied)
	require.Equal(t, "v!", out)

	// Parameterized key (e.g. "foo=bar") resolves to the base key.
	tp, ok = r.FindTransformPlugin("foo=bar", "string")
	require.True(t, ok)
	require.NotNil(t, tp)

	_, ok = r.FindTransformPlugin("unknown", "string")
	require.False(t, ok)

	_, ok = r.FindTransformPlugin("", "string")
	require.False(t, ok)

	_, ok = r.FindTransformPlugin("foo", "int")
	require.False(t, ok)
	require.NotEmpty(t, *warnings, "expected warning for applicable-types mismatch")

	require.NoError(t, r.DisablePlugins("Foo"))
	_, ok = r.FindTransformPlugin("foo", "string")
	require.False(t, ok)
}

// --- Registry: modifier pipeline ---

func TestRegistryModifierPipeline(t *testing.T) {
	t.Parallel()

	r := plugin.NewRegistry()

	guard := &mockModifierPlugin{
		key:   "myguard",
		phase: plugin.PhaseGuard,
		canHandle: func(f model.OptField) bool {
			_, ok := f.Metadata["myguard"]
			return ok
		},
		generate: func(_ plugin.GenerationContext, _ model.OptField, in string) plugin.ModifierResult {
			return plugin.ModifierResult{Code: []string{"guarded"}, OutputVar: in, Final: true}
		},
	}
	post := &mockModifierPlugin{
		key:       "mypost",
		phase:     plugin.PhasePostProcess,
		canHandle: func(model.OptField) bool { return true },
		generate: func(_ plugin.GenerationContext, _ model.OptField, in string) plugin.ModifierResult {
			return plugin.ModifierResult{Code: []string{"posted"}, OutputVar: in}
		},
	}
	check := &mockModifierPlugin{
		key:       "mycheck",
		phase:     plugin.PhaseCheck,
		canHandle: func(model.OptField) bool { return true },
		generate: func(_ plugin.GenerationContext, _ model.OptField, in string) plugin.ModifierResult {
			return plugin.ModifierResult{Code: []string{"checked"}, OutputVar: in}
		},
	}
	mustRegister(t, r, guard, post, check)

	// "mycheck" is sourced via field.Modifiers so this test stays about phase ordering.
	guardedField := model.OptField{
		FieldName: "X",
		Metadata:  map[string]string{"myguard": ""},
		Modifiers: []string{"mypost", "mycheck"},
	}
	mods := r.CollectModifiers(guardedField)
	require.Len(t, mods, 3)
	require.Equal(t, "myguard", mods[0].Key()) // guard phase sorts first

	result := plugin.NewPipeline(r).BuildPipeline(plugin.GenerationContext{}, guardedField, "v")
	require.True(t, result.EarlyReturn)
	require.Equal(t, "_ = v\nguarded", result.Code)

	plainField := model.OptField{
		FieldName: "Y",
		Modifiers: []string{"mypost", "mycheck"},
	}
	result = plugin.NewPipeline(r).BuildPipeline(plugin.GenerationContext{}, plainField, "v")
	require.False(t, result.EarlyReturn)
	require.Equal(t, "posted\nchecked", result.Code)

	noModifiers := plugin.NewPipeline(r).BuildPipeline(plugin.GenerationContext{}, model.OptField{FieldName: "Z"}, "v")
	require.Equal(t, plugin.PipelineResult{ValueVar: "v"}, noModifiers)

	descriptors := r.Descriptors()
	require.Len(t, descriptors, 3)
	require.Equal(t, "myguard", descriptors[0].Key)
	require.Equal(t, plugin.PhaseGuard, descriptors[0].Phase)
	require.Equal(t, plugin.KindModifier, descriptors[0].Kind)
}

// TestRegistryCollectModifiersIgnoresChecks pins that optcheck keys address
// check plugins only: a modifier sharing a check key is not collected.
func TestRegistryCollectModifiersIgnoresChecks(t *testing.T) {
	t.Parallel()

	r := plugin.NewRegistry()
	mustRegister(t, r, &mockModifierPlugin{
		key:       "nonempty",
		phase:     plugin.PhasePostProcess,
		canHandle: func(model.OptField) bool { return true },
		generate: func(_ plugin.GenerationContext, _ model.OptField, in string) plugin.ModifierResult {
			return plugin.ModifierResult{OutputVar: in}
		},
	})
	require.Empty(t, r.CollectModifiers(model.OptField{FieldName: "X", Checks: map[string]string{"nonempty": ""}}))
	require.Len(t, r.CollectModifiers(model.OptField{FieldName: "X", Modifiers: []string{"nonempty"}}), 1)
}

func TestPhaseString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		phase plugin.Phase
		want  string
	}{
		{plugin.PhaseGuard, "guard"},
		{plugin.PhaseTransform, "transform"},
		{plugin.PhasePostProcess, "postprocess"},
		{plugin.PhaseCheck, "check"},
		{plugin.Phase(999), "unknown"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, tt.phase.String())
	}
}

// --- Registry: type default plugins ---

func TestRegistryTypeDefaultPlugins(t *testing.T) {
	t.Parallel()

	r := plugin.NewRegistry()
	mustRegister(t, r,
		&mockTypeDefaultPlugin{name: "Low", priority: 1, types: []string{"custom"}, value: "lowDefault()"},
		&mockTypeDefaultPlugin{name: "High", priority: 10, types: []string{"custom"}, value: "highDefault()"},
	)

	require.Equal(t, "highDefault()", r.FindTypeDefaultPlugin("custom"))
	require.Equal(t, "", r.FindTypeDefaultPlugin("nomatch"))

	require.NoError(t, r.DisablePlugins("High"))
	require.Equal(t, "lowDefault()", r.FindTypeDefaultPlugin("custom"))

	descriptors := r.Descriptors()
	require.Len(t, descriptors, 2)
	require.Equal(t, "High", descriptors[0].Name) // higher priority first
	require.True(t, descriptors[0].Disabled)
	require.Equal(t, plugin.KindTypeDefault, descriptors[1].Kind)
}

// --- Registry: descriptors and type patterns ---

func TestRegistryDescriptorsOrderByKind(t *testing.T) {
	t.Parallel()

	r := plugin.NewRegistry()
	mustRegister(t, r,
		&mockCheckPlugin{spec: plugin.CheckSpec{Key: "ck"}},
		&mockModifierPlugin{key: "mp1", phase: plugin.PhaseGuard},
		&mockTransformPlugin{name: "TP1", key: "tp1"},
		&mockTypeDefaultPlugin{name: "TD1"},
		&mockFieldPlugin{name: "FP1", priority: 5},
	)

	var kinds []plugin.Kind
	var names []string
	for _, d := range r.Descriptors() {
		kinds = append(kinds, d.Kind)
		names = append(names, d.Name)
	}
	require.Equal(t, []plugin.Kind{plugin.KindField, plugin.KindTypeDefault, plugin.KindTransform, plugin.KindModifier, plugin.KindOptCheck}, kinds)
	require.Equal(t, []string{"FP1", "TD1", "TP1", "mp1", "ck"}, names)
	require.Empty(t, plugin.NewRegistry().Descriptors())
}

func TestMatchesType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		patterns  []string
		fieldType string
		wantMatch bool
	}{
		{"slice wildcard matches", []string{"[]"}, "[]int", true},
		{"slice wildcard rejects non-slice", []string{"[]"}, "int", false},
		{"pointer wildcard matches", []string{"*"}, "*Foo", true},
		{"interface wildcard matches any", []string{"interface"}, "any", true},
		{"interface wildcard matches interface{}", []string{"interface"}, "interface{}", true},
		{"exact match", []string{" string "}, "string", true},
		{"no match", []string{"string"}, "int", false},
		{"no patterns", nil, "int", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.wantMatch, plugin.MatchesType(tt.fieldType, tt.patterns))
		})
	}
}

// --- Instance registration ---

type globalTestField struct{}

func (globalTestField) Meta() plugin.Meta {
	return plugin.Meta{Name: "GlobalTestField", Priority: 5, ApplicableTypes: []string{"globaltesttype"}}
}
func (globalTestField) CanHandle(model.OptField) bool { return true }
func (globalTestField) Generate(plugin.GenerationContext, model.OptField, plugin.Pipeline) (plugin.GeneratedCode, error) {
	return plugin.GeneratedCode{Code: "x"}, nil
}

type globalTestTransform struct{}

func (globalTestTransform) Meta() plugin.Meta {
	return plugin.Meta{Name: "GlobalTestTransform", ApplicableTypes: []string{"string"}}
}
func (globalTestTransform) Key() string { return "globaltesttransform" }
func (globalTestTransform) Apply(_ plugin.GenerationContext, _ model.OptField, _, expr string) (string, bool) {
	return expr + "!", true
}

type globalTestCheck struct{}

func (globalTestCheck) Meta() plugin.Meta                     { return plugin.Meta{} }
func (globalTestCheck) Validate(model.OptField, string) error { return nil }
func (globalTestCheck) Spec() plugin.CheckSpec {
	return plugin.CheckSpec{Key: "globaltestcheck", Priority: 1}
}
func (globalTestCheck) Aliases() []string { return nil }
func (globalTestCheck) Generate(plugin.GenerationContext, model.OptField, string, string) []string {
	return []string{"checked"}
}

type globalTestTypeDefault struct{}

func (globalTestTypeDefault) Meta() plugin.Meta { return plugin.Meta{Name: "GlobalTestTypeDefault"} }
func (globalTestTypeDefault) CanProvideDefault(typeStr string) bool {
	return typeStr == "globaltesttype2"
}
func (globalTestTypeDefault) GetDefault(string) string { return "defaultVal()" }

type globalTestModifier struct{}

func (globalTestModifier) Meta() plugin.Meta   { return plugin.Meta{} }
func (globalTestModifier) Key() string         { return "globaltestmod" }
func (globalTestModifier) Phase() plugin.Phase { return plugin.PhaseGuard }
func (globalTestModifier) CanHandle(f model.OptField) bool {
	_, ok := f.Metadata["globaltestmod"]
	return ok
}
func (globalTestModifier) Generate(_ plugin.GenerationContext, _ model.OptField, in string) plugin.ModifierResult {
	return plugin.ModifierResult{Code: []string{"guarded"}, OutputVar: in, Final: true}
}

func TestRegistryRegistration(t *testing.T) {
	t.Parallel()
	r, warnings := newCapturingRegistry(t)

	require.ErrorIs(t, r.Register(42), plugin.ErrUnsupportedPlugin)
	require.ErrorIs(t, r.Register((*mockFieldPlugin)(nil)), plugin.ErrUnsupportedPlugin)

	mustRegister(t, r, globalTestField{}, globalTestTransform{}, globalTestCheck{}, globalTestTypeDefault{}, globalTestModifier{})

	field := model.OptField{FieldName: "X", Type: "globaltesttype"}
	found, err := r.FindPlugin(field)
	require.NoError(t, err)
	require.Equal(t, "GlobalTestField", plugin.GetPluginName(found))

	_, err = r.FindPlugin(model.OptField{FieldName: "Y", Type: "othertype"})
	require.Error(t, err)
	require.NotEmpty(t, *warnings, "expected warning for applicable-types mismatch")

	require.NoError(t, r.DisablePlugins("GlobalTestField"))
	_, err = r.FindPlugin(field)
	require.Error(t, err)

	tp, ok := r.FindTransformPlugin("globaltesttransform", "string")
	require.True(t, ok)
	out, applied := tp.Apply(plugin.GenerationContext{}, model.OptField{}, "string", "v")
	require.True(t, applied)
	require.Equal(t, "v!", out)

	_, ok = r.FindTransformPlugin("globaltesttransform", "int")
	require.False(t, ok)

	cp, spec, err := r.FindCheckPlugin("globaltestcheck")
	require.NoError(t, err)
	require.Equal(t, 1, spec.Priority)
	require.Equal(t, []string{"checked"}, cp.Generate(plugin.GenerationContext{}, model.OptField{}, "v", ""))

	require.Equal(t, "defaultVal()", r.FindTypeDefaultPlugin("globaltesttype2"))

	guardField := model.OptField{FieldName: "Z", Metadata: map[string]string{"globaltestmod": ""}}
	mods := r.CollectModifiers(guardField)
	require.Len(t, mods, 1)
	require.Equal(t, "globaltestmod", mods[0].Key())

	result := plugin.NewPipeline(r).BuildPipeline(plugin.GenerationContext{}, guardField, "v")
	require.True(t, result.EarlyReturn)
	require.Contains(t, result.Code, "guarded")
}

// ambiguousPlugin implements both the field and the type default contracts.
type ambiguousPlugin struct{ globalTestField }

func (ambiguousPlugin) CanProvideDefault(string) bool { return false }
func (ambiguousPlugin) GetDefault(string) string      { return "" }

func TestRegistryRejectsConflicts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		first   any
		second  any
		wantErr error
	}{
		{"ambiguous contracts", nil, ambiguousPlugin{}, plugin.ErrAmbiguousPlugin},
		{"transform and modifier share a key",
			&mockTransformPlugin{name: "T", key: "same"}, &mockModifierPlugin{key: "SAME"}, plugin.ErrDuplicateKey},
		{"modifier and transform share a key",
			&mockModifierPlugin{key: "same"}, &mockTransformPlugin{name: "T", key: "same"}, plugin.ErrDuplicateKey},
		{"check alias reuses a check key",
			&mockCheckPlugin{spec: plugin.CheckSpec{Key: "required"}},
			&mockCheckPlugin{spec: plugin.CheckSpec{Key: "other"}, aliases: []string{"Required"}}, plugin.ErrDuplicateKey},
		{"plugin names are unique across kinds",
			&mockFieldPlugin{name: "Same"}, &mockTypeDefaultPlugin{name: "same"}, plugin.ErrDuplicateName},
		{"empty modifier key", nil, &mockModifierPlugin{key: " "}, plugin.ErrUnsupportedPlugin},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := plugin.NewRegistry()
			if tt.first != nil {
				mustRegister(t, r, tt.first)
			}
			require.ErrorIs(t, r.Register(tt.second), tt.wantErr)
		})
	}
}

func TestRegistryCheckAndValueKeysUseSeparateNamespaces(t *testing.T) {
	t.Parallel()
	r := plugin.NewRegistry()
	mustRegister(t, r, &mockModifierPlugin{key: "nonempty"}, &mockCheckPlugin{spec: plugin.CheckSpec{Key: "required"}, aliases: []string{"nonempty"}})
	_, _, err := r.FindCheckPlugin("nonempty")
	require.NoError(t, err)
}

func TestRegistryDisablePluginsRejectsUnknownNames(t *testing.T) {
	t.Parallel()
	r := plugin.NewRegistry()
	mustRegister(t, r, &mockFieldPlugin{name: "Known"})
	err := r.DisablePlugins("known", "missing")
	require.ErrorIs(t, err, plugin.ErrUnknownPlugin)
	require.ErrorContains(t, err, `"missing" (known: Known)`)
	require.False(t, r.Descriptors()[0].Disabled, "a failed call disables nothing")
	require.NoError(t, r.DisablePlugins(" ", ""))
}

// TestRegistryWarningSinkMayReenter pins that warnings are delivered without
// the registry lock held, so a sink may call back into the registry.
func TestRegistryWarningSinkMayReenter(t *testing.T) {
	t.Parallel()
	var r *plugin.Registry
	var names []string
	r = plugin.NewRegistry(plugin.WithWarningSink(func(string) {
		names = append(names, r.Descriptors()[0].Name)
		_ = r.Register(&mockTypeDefaultPlugin{name: "Late"})
	}))
	mustRegister(t, r, &mockFieldPlugin{name: "Restricted", applicable: []string{"int"}, canHandle: func(model.OptField) bool { return true }})
	_, err := r.FindPlugin(model.OptField{FieldName: "X", Type: "string"})
	require.Error(t, err)
	require.Equal(t, []string{"Restricted"}, names)
}
