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

	"github.com/dgraph-io/ristretto/v2"
	"github.com/goccy/go-json"
)

type memoryCache[T any] struct {
	cache           *ristretto.Cache[string, T]
	serializedCache *ristretto.Cache[string, []byte]
	ttl             time.Duration
	serialize       bool
	lock            sync.Mutex
	loader          func(ctx context.Context, key string) (any, error)
	instrumenter    instrumenter.Instrumenter
}

func newMemoryCache[T any](opts ...Option) (Instance[T], error) {
	opt := newCacheOptions(opts...)

	mc := &memoryCache[T]{
		ttl:          opt.TTL,
		serialize:    opt.Serialize,
		instrumenter: opt.Instrumenter,
	}

	if opt.Serialize {
		c, err := ristretto.NewCache(&ristretto.Config[string, []byte]{
			NumCounters: 1000,
			MaxCost:     1 << 30,
			BufferItems: 64,
		})
		if err != nil {
			return nil, err
		}

		mc.serializedCache = c
	} else {
		c, err := ristretto.NewCache(&ristretto.Config[string, T]{
			NumCounters: 1000,
			MaxCost:     1 << 30,
			BufferItems: 64,
		})
		if err != nil {
			return nil, err
		}

		mc.cache = c
	}

	if opt.Loader != nil {
		loader := opt.Loader
		mc.loader = func(ctx context.Context, key string) (any, error) {
			finish := opt.Instrumenter.Observe(ctx, InstrumentationLoader, key)
			v, err := loader(ctx, key)
			finish(err)

			return v, err
		}
	}

	return mc, nil
}

func (c *memoryCache[T]) unmarshal(b []byte) (T, error) {
	val := new(T)
	if err := json.Unmarshal(b, val); err != nil {
		return *val, fmt.Errorf("invalid cache value: %w", err)
	}

	return *val, nil
}

func (c *memoryCache[T]) loadAndCache(ctx context.Context, key string) (T, error) {
	var zero T

	raw, err := c.loader(ctx, key)
	if err != nil {
		return zero, err
	}

	v, ok := raw.(T)
	if !ok {
		return zero, fmt.Errorf("invalid value from loader: %v", raw)
	}

	if err = c.set(key, v, c.ttl); err != nil {
		return zero, err
	}

	return v, nil
}

func (c *memoryCache[T]) Get(ctx context.Context, key string, _ ...ItemOption[T]) (T, error) {
	finish := c.instrumenter.Observe(ctx, InstrumentationGet, key)

	var val T

	if c.serialize {
		if c.serializedCache == nil {
			finish(ErrCacheClosed)

			return val, ErrCacheClosed
		}

		if b, found := c.serializedCache.Get(key); found {
			v, err := c.unmarshal(b)
			finish(err)

			return v, err
		}
	} else {
		if c.cache == nil {
			finish(ErrCacheClosed)

			return val, ErrCacheClosed
		}

		if v, found := c.cache.Get(key); found {
			finish(nil)

			return v, nil
		}
	}

	if c.loader != nil {
		v, err := c.loadAndCache(ctx, key)
		finish(err)

		return v, err
	}

	finish(nil)

	return val, nil
}

func (c *memoryCache[T]) set(key string, v T, ttl time.Duration) error {
	if c.serialize {
		if c.serializedCache == nil {
			return ErrCacheClosed
		}

		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("invalid cache value: %w", err)
		}

		cost := int64(len(b))

		if ttl == 0 {
			if !c.serializedCache.Set(key, b, cost) {
				return ErrCacheClosed
			}
		} else {
			if !c.serializedCache.SetWithTTL(key, b, cost, ttl) {
				return ErrCacheClosed
			}
		}

		return nil
	}

	if c.cache == nil {
		return ErrCacheClosed
	}

	if ttl == 0 {
		if !c.cache.Set(key, v, 1) {
			return ErrCacheClosed
		}
	} else {
		if !c.cache.SetWithTTL(key, v, 1, ttl) {
			return ErrCacheClosed
		}
	}

	return nil
}

func (c *memoryCache[T]) Pop(ctx context.Context, key string) (T, error) {
	var val T

	c.lock.Lock()
	defer c.lock.Unlock()

	finish := c.instrumenter.Observe(ctx, InstrumentationGet, key)

	if c.serialize {
		if c.serializedCache == nil {
			finish(ErrCacheClosed)

			return val, ErrCacheClosed
		}

		b, exists := c.serializedCache.Get(key)
		if !exists {
			finish(nil)

			return val, KeyNotFoundError{Key: key}
		}

		c.serializedCache.Del(key)

		v, err := c.unmarshal(b)
		finish(err)

		return v, err
	}

	if c.cache == nil {
		finish(ErrCacheClosed)

		return val, ErrCacheClosed
	}

	v, exists := c.cache.Get(key)
	if !exists {
		finish(nil)

		return val, KeyNotFoundError{Key: key}
	}

	c.cache.Del(key)

	finish(nil)

	return v, nil
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
	finish := c.instrumenter.Observe(ctx, InstrumentationDelete, key)
	defer finish(nil)

	if c.serialize {
		if c.serializedCache == nil {
			return ErrCacheClosed
		}

		c.serializedCache.Del(key)
	} else {
		if c.cache == nil {
			return ErrCacheClosed
		}

		c.cache.Del(key)
	}

	return nil
}

func (c *memoryCache[T]) Close() {
	if c.serialize {
		if c.serializedCache != nil {
			c.serializedCache.Clear()
			c.serializedCache = nil
		}
	} else {
		if c.cache != nil {
			c.cache.Clear()
			c.cache = nil
		}
	}
}
