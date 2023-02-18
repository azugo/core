// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"azugo.io/core/instrumenter"

	"github.com/dgraph-io/ristretto"
)

type memoryCache[T any] struct {
	cache        *ristretto.Cache
	ttl          time.Duration
	lock         sync.Mutex
	loader       func(ctx context.Context, key string) (interface{}, error)
	instrumenter instrumenter.Instrumenter
}


func newMemoryCache[T any](opts ...CacheOption) (CacheInstance[T], error) {
	opt := newCacheOptions(opts...)
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000, // number of keys to track frequency of (10k).
		MaxCost:     1 << 20, // maximum cost of cache (1GB).
		BufferItems: 64, // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	loader := opt.Loader
	if loader != nil {
		loader = func(ctx context.Context, key string) (interface{}, error) {
			finish := opt.Instrumenter.Observe(ctx, InstrumentationCacheLoader, key)
			v, err := opt.Loader(ctx, key)
			finish(err)
			return v, err
		}
	}

	return &memoryCache[T]{
		cache:        c,
		ttl:          opt.TTL,
		loader:       loader,
		instrumenter: opt.Instrumenter,
	}, nil
}

func (c *memoryCache[T]) getLoader(ctx context.Context, opts ...ItemOption[T]) func(string) (interface{}, error) {
	return func(key string) (interface{}, error) {
		v, err := c.loader(ctx, key)
		if err != nil {
			return nil, err
		}
		vv, ok := v.(T)
		if !ok {
			return nil, fmt.Errorf("invalid value from loader: %v", v)
		}
		return vv, nil
	}
}

func (c *memoryCache[T]) Get(ctx context.Context, key string, opts ...ItemOption[T]) (T, error) {
	var val T
	if c.cache == nil {
		return val, ErrCacheClosed
	}

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheGet, key)

	var value interface{}
	var found bool
	if c.loader != nil {
		value, found = c.getWithLoader(key, c.getLoader(ctx, opts...))
	} else {
		value, found = c.cache.Get(key)
	}
	if !found {
		finish(ErrKeyNotFound{Key: key})
		return val, ErrKeyNotFound{Key: key}
	}
	finish(nil)
	return value.(T), nil
}

func (c *memoryCache[T]) getWithLoader(key string, loader func(string) (interface{}, error)) (interface{}, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	i, exists := c.cache.Get(key)
	if exists {
		return i, true
	}
	if i == nil {
		v, err := loader(key)
		if err != nil {
			return nil, false
		}
		_ = c.cache.SetWithTTL(key, v, 1, c.ttl)
		return v, true
	}
	return i, true
}

func (c *memoryCache[T]) Pop(ctx context.Context, key string) (T, error) {
	var val T
	if c.cache == nil {
		return val, ErrCacheClosed
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheGet, key)

	i, exists := c.cache.Get(key)
	if !exists {
		finish(nil)
		return val, ErrKeyNotFound{Key: key}
	}
	c.cache.Del(key)
	finish(nil)
	return i.(T), nil
}

func (c *memoryCache[T]) Set(ctx context.Context, key string, value T, opts ...ItemOption[T]) error {
	if c.cache == nil {
		return ErrCacheClosed
	}

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheSet, key)
	defer finish(nil)

	_ = c.cache.SetWithTTL(key, value, 1, c.ttl)
	return nil
}

func (c *memoryCache[T]) Delete(ctx context.Context, key string) error {
	if c.cache == nil {
		return ErrCacheClosed
	}

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheDelete, key)
	defer finish(nil)

	c.cache.Del(key)
	return nil
}

func (c *memoryCache[T]) Close() {
	if c.cache == nil {
		return
	}
	c.cache.Clear()
	c.cache = nil
}