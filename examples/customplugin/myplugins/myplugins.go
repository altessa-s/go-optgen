// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package myplugins

import (
	"fmt"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

// Register adds the plugins of this package to registry. Pass it to
// optgen.WithPlugins.
func Register(registry *plugin.Registry) error {
	if err := registry.Register(NewCollapse()); err != nil {
		return err
	}
	return registry.Register(NewHostPort())
}

// Collapse is the optval:"collapse" transform: it replaces every run of
// white space with a single space and trims both ends.
type Collapse struct {
	plugin.TransformBase
}

// NewCollapse returns the collapse transform for string fields.
func NewCollapse() *Collapse {
	return &Collapse{TransformBase: plugin.NewTransformBase("collapse", plugin.WithApplicableTypes("string"))}
}

// Apply wraps expr in the generated transformation.
func (*Collapse) Apply(ctx plugin.GenerationContext, _ model.OptField, _, expr string) (string, bool) {
	ctx.AddImport("strings")
	return `strings.Join(strings.Fields(` + expr + `), " ")`, true
}

// HostPort is the optcheck:"hostport" check: the value must split into a
// host and a port with net.SplitHostPort.
type HostPort struct {
	plugin.CheckBase
}

// hostPortPriority orders hostport after the built-in checks.
const hostPortPriority = 200

// NewHostPort returns the hostport check.
func NewHostPort() *HostPort {
	return &HostPort{CheckBase: plugin.NewCheckBase("hostport", hostPortPriority)}
}

// Validate accepts string fields only, so a misplaced tag fails generation
// instead of producing code that does not compile.
func (*HostPort) Validate(field model.OptField, _ string) error {
	if plugin.ValueKindOf(field) != plugin.ValueString {
		return fmt.Errorf("%w %s (supported: string)", plugin.ErrNotApplicable, field.Type)
	}
	return nil
}

// Generate emits the check of valueVar. The failure honors the generation
// settings like the built-in checks: an error result with --option-error, a
// panic by default, or keeping the previous value with --check-failure skip.
func (*HostPort) Generate(ctx plugin.GenerationContext, field model.OptField, valueVar, _ string) []string {
	ctx.AddImport("net")
	var fail string
	switch {
	case ctx.OptionReturnsError:
		ctx.AddImport("fmt")
		fail = fmt.Sprintf("return fmt.Errorf(%q, err)", field.FieldName+": %w")
	case ctx.CheckFailure == model.CheckFailureSkip:
		fail = "return"
	default:
		ctx.AddImport("fmt")
		fail = fmt.Sprintf("panic(fmt.Errorf(%q, err))", field.FieldName+": %w")
	}
	return []string{
		"if _, _, err := net.SplitHostPort(" + valueVar + "); err != nil {",
		"  " + fail,
		"}",
	}
}
