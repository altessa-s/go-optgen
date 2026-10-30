// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package cache

import "slices"

// LoaderCapacity is the default capacity of a cache with a loader, used
// unless WithCapacity sets one explicitly.
const LoaderCapacity = 1024

// Cache is a FIFO-evicting map. It is not safe for concurrent use.
type Cache[K comparable, V any] struct {
	opts    *options[K, V]
	entries map[K]V
	order   []K
}

// New returns an empty Cache.
func New[K comparable, V any](opts ...Option[K, V]) *Cache[K, V] {
	o := newOptions(opts...)
	// isSet tells an explicit WithCapacity(DefaultCapacity) apart from the
	// default, so only the default is raised for a loading cache.
	if o.loader != nil && !o.isSet(optionsFieldCapacity) {
		o.capacity = LoaderCapacity
	}
	return &Cache[K, V]{opts: o, entries: make(map[K]V)}
}

// Cap returns the maximum number of entries.
func (c *Cache[K, V]) Cap() int { return c.opts.capacity }

// Put stores v under k, evicting the oldest entry when the cache is full.
func (c *Cache[K, V]) Put(k K, v V) {
	if _, ok := c.entries[k]; !ok {
		if len(c.order) == c.opts.capacity {
			c.evict()
		}
		c.order = append(c.order, k)
	}
	c.entries[k] = v
}

// Get returns the value under k, calling the loader on a miss when one is
// configured.
func (c *Cache[K, V]) Get(k K) (V, bool, error) {
	if v, ok := c.entries[k]; ok {
		return v, true, nil
	}
	var zero V
	if c.opts.loader == nil {
		return zero, false, nil
	}
	v, err := c.opts.loader(k)
	if err != nil {
		return zero, false, err
	}
	c.Put(k, v)
	return v, true, nil
}

func (c *Cache[K, V]) evict() {
	oldest := c.order[0]
	c.order = slices.Delete(c.order, 0, 1)
	v := c.entries[oldest]
	delete(c.entries, oldest)
	if c.opts.onEvict != nil {
		c.opts.onEvict(oldest, v)
	}
}
