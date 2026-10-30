// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package builtin

import (
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/check"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/defaults"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/fields"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/guard"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/postprocess"
	"github.com/altessa-s/go-optgen/internal/plugin/builtin/transform"
	"github.com/altessa-s/go-optgen/plugin"
)

// Register installs fresh instances of every built-in plugin in r.
func Register(r *plugin.Registry) error {
	for _, register := range []func(*plugin.Registry) error{
		check.Register,
		defaults.Register,
		fields.Register,
		guard.Register,
		postprocess.Register,
		transform.Register,
	} {
		if err := register(r); err != nil {
			return err
		}
	}
	return nil
}
