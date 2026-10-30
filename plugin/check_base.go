// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package plugin

// CheckBase provides common functionality for CheckPlugin implementations.
// Embed this type in your plugin struct to get default Meta(), Spec() and
// Aliases() implementations.
//
// Example:
//
//	type RequiredCheck struct {
//	    plugin.CheckBase
//	}
//
//	func NewRequiredCheck() *RequiredCheck {
//	    return &RequiredCheck{
//	        CheckBase: plugin.NewCheckBase("required", 10, plugin.WithAliases("present")),
//	    }
//	}
//
//	func (c *RequiredCheck) Validate(field model.OptField, rawValue string) error {
//	    // applicability checks
//	}
//
//	func (c *RequiredCheck) Generate(ctx plugin.GenerationContext, field model.OptField, valueVar, rawValue string) []string {
//	    // validation logic
//	}
type CheckBase struct {
	key      string
	priority int
	options  *checkOptions
}

// NewCheckBase creates a check descriptor using generated options.
func NewCheckBase(key string, priority int, opts ...CheckOption) CheckBase {
	return CheckBase{key: key, priority: priority, options: newCheckOptions(opts...)}
}

// Meta returns the plugin metadata.
func (b *CheckBase) Meta() Meta {
	return Meta{Kind: KindOptCheck, Priority: b.priority}
}

// Spec returns the check specification.
func (b *CheckBase) Spec() CheckSpec {
	if b.options == nil {
		return CheckSpec{Key: b.key, Priority: b.priority}
	}
	return CheckSpec{Key: b.key, Priority: b.priority, RequiresValue: b.options.requiresValue}
}

// Aliases returns alternative keys for this check.
func (b *CheckBase) Aliases() []string {
	if b.options == nil {
		return nil
	}
	return b.options.aliases
}
