// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package cache shows options for a generic struct and --with-isset.
//
// # Generation
//
//	//go:generate optgen generate --with-isset --source-file $GOFILE
//
// Every generated option and constructor carries the type parameters of
// options[K, V]. An option whose argument does not mention them, such as
// WithCapacity, needs explicit type arguments:
//
//	c := cache.New(cache.WithCapacity[string, int](16))
//
// The isSet method tells an explicit option from a default, which New uses
// to pick a larger default capacity for a loading cache.
package cache
