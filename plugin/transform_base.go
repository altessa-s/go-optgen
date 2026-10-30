// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package plugin

const (
	// DefaultPriority is the default priority for transformation plugins.
	DefaultPriority = 10
)

// TransformBase provides common functionality for TransformPlugin implementations.
// Embed this type in your plugin struct to get default Meta() and Key() implementations.
//
// Example:
//
//	type LowerTransformPlugin struct {
//	    plugin.TransformBase
//	}
//
//	func NewLowerTransformPlugin() *LowerTransformPlugin {
//	    return &LowerTransformPlugin{
//	        TransformBase: plugin.NewTransformBase("lower", plugin.WithApplicableTypes("string")),
//	    }
//	}
//
//	func (t *LowerTransformPlugin) Apply(ctx plugin.GenerationContext, _ model.OptField, _ string, expr string) (string, bool) {
//	    ctx.AddImport("strings")
//	    return "strings.ToLower(" + expr + ")", true
//	}
type TransformBase struct {
	key     string
	options *transformOptions
}

// NewTransformBase creates a transform descriptor using generated options.
func NewTransformBase(key string, opts ...TransformOption) TransformBase {
	return TransformBase{key: key, options: newTransformOptions(opts...)}
}

// Meta returns the plugin metadata.
func (b *TransformBase) Meta() Meta {
	if b.options == nil {
		return Meta{Kind: KindTransform, Priority: DefaultPriority}
	}
	return Meta{Kind: KindTransform, ApplicableTypes: b.options.applicableTypes,
		Priority: b.options.priority, DefaultForTypes: b.options.defaultForTypes, DisabledBy: b.options.disabledBy}
}

// Key returns the modifier token used in tags.
func (b *TransformBase) Key() string { return b.key }
