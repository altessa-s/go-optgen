// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package fields

import (
	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const manualPriority = 110

// ManualPlugin skips generating a With* function for a field while still allowing
// the field to participate in defaultOptions/newOptions generation.
//
// Use by setting optgen metadata:
//
//	opt:"FieldName" optgen:"manual"
//
// This is useful when a field needs a custom With* implementation (validation,
// normalization, cross-field behavior) but you still want defaults/newOptions
// generated consistently.
type ManualPlugin struct{}

// Meta returns plugin metadata.
func (p *ManualPlugin) Meta() plugin.Meta {
	return plugin.Meta{
		Kind:     plugin.KindField,
		Priority: manualPriority, // explicit user request should override all other plugins
	}
}

// CanHandle reports whether the plugin applies to field.
func (p *ManualPlugin) CanHandle(field model.OptField) bool {
	// Handle explicit "manual" metadata or explicit default-only model fields
	return plugin.IsTruthyMetadata(field.Metadata, "manual") || field.DefaultOnly
}

// Generate renders the option function for field.
func (p *ManualPlugin) Generate(_ plugin.GenerationContext, _ model.OptField, _ plugin.Pipeline) (plugin.GeneratedCode, error) {
	// Intentionally generate no With* function.
	return plugin.GeneratedCode{Code: ""}, nil
}
