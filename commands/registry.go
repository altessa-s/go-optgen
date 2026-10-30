// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package commands

import (
	"fmt"
	"io"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/defaults"
	"github.com/altessa-s/go-optgen/plugin"
)

// newSessionRegistry assembles the plugin registry of one command execution:
// the built-in plugins, the configured type defaults, the plugins of
// [WithPlugins], and finally the disabled plugin names. Diagnostics are
// written to stderr.
func newSessionRegistry(stderr io.Writer, o *options, typeDefaults map[string]string, disabled []string) (*plugin.Registry, error) {
	r := plugin.NewRegistry(plugin.WithWarningSink(warningSink(stderr)))
	if err := builtin.Register(r); err != nil {
		return nil, fmt.Errorf("failed to register built-in plugins: %w", err)
	}
	if len(typeDefaults) > 0 {
		if err := r.Register(defaults.NewConfigured(typeDefaults)); err != nil {
			return nil, fmt.Errorf("failed to register configured type defaults: %w", err)
		}
	}
	for _, register := range o.plugins {
		if err := register(r); err != nil {
			return nil, fmt.Errorf("failed to register plugins: %w", err)
		}
	}
	if err := r.DisablePlugins(disabled...); err != nil {
		return nil, err
	}
	return r, nil
}

// warningSink prints diagnostics to stderr.
func warningSink(stderr io.Writer) func(string) {
	return func(msg string) {
		_, _ = fmt.Fprintf(stderr, "optgen: WARN: %s\n", msg)
	}
}
