// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"context"
	"fmt"
	"sync"

	"azugo.io/core/instrumenter"

	"github.com/lafriks/ttlcache/v3"
)

type memoryCache[T any] struct {
	cache        *ttlcache.Cache[string, T]
	lock         sync.Mutex
	loader       func(ctx context.Context, key string) (interface{}, error)
	instrumenter instrumenter.Instrumenter
}

func newMemoryCache[T any](opts ...CacheOption) (CacheInstance[T], error) {
	opt := newCacheOptions(opts...)
	c := ttlcache.New(ttlcache.WithTTL[string, T](opt.TTL))

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
		loader:       loader,
		instrumenter: opt.Instrumenter,
	}, nil
}

func (c *memoryCache[T]) getLoader(ctx context.Context, opts ...ItemOption[T]) ttlcache.LoaderFunc[string, T] {
	return func(cache *ttlcache.Cache[string, T], key string) (*ttlcache.Item[string, T], error) {
		opt := newItemOptions(opts...)
		ttl := opt.TTL
		if ttl == 0 {
			ttl = ttlcache.DefaultTTL
		}

		v, err := c.loader(ctx, key)
		if err != nil {
			return nil, err
		}
		vv, ok := v.(T)
		if !ok {
			return nil, fmt.Errorf("invalid value from loader: %v", v)
		}
		return cache.Set(key, vv, ttl), nil
	}
}

func (c *memoryCache[T]) Get(ctx context.Context, key string, opts ...ItemOption[T]) (T, error) {
	var val T
	if c.cache == nil {
		return val, ErrCacheClosed
	}

	cacheOpts := make([]ttlcache.Option[string, T], 0)

	if c.loader != nil {
		cacheOpts = append(cacheOpts, ttlcache.WithLoader[string, T](c.getLoader(ctx, opts...)))
	}

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheGet, key)

	i, err := c.cache.Get(key, cacheOpts...)
	if err != nil || i == nil {
		finish(err)
		return val, err
	}
	if i.IsExpired() {
		finish(nil)
		return val, nil
	}
	finish(nil)
	return i.Value(), nil
}

func (c *memoryCache[T]) Pop(ctx context.Context, key string) (T, error) {
	var val T
	if c.cache == nil {
		return val, ErrCacheClosed
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheGet, key)

	i, err := c.cache.Get(key)
	if err != nil {
		finish(err)
		return val, err
	}
	if i == nil || i.IsExpired() {
		finish(nil)
		return val, ErrKeyNotFound{Key: key}
	}
	c.cache.Delete(key)
	finish(nil)
	return i.Value(), nil
}

func (c *memoryCache[T]) Set(ctx context.Context, key string, value T, opts ...ItemOption[T]) error {
	if c.cache == nil {
		return ErrCacheClosed
	}
	opt := newItemOptions(opts...)
	ttl := opt.TTL
	if ttl == 0 {
		ttl = ttlcache.DefaultTTL
	}

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheSet, key)
	defer finish(nil)

	_ = c.cache.Set(key, value, ttl)
	return nil
}

func (c *memoryCache[T]) Delete(ctx context.Context, key string) error {
	if c.cache == nil {
		return ErrCacheClosed
	}

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheDelete, key)
	defer finish(nil)

	c.cache.Delete(key)
	return nil
}

func (c *memoryCache[T]) Close() {
	if c.cache == nil {
		return
	}
	c.cache.DeleteAll()
	c.cache = nil
}
