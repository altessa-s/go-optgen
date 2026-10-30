// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package defaults

import (
	"strings"
)

// MapDefaultPlugin provides default values for map types.
// It generates: make(map[K]V)
type MapDefaultPlugin struct {
	typeDefaultBase
}

// CanProvideDefault reports whether the plugin has a default for typeStr.
func (p *MapDefaultPlugin) CanProvideDefault(typeStr string) bool {
	return strings.HasPrefix(typeStr, "map[")
}

// GetDefault returns the default expression for typeStr.
func (p *MapDefaultPlugin) GetDefault(typeStr string) string {
	return "make(" + typeStr + ")"
}
