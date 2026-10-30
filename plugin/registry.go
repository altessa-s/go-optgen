// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package plugin

import (
	"cmp"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/altessa-s/go-optgen/model"
)

// normalizeKey normalizes a plugin key to lowercase with trimmed spaces.
func normalizeKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// extractModifierKey extracts the key from a modifier string (before '=').
// Example: "positive=allow_zero" -> "positive"
func extractModifierKey(mod string) string {
	if idx := strings.IndexByte(mod, '='); idx > 0 {
		return mod[:idx]
	}
	return mod
}

// Registry stores plugins for one generation session.
//
// Keys share two namespaces: optgen/optval keys belong to exactly one transform
// or modifier plugin, and optcheck keys (including aliases) to exactly one check
// plugin. Plugin names are unique across all kinds because they address plugins
// for disabling. Register plugins before sharing the registry with concurrent
// readers.
type Registry struct {
	mu                 sync.RWMutex
	options            *registryOptions
	fieldPlugins       []FieldPlugin
	transformPlugins   map[string]TransformPlugin // key -> plugin (keys are normalized to lower)
	typeDefaultPlugins []TypeDefaultPlugin
	modifierPlugins    map[string]ModifierPlugin // key -> modifier plugin (unified pipeline)
	checkSpecs         map[string]CheckSpec
	checkPlugins       map[string]CheckPlugin
	names              map[string]bool // normalized names of registered plugins
	disabledPlugins    map[string]bool // normalized names of disabled plugins
}

// Descriptor is a read-only snapshot of a registered plugin, used for
// listing plugins and for policies derived from plugin metadata.
type Descriptor struct {
	Name string
	// Kind is the plugin contract the plugin was registered under.
	Kind Kind
	// Key is the tag key of transform, modifier and check plugins.
	Key string
	// Aliases are the alternative keys of a check plugin.
	Aliases []string
	// Phase is the pipeline phase of a modifier plugin.
	Phase    Phase
	Meta     Meta
	Disabled bool
}

// NewRegistry creates a new plugin registry.
func NewRegistry(opts ...RegistryOption) *Registry {
	return &Registry{
		options:          newRegistryOptions(opts...),
		transformPlugins: make(map[string]TransformPlugin),
		modifierPlugins:  make(map[string]ModifierPlugin),
		checkSpecs:       make(map[string]CheckSpec),
		checkPlugins:     make(map[string]CheckPlugin),
		names:            make(map[string]bool),
		disabledPlugins:  make(map[string]bool),
	}
}

// Register registers plugins in this registry. The contract is determined by
// type assertion: [FieldPlugin], [TransformPlugin], [ModifierPlugin],
// [TypeDefaultPlugin] or [CheckPlugin]. Call it during application assembly,
// before generation starts.
//
// Register stops at the first plugin that implements no contract or several,
// has an empty key, or reuses a key or name owned by another plugin; plugins
// before it stay registered.
func (r *Registry) Register(plugins ...any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range plugins {
		if err := r.registerLocked(p); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) registerLocked(p any) error {
	if v := reflect.ValueOf(p); !v.IsValid() || (v.Kind() == reflect.Pointer && v.IsNil()) {
		return fmt.Errorf("%w: nil plugin", ErrUnsupportedPlugin)
	}
	// Every plugin contract embeds PluginMeta.
	meta, ok := p.(PluginMeta)
	if !ok {
		return fmt.Errorf("%w %T", ErrUnsupportedPlugin, p)
	}
	field, isField := p.(FieldPlugin)
	transform, isTransform := p.(TransformPlugin)
	modifier, isModifier := p.(ModifierPlugin)
	typeDefault, isTypeDefault := p.(TypeDefaultPlugin)
	check, isCheck := p.(CheckPlugin)
	switch contracts := countTrue(isField, isTransform, isModifier, isTypeDefault, isCheck); {
	case contracts == 0:
		return fmt.Errorf("%w %T", ErrUnsupportedPlugin, p)
	case contracts > 1:
		return fmt.Errorf("%w: %T", ErrAmbiguousPlugin, p)
	}
	name := normalizeKey(GetPluginName(meta))
	if r.names[name] {
		return fmt.Errorf("%w %q", ErrDuplicateName, GetPluginName(meta))
	}
	switch {
	case isField:
		r.fieldPlugins = append(r.fieldPlugins, field)
		sortByPriority(r.fieldPlugins)
	case isTypeDefault:
		r.typeDefaultPlugins = append(r.typeDefaultPlugins, typeDefault)
		sortByPriority(r.typeDefaultPlugins)
	case isTransform:
		key, err := r.claimValueKey(transform.Key(), p)
		if err != nil {
			return err
		}
		r.transformPlugins[key] = transform
	case isModifier:
		key, err := r.claimValueKey(modifier.Key(), p)
		if err != nil {
			return err
		}
		r.modifierPlugins[key] = modifier
	case isCheck:
		if err := r.registerCheckLocked(check); err != nil {
			return err
		}
	}
	r.names[name] = true
	return nil
}

func countTrue(values ...bool) int {
	n := 0
	for _, v := range values {
		if v {
			n++
		}
	}
	return n
}

func sortByPriority[T PluginMeta](plugins []T) {
	slices.SortFunc(plugins, func(a, b T) int {
		if c := cmp.Compare(b.Meta().Priority, a.Meta().Priority); c != 0 {
			return c // higher priority first
		}
		return cmp.Compare(GetPluginName(a), GetPluginName(b))
	})
}

// claimValueKey validates an optgen/optval key shared by transforms and modifiers.
func (r *Registry) claimValueKey(raw string, p any) (string, error) {
	key := normalizeKey(raw)
	if key == "" {
		return "", fmt.Errorf("%w: %T has an empty key", ErrUnsupportedPlugin, p)
	}
	if owner, ok := r.transformPlugins[key]; ok {
		return "", fmt.Errorf("%w %q: %T conflicts with %s", ErrDuplicateKey, key, p, GetPluginName(owner))
	}
	if owner, ok := r.modifierPlugins[key]; ok {
		return "", fmt.Errorf("%w %q: %T conflicts with %s", ErrDuplicateKey, key, p, GetPluginName(owner))
	}
	return key, nil
}

func (r *Registry) registerCheckLocked(h CheckPlugin) error {
	spec := h.Spec()
	spec.Key = normalizeKey(spec.Key)
	if spec.Key == "" {
		return fmt.Errorf("%w: %T has an empty key", ErrUnsupportedPlugin, h)
	}
	keys := []string{spec.Key}
	for _, a := range h.Aliases() {
		if a = normalizeKey(a); a != "" && !slices.Contains(keys, a) {
			keys = append(keys, a)
		}
	}
	for _, key := range keys {
		if owner, ok := r.checkPlugins[key]; ok {
			return fmt.Errorf("%w %q: %T conflicts with %s", ErrDuplicateKey, key, h, GetPluginName(owner))
		}
	}
	for _, key := range keys {
		keySpec := spec
		keySpec.Key = key
		r.checkSpecs[key] = keySpec
		r.checkPlugins[key] = h
	}
	return nil
}

// DisablePlugins disables plugins by name, case-insensitively. Disabled
// plugins are skipped by every lookup. It returns an error wrapping
// [ErrUnknownPlugin] for a name that is not registered and disables none.
func (r *Registry) DisablePlugins(names ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := make([]string, 0, len(names))
	for _, name := range names {
		key := normalizeKey(name)
		if key == "" {
			continue
		}
		if !r.names[key] {
			return fmt.Errorf("%w %q (known: %s)", ErrUnknownPlugin, strings.TrimSpace(name), strings.Join(r.namesLocked(), ", "))
		}
		keys = append(keys, key)
	}
	for _, key := range keys {
		r.disabledPlugins[key] = true
	}
	return nil
}

func (r *Registry) namesLocked() []string {
	descriptors := r.descriptorsLocked()
	out := make([]string, 0, len(descriptors))
	for _, d := range descriptors {
		out = append(out, d.Name)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// isPluginDisabledLocked checks if a plugin is disabled (caller must hold lock).
func (r *Registry) isPluginDisabledLocked(p PluginMeta) bool {
	return r.disabledPlugins[normalizeKey(GetPluginName(p))]
}

// warn delivers diagnostics after the registry lock is released, so a sink may
// call back into the registry.
func (r *Registry) warn(messages []string) {
	if r.options.warningSink == nil {
		return
	}
	for _, msg := range messages {
		r.options.warningSink(msg)
	}
}

// FindPlugin finds the best plugin for a field.
// Returns the first plugin that can handle the field (sorted by priority).
// Skips disabled plugins.
//
// Returns an error if no plugin can handle the field.
func (r *Registry) FindPlugin(field model.OptField) (FieldPlugin, error) {
	var warnings []string
	defer func() { r.warn(warnings) }()
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.fieldPlugins {
		if r.isPluginDisabledLocked(p) {
			continue
		}
		match := p.CanHandle(field)

		// If the plugin declares applicable types and this field type is not allowed,
		// ignore it. If it otherwise matches (CanHandle returns true), emit a warning.
		if allowed := p.Meta().ApplicableTypes; len(allowed) > 0 && !MatchesType(field.Type, allowed) {
			if match {
				warnings = append(warnings, fmt.Sprintf("field plugin %s ignored for field %s (type %s): applicable types: %s",
					GetPluginName(p), field.FieldName, field.Type, strings.Join(allowed, ", ")))
			}
			continue
		}
		if match {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no plugin found for field %s (type: %s)", field.FieldName, field.Type)
}

// FindTransformPlugin looks up a [TransformPlugin] by its normalized key and
// verifies that it is neither disabled nor type-restricted away from typeStr.
// The key may include a parameter suffix (e.g. "trimprefix=/"); the portion
// before '=' is used for lookup. Returns (nil, false) when no applicable
// transform is found.
func (r *Registry) FindTransformPlugin(key, typeStr string) (TransformPlugin, bool) {
	// Extract key from parameterized modifiers (e.g., "trimprefix=/" -> "trimprefix")
	k := normalizeKey(extractModifierKey(key))
	if k == "" {
		return nil, false
	}

	var warnings []string
	defer func() { r.warn(warnings) }()
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.transformPlugins[k]
	if !ok || r.isPluginDisabledLocked(p) {
		return nil, false
	}
	if allowed := p.Meta().ApplicableTypes; len(allowed) > 0 && typeStr != "" && !MatchesType(typeStr, allowed) {
		warnings = append(warnings, fmt.Sprintf("transform %s ignored for type %s: applicable types: %s", k, typeStr, strings.Join(allowed, ", ")))
		return nil, false
	}
	return p, true
}

// CollectModifiers gathers all [ModifierPlugin] instances that apply to field,
// sourced from optgen metadata (guards like "notnil") and optval modifiers
// (post-processors). optcheck keys address check plugins only.
// The result is stable-sorted by [Phase]; within the same phase, tag declaration
// order is preserved. Disabled plugins and plugins whose CanHandle returns false
// are excluded.
func (r *Registry) CollectModifiers(field model.OptField) []ModifierPlugin {
	var warnings []string
	defer func() { r.warn(warnings) }()
	r.mu.RLock()
	defer r.mu.RUnlock()

	capacity := len(field.Metadata) + len(field.Modifiers)
	modifiers := make([]ModifierPlugin, 0, capacity)
	seen := make(map[string]bool, capacity)

	tryAdd := func(key string) {
		k := normalizeKey(key)
		if k == "" || seen[k] {
			return
		}
		p := r.modifierPlugins[k]
		if p == nil || r.isPluginDisabledLocked(p) {
			return
		}
		if allowed := p.Meta().ApplicableTypes; len(allowed) > 0 && !modifierTypeAllowed(field, allowed) {
			warnings = append(warnings, fmt.Sprintf("modifier %s ignored for field %s (type %s): applicable types: %s",
				k, field.FieldName, field.Type, strings.Join(allowed, ", ")))
			return
		}
		if p.CanHandle(field) {
			seen[k] = true
			modifiers = append(modifiers, p)
		}
	}

	for _, key := range slices.Sorted(maps.Keys(field.Metadata)) {
		tryAdd(key)
	}
	for _, mod := range field.Modifiers {
		tryAdd(extractModifierKey(mod))
	}

	// Sort by phase (stable sort preserves tag order within each phase)
	slices.SortStableFunc(modifiers, func(a, b ModifierPlugin) int {
		return cmp.Compare(a.Phase(), b.Phase())
	})
	return modifiers
}

func modifierTypeAllowed(field model.OptField, allowed []string) bool {
	return MatchesType(field.Type, allowed) || (field.IsSlice && MatchesType(field.ElemType, allowed))
}

// FindCheckPlugin finds a check plugin by key (or alias). It returns an error
// wrapping [ErrUnknownCheck] when no plugin provides key and
// [ErrPluginDisabled] when the providing plugin is disabled.
func (r *Registry) FindCheckPlugin(key string) (CheckPlugin, CheckSpec, error) {
	k := normalizeKey(key)
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.checkPlugins[k]
	if !ok {
		return nil, CheckSpec{}, fmt.Errorf("%w %q", ErrUnknownCheck, key)
	}
	if r.isPluginDisabledLocked(h) {
		return nil, CheckSpec{}, fmt.Errorf("optcheck %q: %w: %s", key, ErrPluginDisabled, GetPluginName(h))
	}
	return h, r.checkSpecs[k], nil
}

// FindTypeDefaultPlugin finds a default value plugin for a type.
// Returns the first plugin that can handle the type (sorted by priority).
// Skips disabled plugins.
//
// Returns empty string if no plugin can handle the type.
func (r *Registry) FindTypeDefaultPlugin(typeStr string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.typeDefaultPlugins {
		if r.isPluginDisabledLocked(p) {
			continue
		}
		if p.CanProvideDefault(typeStr) {
			return p.GetDefault(typeStr)
		}
	}
	return ""
}

// Descriptors returns snapshots of all registered plugins ordered by kind
// (field, type default, transform, modifier, optcheck), then phase, priority
// in evaluation order (descending, ascending for checks), name and key.
func (r *Registry) Descriptors() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.descriptorsLocked()
}

func (r *Registry) descriptorsLocked() []Descriptor {
	var out []Descriptor
	add := func(kind Kind, key string, phase Phase, p PluginMeta) {
		out = append(out, Descriptor{Name: GetPluginName(p), Kind: kind, Key: key, Phase: phase, Meta: p.Meta(), Disabled: r.isPluginDisabledLocked(p)})
	}
	for _, p := range r.fieldPlugins {
		add(KindField, "", 0, p)
	}
	for _, p := range r.typeDefaultPlugins {
		add(KindTypeDefault, "", 0, p)
	}
	for key, p := range r.transformPlugins {
		add(KindTransform, key, PhaseTransform, p)
	}
	for key, p := range r.modifierPlugins {
		add(KindModifier, key, p.Phase(), p)
	}
	for key, p := range r.checkPlugins {
		if normalizeKey(p.Spec().Key) != key {
			continue // aliases are reported with their primary key
		}
		add(KindOptCheck, key, PhaseCheck, p)
		for _, a := range p.Aliases() {
			if a = normalizeKey(a); a != "" && a != key {
				out[len(out)-1].Aliases = append(out[len(out)-1].Aliases, a)
			}
		}
	}
	order := []Kind{KindField, KindTypeDefault, KindTransform, KindModifier, KindOptCheck}
	slices.SortFunc(out, func(a, b Descriptor) int {
		priority := cmp.Compare(b.Meta.Priority, a.Meta.Priority)
		if a.Kind == KindOptCheck {
			priority = -priority // checks run in ascending priority
		}
		return cmp.Or(
			cmp.Compare(slices.Index(order, a.Kind), slices.Index(order, b.Kind)),
			cmp.Compare(a.Phase, b.Phase),
			priority,
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.Key, b.Key),
		)
	})
	return out
}

// MatchesType reports whether typeStr matches one of patterns. Besides exact
// type strings, "[]" matches any slice type, "*" any pointer type and
// "interface" the types interface{} and any.
func MatchesType(typeStr string, patterns []string) bool {
	for _, t := range patterns {
		switch t = strings.TrimSpace(t); {
		case t == "[]" && strings.HasPrefix(typeStr, "[]"),
			t == "*" && strings.HasPrefix(typeStr, "*"),
			t == "interface" && (typeStr == "interface{}" || typeStr == "any"),
			t == typeStr:
			return true
		}
	}
	return false
}
