// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package check

import "github.com/altessa-s/go-optgen/plugin"

// Register installs fresh built-in plugins in r.
func Register(r *plugin.Registry) error {
	return r.Register(&MaxLenCheck{
		CheckBase: plugin.NewCheckBase("maxlen", maxlenPriority, plugin.WithRequiresValue(true)),
	},
		&MinLenCheck{
			CheckBase: plugin.NewCheckBase("minlen", minlenPriority, plugin.WithRequiresValue(true)),
		},
		&NonZeroCheck{
			CheckBase: plugin.NewCheckBase("nonzero", nonzeroPriority),
		},
		&OneOfCheck{
			CheckBase: plugin.NewCheckBase("oneof", oneofPriority, plugin.WithRequiresValue(true)),
		},
		&RequiredCheck{
			CheckBase: plugin.NewCheckBase("required", requiredPriority, plugin.WithAliases("nonempty")),
		})
}
