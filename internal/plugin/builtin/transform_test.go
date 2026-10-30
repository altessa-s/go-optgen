// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package builtin_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/plugin"
)

func TestHasModifier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		modifiers []string
		modifier  string
		expected  bool
	}{
		{
			name:      "empty modifiers",
			modifiers: []string{},
			modifier:  "dedup",
			expected:  false,
		},
		{
			name:      "has modifier",
			modifiers: []string{"dedup", "lower"},
			modifier:  "dedup",
			expected:  true,
		},
		{
			name:      "does not have modifier",
			modifiers: []string{"lower", "upper"},
			modifier:  "dedup",
			expected:  false,
		},
		{
			name:      "single modifier match",
			modifiers: []string{"trimspaces"},
			modifier:  "trimspaces",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := plugin.HasModifier(tt.modifiers, tt.modifier)
			require.Equal(t, tt.expected, result)
		})
	}
}
