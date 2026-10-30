// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package defaults

// LoggerDefaultPlugin provides default values for *slog.Logger types.
// It generates: slog.New(slog.DiscardHandler)
type LoggerDefaultPlugin struct {
	typeDefaultBase
}

// CanProvideDefault reports whether the plugin has a default for typeStr.
func (p *LoggerDefaultPlugin) CanProvideDefault(typeStr string) bool {
	return typeStr == "*slog.Logger"
}

// GetDefault returns the default expression for typeStr.
func (p *LoggerDefaultPlugin) GetDefault(_ string) string {
	return "slog.New(slog.DiscardHandler)"
}
