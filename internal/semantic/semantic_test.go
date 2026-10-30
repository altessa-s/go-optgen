// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package semantic_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin"
	"github.com/altessa-s/go-optgen/internal/semantic"
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

func TestPrepareValidatesWithoutImplicitRegistration(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, key, value          string
		bare, register, wantError bool
	}{
		{name: "empty catalog", key: "required", wantError: true},
		{name: "unknown", key: "unknown", register: true, wantError: true},
		{name: "bare bound", key: "minlen", value: "true", bare: true, register: true, wantError: true},
		{name: "empty bound", key: "minlen", register: true, wantError: true},
		{name: "explicit bound", key: "minlen", value: "3", register: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := plugin.NewRegistry()
			if tc.register {
				require.NoError(t, builtin.Register(r))
			}
			_, err := semantic.New(r).Prepare([]model.OptField{{Type: "string", Checks: map[string]string{tc.key: tc.value}, BareChecks: map[string]bool{tc.key: tc.bare}}})
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPrepareCopiesFieldsAndRespectsDisabledDefaults(t *testing.T) {
	t.Parallel()
	r := builtinRegistry(t)
	input := []model.OptField{{Type: "string", Modifiers: []string{"upper"}}}
	prepared, err := semantic.New(r).Prepare(input)
	require.NoError(t, err)
	require.Equal(t, []string{"trimspaces", "upper"}, prepared[0].Modifiers)
	require.Equal(t, []string{"upper"}, input[0].Modifiers)
	require.NoError(t, r.DisablePlugins("TrimSpacesTransformPlugin"))
	prepared, err = semantic.New(r).Prepare([]model.OptField{{Type: "string"}})
	require.NoError(t, err)
	require.Empty(t, prepared[0].Modifiers)
	prepared, err = semantic.New(r).Prepare([]model.OptField{{Type: "string", Modifiers: []string{"notrim"}}})
	require.NoError(t, err)
	require.Empty(t, prepared[0].Modifiers, "disabler keys are removed even when their plugin is disabled")
}

// TestOptcheckKeysDoNotResolveModifiers pins that a modifier key is not a
// valid optcheck key: checks resolve through check plugins only.
func TestOptcheckKeysDoNotResolveModifiers(t *testing.T) {
	t.Parallel()
	r := plugin.NewRegistry()
	require.NoError(t, r.Register(checkModifier{}))
	_, err := semantic.New(r).Prepare([]model.OptField{{Checks: map[string]string{"custom": "true"}}})
	require.ErrorIs(t, err, plugin.ErrUnknownCheck)
}

type checkModifier struct{}

func (checkModifier) Meta() plugin.Meta             { return plugin.Meta{} }
func (checkModifier) Key() string                   { return "custom" }
func (checkModifier) Phase() plugin.Phase           { return plugin.PhaseCheck }
func (checkModifier) CanHandle(model.OptField) bool { return true }
func (checkModifier) Generate(plugin.GenerationContext, model.OptField, string) plugin.ModifierResult {
	return plugin.ModifierResult{}
}

func TestPrepareRejectsInapplicableChecks(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		field   model.OptField
		wantErr error
	}{
		{"minlen on int", model.OptField{FieldName: "n", Type: "int", Checks: map[string]string{"minlen": "3"}}, plugin.ErrNotApplicable},
		{"required on bool", model.OptField{FieldName: "b", Type: "bool", Checks: map[string]string{"required": ""}}, plugin.ErrNotApplicable},
		{"nonzero on string", model.OptField{FieldName: "s", Type: "string", Checks: map[string]string{"nonzero": ""}}, plugin.ErrNotApplicable},
		{"oneof on int", model.OptField{FieldName: "n", Type: "int", Checks: map[string]string{"oneof": "a"}}, plugin.ErrNotApplicable},
		{"empty oneof list", model.OptField{FieldName: "s", Type: "string", Checks: map[string]string{"oneof": "[]"}}, plugin.ErrInvalidCheckValue},
		{"required on named interface", model.OptField{FieldName: "w", Type: "io.Writer", Checks: map[string]string{"required": ""}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := semantic.New(builtinRegistry(t)).Prepare([]model.OptField{tc.field})
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

func TestPrepareDropsDisabledChecksWithWarning(t *testing.T) {
	t.Parallel()
	r := builtinRegistry(t)
	require.NoError(t, r.DisablePlugins("NonZeroCheck"))
	var warnings []string
	input := []model.OptField{{FieldName: "count", Type: "int", Checks: map[string]string{"nonzero": ""}}}
	prepared, err := semantic.New(r, semantic.WithWarningSink(func(msg string) { warnings = append(warnings, msg) })).Prepare(input)
	require.NoError(t, err)
	require.Empty(t, prepared[0].Checks)
	require.Contains(t, input[0].Checks, "nonzero", "the input map is not modified")
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "NonZeroCheck")
}

func TestPrepareAppliesPrefixBeforeValidation(t *testing.T) {
	t.Parallel()
	r := builtinRegistry(t)
	prepared, err := semantic.New(r, semantic.WithPrefix("Validation")).Prepare([]model.OptField{{FieldName: "issuer", OptionName: "Issuer", Type: "int"}})
	require.NoError(t, err)
	require.Equal(t, "ValidationIssuer", prepared[0].OptionName)
	_, err = semantic.New(r).Prepare([]model.OptField{{FieldName: "x", OptionName: "a-b", Type: "int", Metadata: map[string]string{"manual": "true"}}})
	require.Error(t, err, "manual fields get no special treatment")
	_, err = semantic.New(r).Prepare([]model.OptField{{FieldName: "x", OptionName: "a-b", Type: "int", DefaultOnly: true}})
	require.NoError(t, err)
}

// fakeCatalog serves fixed descriptors for default modifier policy tests.
type fakeCatalog struct{ descriptors []plugin.Descriptor }

func (c fakeCatalog) Descriptors() []plugin.Descriptor  { return c.descriptors }
func (fakeCatalog) FindTypeDefaultPlugin(string) string { return "" }
func (fakeCatalog) FindCheckPlugin(key string) (plugin.CheckPlugin, plugin.CheckSpec, error) {
	return nil, plugin.CheckSpec{}, plugin.ErrUnknownCheck
}

func transformDescriptor(key string, priority int, disabledBy string, defaultFor ...string) plugin.Descriptor {
	return plugin.Descriptor{Name: key, Kind: plugin.KindTransform, Key: key,
		Meta: plugin.Meta{Priority: priority, DefaultForTypes: defaultFor, DisabledBy: disabledBy}}
}

func TestPrepareDefaultModifiers(t *testing.T) {
	t.Parallel()
	trim := transformDescriptor("trimspaces", 10, "notrim", "string")
	lower := transformDescriptor("lower", 5, "", "string")
	disabled := transformDescriptor("upper", 20, "", "string")
	disabled.Disabled = true
	field := plugin.Descriptor{Name: "F", Kind: plugin.KindField, Meta: plugin.Meta{DefaultForTypes: []string{"string"}}}
	tests := []struct {
		name        string
		descriptors []plugin.Descriptor
		field       model.OptField
		want        []string
	}{
		{"no plugins", nil, model.OptField{Type: "string"}, nil},
		{"matching default", []plugin.Descriptor{trim}, model.OptField{Type: "string"}, []string{"trimspaces"}},
		{"not matching", []plugin.Descriptor{trim}, model.OptField{Type: "int"}, nil},
		{"disabler removes the default and itself", []plugin.Descriptor{trim}, model.OptField{Type: "string", Modifiers: []string{"notrim"}}, nil},
		{"explicit modifier is not repeated", []plugin.Descriptor{trim}, model.OptField{Type: "string", Modifiers: []string{"trimspaces"}}, []string{"trimspaces"}},
		{"priority order", []plugin.Descriptor{lower, trim}, model.OptField{Type: "string"}, []string{"trimspaces", "lower"}},
		{"disabler leaves others", []plugin.Descriptor{lower, trim}, model.OptField{Type: "string", Modifiers: []string{"notrim"}}, []string{"lower"}},
		{"disabler in metadata", []plugin.Descriptor{trim}, model.OptField{Type: "string", Metadata: map[string]string{"notrim": "true"}}, nil},
		{"disabled plugins and field plugins are ignored", []plugin.Descriptor{disabled, field}, model.OptField{Type: "string"}, nil},
		{"slice element defaults", []plugin.Descriptor{trim}, model.OptField{Type: "[]string", ElemType: "string", IsSlice: true}, []string{"trimspaces"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			prepared, err := semantic.New(fakeCatalog{tt.descriptors}).Prepare([]model.OptField{tt.field})
			require.NoError(t, err)
			require.Equal(t, tt.want, prepared[0].Modifiers)
			require.Equal(t, len(tt.want) > 0, prepared[0].HasModifiers)
		})
	}
}

func builtinRegistry(tb testing.TB) *plugin.Registry {
	tb.Helper()
	r := plugin.NewRegistry()
	require.NoError(tb, builtin.Register(r))
	return r
}

func TestPreparePreservesExplicitModifierOrder(t *testing.T) {
	t.Parallel()
	p := semantic.New(builtinRegistry(t))
	prepared, err := p.Prepare([]model.OptField{{Type: "string", Modifiers: []string{"notrim", "lower", "upper", "lower"}}})
	require.NoError(t, err)
	require.Equal(t, []string{"lower", "upper", "lower"}, prepared[0].Modifiers)
	prepared, err = p.Prepare([]model.OptField{{Type: "time.Duration", Modifiers: []string{"positive=allow_zero"}}})
	require.NoError(t, err)
	require.Equal(t, []string{"positive=allow_zero"}, prepared[0].Modifiers)
}

func TestPrepareWithoutDefaultModifiers(t *testing.T) {
	t.Parallel()
	p := semantic.New(builtinRegistry(t), semantic.WithDefaultModifiers(false))
	prepared, err := p.Prepare([]model.OptField{
		{Type: "string", Modifiers: []string{"notrim", "upper"}},
		{Type: "[]string", ElemType: "string", IsSlice: true},
		{Type: "*T"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"upper"}, prepared[0].Modifiers, "explicit modifiers stay, disablers are removed")
	require.Empty(t, prepared[1].Modifiers)
	require.Empty(t, prepared[2].Modifiers)
}
