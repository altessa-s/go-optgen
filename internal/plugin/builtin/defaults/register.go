// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package defaults

import "github.com/altessa-s/go-optgen/plugin"

// Register installs fresh built-in plugins in r.
func Register(r *plugin.Registry) error {
	return r.Register(
		&LoggerDefaultPlugin{},
		&MapDefaultPlugin{},
	)
}
