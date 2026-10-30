// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package cache_test

import (
	"fmt"
	"strconv"

	"github.com/altessa-s/go-optgen/examples/cache"
)

func Example() {
	c := cache.New(
		// Options whose argument does not mention K or V need explicit
		// type arguments; the others infer them.
		cache.WithCapacity[string, int](2),
		cache.WithOnEvict(func(k string, v int) { fmt.Println("evicted", k, v) }),
	)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)
	v, ok, _ := c.Get("a")
	fmt.Println(v, ok, c.Cap())
	// Output:
	// evicted a 1
	// 0 false 2
}

// isSet distinguishes an explicit option from the default value.
func Example_isSet() {
	load := func(k string) (int, error) { return strconv.Atoi(k) }

	loading := cache.New(cache.WithLoader(load))
	explicit := cache.New(cache.WithLoader(load), cache.WithCapacity[string, int](cache.DefaultCapacity))
	fmt.Println(loading.Cap(), explicit.Cap())

	v, ok, err := loading.Get("42")
	fmt.Println(v, ok, err)
	_, _, err = loading.Get("x")
	fmt.Println(err)
	// Output:
	// 1024 128
	// 42 true <nil>
	// strconv.Atoi: parsing "x": invalid syntax
}
