// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package optgen

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/altessa-s/go-optgen/internal/application"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/defaults"
	"github.com/altessa-s/go-optgen/plugin"
)

// newSessionRegistry assembles the plugin registry of one command execution:
// the built-in plugins, the configured type defaults, the plugins of
// [WithPlugins], and finally the disabled plugin names. Diagnostics are
// delivered to warn.
func newSessionRegistry(warn func(string), o *options, typeDefaults map[string]string, disabled []string) (*plugin.Registry, error) {
	r := plugin.NewRegistry(plugin.WithWarningSink(warn))
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

// warningCounter forwards warnings to sink and counts them for --strict.
type warningCounter struct {
	mu    sync.Mutex
	count int
	sink  func(string)
}

func (w *warningCounter) warn(msg string) {
	w.mu.Lock()
	w.count++
	w.mu.Unlock()
	w.sink(msg)
}

func (w *warningCounter) reported() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.count
}

// errStrictWarnings reports warnings in --strict mode.
var errStrictWarnings = errors.New("warnings reported in strict mode")

// strictOutput refuses to publish once a warning has been reported.
type strictOutput struct {
	next     application.Output
	warnings *warningCounter
}

func (s strictOutput) Write(ctx context.Context, path string, content []byte, formatter string) error {
	if n := s.warnings.reported(); n > 0 {
		return fmt.Errorf("%w: %d warning(s); %s not written", errStrictWarnings, n, path)
	}
	return s.next.Write(ctx, path, content, formatter)
}
