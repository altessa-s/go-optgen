// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package model_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/model"
)

func TestRenderSettingsVerify(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		settings model.RenderSettings
		valid    bool
	}{
		{name: "zero value", valid: true},
		{name: "documented modes", settings: model.RenderSettings{StringSetter: model.StringSetterPlain, CheckFailure: model.CheckFailureSkip,
			SliceSetter: model.SliceSetterSlice, DefaultsFrom: model.DefaultsFromFunc, DefaultsName: "getDefaults"}, valid: true},
		{name: "var without name", settings: model.RenderSettings{DefaultsFrom: model.DefaultsFromVar}, valid: true},
		{name: "string setter", settings: model.RenderSettings{StringSetter: "fancy"}},
		{name: "check failure", settings: model.RenderSettings{CheckFailure: "ignore"}},
		{name: "slice setter", settings: model.RenderSettings{SliceSetter: "array"}},
		{name: "defaults source", settings: model.RenderSettings{DefaultsFrom: "file"}},
		{name: "tag with name", settings: model.RenderSettings{DefaultsName: "x"}},
		{name: "none with name", settings: model.RenderSettings{DefaultsFrom: model.DefaultsFromNone, DefaultsName: "x"}},
		{name: "invalid identifier", settings: model.RenderSettings{DefaultsFrom: model.DefaultsFromVar, DefaultsName: "1x"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.settings.Verify()
			if tc.valid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, model.ErrInvalidSettings)
		})
	}
}
