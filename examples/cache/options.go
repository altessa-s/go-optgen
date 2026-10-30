// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package cache

//go:generate optgen generate --with-isset --source-file $GOFILE

// DefaultCapacity is the number of entries a cache holds by default.
const DefaultCapacity = 128

// options is generic: every generated option carries the type parameters,
// so WithOnEvict[string, int] only configures a Cache[string, int].
type options[K comparable, V any] struct {
	capacity int `optgen:"default=DefaultCapacity" optval:"positive"`
	onEvict  func(K, V)
	loader   func(K) (V, error)

	// set records which options were applied (--with-isset).
	set optionsFieldSet
}
