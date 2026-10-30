// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package optgen

import "github.com/altessa-s/go-optgen/plugin"

//go:generate optgen generate

type options struct {
	// plugins register additional plugins into the registry of every command
	// execution, after the built-in plugins and before --disable-plugins.
	plugins []func(*plugin.Registry) error `optval:"nodup"`
}
