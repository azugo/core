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
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
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

	var value interface{}
	var found bool
	finish := c.instrumenter.Observe(ctx, InstrumentationCacheGet, key)
	if value, found = c.cache.Get(key); found {
		finish(nil)
		return value.(T), nil
	}
	if c.loader != nil {
		var err error
		if value, err = c.getWithLoader(ctx, key, c.getLoader(ctx, opts...)); err != nil {
			return val, err
		}
		finish(nil)
		return value.(T), nil
	}
	finish(nil)
	return val, nil
}

func (c *memoryCache[T]) set(key string, v interface{}, ttl time.Duration) error {
	if c.cache == nil {
		return ErrCacheClosed
	}
	success := c.cache.SetWithTTL(key, v, 1, ttl)
	if !success {
		return ErrCacheClosed
	}
	// Even if a Set gets applied, it might take a few milliseconds after the call has returned to the user.
	// In database terms, it is an eventual consistency model.
	time.Sleep(10 * time.Millisecond)
	return nil
}

func (c *memoryCache[T]) getWithLoader(ctx context.Context, key string, loader func(string) (interface{}, error)) (interface{}, error) {
	v, err := loader(key)
	if err != nil {
		return nil, ErrKeyNotFound{Key: key}
	}
	err = c.set(key, v, c.ttl)
	return v, err
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
	opt := newItemOptions(opts...)
	ttl := opt.TTL
	if ttl == 0 {
		ttl = c.ttl
	}

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheSet, key)
	defer finish(nil)

	return c.set(key, value, ttl)
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