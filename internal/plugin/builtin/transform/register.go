// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package transform

import "github.com/altessa-s/go-optgen/plugin"

// Register installs fresh built-in plugins in r.
func Register(r *plugin.Registry) error {
	return r.Register(
		&LowerTransformPlugin{
			TransformBase: plugin.NewTransformBase("lower", plugin.WithApplicableTypes("string")),
		},
		&TrimPrefixTransformPlugin{
			TransformBase: plugin.NewTransformBase("trimprefix", plugin.WithApplicableTypes("string")),
		},
		&TrimSpacesTransformPlugin{
			TransformBase: plugin.NewTransformBase("trimspaces",
				plugin.WithApplicableTypes("string"),
				plugin.WithDefaultForTypes("string"),
				plugin.WithDisabledBy("notrim"),
			),
		},
		&TrimSuffixTransformPlugin{
			TransformBase: plugin.NewTransformBase("trimsuffix", plugin.WithApplicableTypes("string")),
		},
		&UpperTransformPlugin{
			TransformBase: plugin.NewTransformBase("upper", plugin.WithApplicableTypes("string")),
		},
	)
}
