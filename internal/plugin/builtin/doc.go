// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

// Package builtin registers every built-in plugin: checks, type defaults,
// field plugins, guards, post-processors, and transforms.
//
// # Usage
//
//	r := plugin.NewRegistry()
//	if err := builtin.Register(r); err != nil {
//		return err
//	}
package builtin
