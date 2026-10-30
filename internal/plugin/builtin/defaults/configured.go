// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package defaults

import (
	"maps"

	"github.com/altessa-s/go-optgen/plugin"
)

// configuredPriority ranks configured defaults above the built-in ones.
const configuredPriority = 100

// ConfiguredDefaultPlugin supplies the default expressions configured per
// field type (the type-defaults section of .optgen.yaml). Packages named in
// an expression must be imported by the source file of the options struct.
type ConfiguredDefaultPlugin struct {
	defaults map[string]string
}

// NewConfigured returns a plugin serving a copy of defaults, keyed by the
// field type as written in the source (for example "*slog.Logger").
func NewConfigured(defaults map[string]string) *ConfiguredDefaultPlugin {
	return &ConfiguredDefaultPlugin{defaults: maps.Clone(defaults)}
}

// Meta returns plugin metadata.
func (p *ConfiguredDefaultPlugin) Meta() plugin.Meta {
	return plugin.Meta{Kind: plugin.KindTypeDefault, Priority: configuredPriority}
}

// CanProvideDefault reports whether a default is configured for typeStr.
func (p *ConfiguredDefaultPlugin) CanProvideDefault(typeStr string) bool {
	_, ok := p.defaults[typeStr]
	return ok
}

// GetDefault returns the configured default expression for typeStr.
func (p *ConfiguredDefaultPlugin) GetDefault(typeStr string) string {
	return p.defaults[typeStr]
}
