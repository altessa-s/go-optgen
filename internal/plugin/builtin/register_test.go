// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package builtin_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/plugin/builtin"
	"github.com/altessa-s/go-optgen/plugin"
)

func TestRegisterInstallsEveryKindOnce(t *testing.T) {
	t.Parallel()
	r := plugin.NewRegistry()
	require.NoError(t, builtin.Register(r))
	kinds := make(map[plugin.Kind]int)
	for _, d := range r.Descriptors() {
		kinds[d.Kind]++
	}
	for _, kind := range []plugin.Kind{plugin.KindField, plugin.KindTypeDefault, plugin.KindTransform, plugin.KindModifier, plugin.KindOptCheck} {
		require.Positive(t, kinds[kind], kind)
	}
	require.ErrorIs(t, builtin.Register(r), plugin.ErrDuplicateName, "a registry accepts one copy of each plugin")
}
