// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package semantic

import (
	"cmp"
	"errors"
	"fmt"
	"go/token"
	"maps"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// Catalog supplies declaration policies without owning parsing or rendering.
type Catalog interface {
	Descriptors() []plugin.Descriptor
	FindTypeDefaultPlugin(string) string
	NormalizeDefault(typeStr, def string) string
	FindCheckPlugin(string) (plugin.CheckPlugin, plugin.CheckSpec, error)
}

// Preparer applies a catalog's policies to declarations.
type Preparer struct {
	catalog Catalog
	options *options
}

// New binds the required catalog to a semantic preparer.
func New(catalog Catalog, opts ...Option) *Preparer {
	return &Preparer{catalog: catalog, options: newOptions(opts...)}
}

// Prepare validates declarations and returns enriched copies: option names
// carry the configured prefix, checks of disabled plugins are dropped with a
// warning, and type defaults and default modifiers are resolved.
func (p *Preparer) Prepare(fields []model.OptField) ([]model.OptField, error) {
	out := slices.DeleteFunc(slices.Clone(fields), p.excluded)
	policy := newModifierPolicy(p.catalog.Descriptors())
	if !p.options.defaultModifiers {
		policy.defaults = nil
	}
	for i := range out {
		f := &out[i]
		if first, _ := utf8.DecodeRuneInString(f.FieldName); unicode.IsUpper(first) {
			p.warn(fmt.Sprintf("field %s is exported; keep option fields unexported so callers configure through options", f.FieldName))
		}
		f.OptionName = p.options.prefix + f.OptionName
		f.Mandatory = f.Mandatory || f.Metadata["mandatory"] == "true"
		if !f.DefaultOnly && f.OptionName != "" && !token.IsIdentifier(f.OptionName) {
			return nil, fmt.Errorf("field %s: invalid option name %q", f.FieldName, f.OptionName)
		}
		if err := p.prepareChecks(f); err != nil {
			return nil, err
		}
		if f.Default == "" {
			f.Default = p.catalog.FindTypeDefaultPlugin(f.Type)
		} else {
			f.Default = p.catalog.NormalizeDefault(f.Type, f.Default)
		}
		f.Modifiers = policy.modifiers(*f)
		f.HasModifiers = len(f.Modifiers) > 0
	}
	return out, nil
}

// maxIsSetFields is the capacity of the uint64 field set behind WithIsSet.
const maxIsSetFields = 64

// PrepareInput prepares in's fields with [Preparer.Prepare] and applies the
// settings' policies: [model.DefaultsFromNone] drops every default, and
// WithIsSet requires a FieldSet field and at most 64 fields.
func (p *Preparer) PrepareInput(in model.GenerateInput) (model.GenerateInput, error) {
	fields, err := p.Prepare(in.Fields)
	if err != nil {
		return model.GenerateInput{}, err
	}
	if in.DefaultsFrom == model.DefaultsFromNone {
		for i := range fields {
			fields[i].Default = ""
		}
	}
	if in.WithIsSet {
		if in.IsSetField == "" {
			return model.GenerateInput{}, fmt.Errorf("with-isset: declare a field of type %sFieldSet in %s", in.TypeName, in.TypeName)
		}
		if len(fields) > maxIsSetFields {
			return model.GenerateInput{}, fmt.Errorf("with-isset: %s has %d fields, at most %d are supported", in.TypeName, len(fields), maxIsSetFields)
		}
	}
	in.Fields = fields
	return in, nil
}

// prepareChecks validates every optcheck key against its plugin and drops the
// keys of disabled plugins.
func (p *Preparer) prepareChecks(f *model.OptField) error {
	if len(f.Checks) == 0 {
		return nil
	}
	checks := maps.Clone(f.Checks)
	for _, key := range slices.Sorted(maps.Keys(f.Checks)) {
		check, spec, err := p.catalog.FindCheckPlugin(key)
		if errors.Is(err, plugin.ErrPluginDisabled) {
			p.warn(fmt.Sprintf("field %s: check skipped: %v", f.FieldName, err))
			delete(checks, key)
			continue
		}
		if err != nil {
			return fmt.Errorf("field %s: %w", f.FieldName, err)
		}
		value := f.Checks[key]
		if spec.RequiresValue && (f.BareChecks[key] || strings.TrimSpace(value) == "") {
			return fmt.Errorf("field %s: optcheck %s requires a value", f.FieldName, key)
		}
		if err := check.Validate(*f, value); err != nil {
			return fmt.Errorf("field %s: optcheck %s: %w", f.FieldName, key, err)
		}
	}
	f.Checks = checks
	return nil
}

func (p *Preparer) warn(msg string) {
	if p.options.warningSink != nil {
		p.options.warningSink(msg)
	}
}

// modifierPolicy resolves the modifiers that transform and modifier plugins
// apply by default, and the disabler keys that suppress them.
type modifierPolicy struct {
	defaults  []plugin.Descriptor
	disablers []string
}

func newModifierPolicy(descriptors []plugin.Descriptor) modifierPolicy {
	var policy modifierPolicy
	for _, d := range descriptors {
		if d.Kind != plugin.KindTransform && d.Kind != plugin.KindModifier {
			continue
		}
		if d.Meta.DisabledBy != "" && !slices.Contains(policy.disablers, d.Meta.DisabledBy) {
			policy.disablers = append(policy.disablers, d.Meta.DisabledBy)
		}
		if !d.Disabled && len(d.Meta.DefaultForTypes) > 0 {
			policy.defaults = append(policy.defaults, d)
		}
	}
	slices.SortFunc(policy.defaults, func(a, b plugin.Descriptor) int {
		return cmp.Or(cmp.Compare(b.Meta.Priority, a.Meta.Priority), cmp.Compare(a.Key, b.Key))
	})
	return policy
}

// modifiers returns default modifiers (slice element defaults first) followed
// by the explicit ones, without disabler keys.
func (p modifierPolicy) modifiers(f model.OptField) []string {
	existing := slices.Clone(f.Modifiers)
	for key := range f.Metadata {
		existing = append(existing, key)
	}
	var defaults []string
	if f.IsSlice {
		defaults = append(defaults, p.defaultsFor(f.ElemType, existing)...)
	}
	defaults = append(defaults, p.defaultsFor(f.Type, existing)...)
	var mods []string
	for _, mod := range defaults {
		if !slices.Contains(p.disablers, mod) && !slices.Contains(mods, mod) {
			mods = append(mods, mod)
		}
	}
	for _, mod := range f.Modifiers {
		if !slices.Contains(p.disablers, mod) {
			mods = append(mods, mod)
		}
	}
	return mods
}

// defaultsFor returns the keys of plugins applied by default to typeStr,
// excluding those already present in existing or disabled by a key in it.
func (p modifierPolicy) defaultsFor(typeStr string, existing []string) []string {
	var keys []string
	for _, d := range p.defaults {
		if !plugin.MatchesType(typeStr, d.Meta.DefaultForTypes) {
			continue
		}
		if d.Meta.DisabledBy != "" && slices.Contains(existing, d.Meta.DisabledBy) {
			continue
		}
		if slices.ContainsFunc(existing, func(mod string) bool { return modifierKey(mod) == d.Key }) {
			continue
		}
		keys = append(keys, d.Key)
	}
	return keys
}

// modifierKey returns the normalized key of a modifier such as "positive=allow_zero".
func modifierKey(mod string) string {
	key, _, _ := strings.Cut(mod, "=")
	return strings.ToLower(strings.TrimSpace(key))
}

// excluded reports whether f matches an exclude pattern.
func (p *Preparer) excluded(f model.OptField) bool {
	return slices.ContainsFunc(p.options.exclude, func(pattern string) bool {
		matched, err := path.Match(pattern, f.FieldName)
		return err == nil && matched
	})
}
