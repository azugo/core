// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"azugo.io/core/instrumenter"

	"github.com/dgraph-io/ristretto"
	"github.com/goccy/go-json"
)

type memoryCache[T any] struct {
	cache        *ristretto.Cache
	ttl          time.Duration
	serialize    bool
	lock         sync.Mutex
	loader       func(ctx context.Context, key string) (any, error)
	instrumenter instrumenter.Instrumenter
}

func newMemoryCache[T any](opts ...Option) (Instance[T], error) {
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,    // number of keys to track frequency of (10k).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	opt := newCacheOptions(opts...)

	loader := opt.Loader
	if loader != nil {
		loader = func(ctx context.Context, key string) (any, error) {
			finish := opt.Instrumenter.Observe(ctx, InstrumentationLoader, key)
			v, err := opt.Loader(ctx, key)
			finish(err)

			return v, err
		}
	}

	return &memoryCache[T]{
		cache:        c,
		ttl:          opt.TTL,
		serialize:    opt.Serialize,
		loader:       loader,
		instrumenter: opt.Instrumenter,
	}, nil
}

func (c *memoryCache[T]) getLoader(ctx context.Context) func(string) (any, error) {
	return func(key string) (any, error) {
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

func (c *memoryCache[T]) unmarshal(value any) (T, error) {
	if !c.serialize {
		val, _ := value.(T)

		return val, nil
	}

	val := new(T)

	b, ok := value.([]byte)
	if !ok {
		return *val, errors.New("invalid cache value type")
	}

	if err := json.Unmarshal(b, val); err != nil {
		return *val, fmt.Errorf("invalid cache value: %w", err)
	}

	return *val, nil
}

func (c *memoryCache[T]) Get(ctx context.Context, key string, _ ...ItemOption[T]) (T, error) {
	finish := c.instrumenter.Observe(ctx, InstrumentationGet, key)

	var val T

	if c.cache == nil {
		finish(ErrCacheClosed)

		return val, ErrCacheClosed
	}

	var (
		value any
		found bool
	)

	if value, found = c.cache.Get(key); found {
		v, err := c.unmarshal(value)
		finish(err)

		return v, err
	}

	if c.loader != nil {
		var err error
		if value, err = c.getWithLoader(key, c.getLoader(ctx)); err != nil {
			finish(err)

			return val, err
		}

		v, err := c.unmarshal(value)
		finish(err)

		return v, err
	}

	finish(nil)

	return val, nil
}

func (c *memoryCache[T]) set(key string, v T, ttl time.Duration) error {
	if c.cache == nil {
		return ErrCacheClosed
	}

	var (
		stored any
		cost   int64 = 1
	)

	if c.serialize {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("invalid cache value: %w", err)
		}

		stored = b
		cost = int64(len(b))
	} else {
		stored = v
	}

	if ttl == 0 {
		if !c.cache.Set(key, stored, cost) {
			return ErrCacheClosed
		}

		return nil
	}

	if !c.cache.SetWithTTL(key, stored, cost, ttl) {
		return ErrCacheClosed
	}

	return nil
}

func (c *memoryCache[T]) getWithLoader(key string, loader func(string) (any, error)) (any, error) {
	v, err := loader(key)
	if err != nil {
		return nil, err
	}

	vv, _ := v.(T)

	if err = c.set(key, vv, c.ttl); err != nil {
		return nil, err
	}

	if !c.serialize {
		return vv, nil
	}

	b, err := json.Marshal(vv)
	if err != nil {
		return nil, fmt.Errorf("invalid cache value: %w", err)
	}

	return b, nil
}

func (c *memoryCache[T]) Pop(ctx context.Context, key string) (T, error) {
	var val T
	if c.cache == nil {
		return val, ErrCacheClosed
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	finish := c.instrumenter.Observe(ctx, InstrumentationGet, key)

	i, exists := c.cache.Get(key)
	if !exists {
		finish(nil)

		return val, KeyNotFoundError{Key: key}
	}

	c.cache.Del(key)

	v, err := c.unmarshal(i)
	finish(err)

	return v, err
}

func (c *memoryCache[T]) Set(ctx context.Context, key string, value T, opts ...ItemOption[T]) error {
	finish := c.instrumenter.Observe(ctx, InstrumentationSet, key)

	opt := newItemOptions(opts...)

	ttl := opt.TTL
	if ttl == 0 {
		ttl = c.ttl
	}

	err := c.set(key, value, ttl)

	finish(err)

	return err
}

func (c *memoryCache[T]) Delete(ctx context.Context, key string) error {
	if c.cache == nil {
		return ErrCacheClosed
	}

	finish := c.instrumenter.Observe(ctx, InstrumentationDelete, key)
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
