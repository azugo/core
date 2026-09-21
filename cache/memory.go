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

	"github.com/dgraph-io/ristretto/v2"
	"github.com/goccy/go-json"
)

type memoryCache[T any] struct {
	cache           *ristretto.Cache[string, T]
	serializedCache *ristretto.Cache[string, []byte]
	name            string
	prefix          string
	ttl             time.Duration
	serialize       bool
	lock            sync.Mutex
	loader          func(ctx context.Context, key string) (any, error)
	instrumenter    instrumenter.Instrumenter
}

func newMemoryCache[T any](prefix string, opts ...Option) (Instance[T], error) {
	opt := newCacheOptions(opts...)

	keyPrefix := opt.KeyPrefix
	if keyPrefix != "" {
		keyPrefix += ":"
	}

	mc := &memoryCache[T]{
		name:         prefix,
		prefix:       keyPrefix + prefix + ":",
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
			finish := instrumenter.ObserveKey(ctx, opt.Instrumenter, InstrumentationLoader, key)
			v, err := loader(ctx, key)
			finish(err)

			return v, err
		}
	}

	return mc, nil
}

func (c *memoryCache[T]) observe(ctx context.Context, op, key string) func(error) {
	if c.instrumenter.Empty() {
		return instrumenter.NullFinish
	}

	return c.instrumenter.Observe(ctx, op, c.prefix+key, string(MemoryCache), c.name)
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
	finish := c.observe(ctx, InstrumentationGet, key)

	var val T

	if c.serialize {
		if c.serializedCache == nil {
			finish(ErrCacheClosed)

			return val, ErrCacheClosed
		}

		if b, found := c.serializedCache.Get(key); found {
			c.observe(ctx, InstrumentationGetHit, key)(nil)

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
			c.observe(ctx, InstrumentationGetHit, key)(nil)
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

// Sync waits until every buffered write has been applied.
func (c *memoryCache[T]) Sync(context.Context) error {
	if c.serialize {
		if c.serializedCache == nil {
			return ErrCacheClosed
		}

		c.serializedCache.Wait()

		return nil
	}

	if c.cache == nil {
		return ErrCacheClosed
	}

	c.cache.Wait()

	return nil
}

func (c *memoryCache[T]) Pop(ctx context.Context, key string) (T, error) {
	var val T

	c.lock.Lock()
	defer c.lock.Unlock()

	finish := c.observe(ctx, InstrumentationGet, key)

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
		c.serializedCache.Wait()

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
	c.cache.Wait()

	finish(nil)

	return v, nil
}

func (c *memoryCache[T]) Set(ctx context.Context, key string, value T, opts ...ItemOption[T]) error {
	finish := c.observe(ctx, InstrumentationSet, key)

	opt := newItemOptions(opts...)

	ttl := opt.TTL
	if ttl == 0 {
		ttl = c.ttl
	}

	err := c.set(key, value, ttl)

	finish(err)

	return err
}

// Add stores value under key when no value is present.
func (c *memoryCache[T]) Add(ctx context.Context, key string, value T, opts ...ItemOption[T]) (bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	finish := c.observe(ctx, InstrumentationSet, key)

	_, found, err := c.peek(key)
	if err != nil || found {
		finish(err)

		return false, err
	}

	if err := c.write(key, value, opts...); err != nil {
		finish(err)

		return false, err
	}

	finish(nil)

	return true, nil
}

// Swap stores value under key and returns the value it replaced.
func (c *memoryCache[T]) Swap(ctx context.Context, key string, value T, opts ...ItemOption[T]) (T, bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	finish := c.observe(ctx, InstrumentationSet, key)

	prev, found, err := c.peek(key)
	if err != nil {
		finish(err)

		return prev, false, err
	}

	if err := c.write(key, value, opts...); err != nil {
		finish(err)

		return prev, false, err
	}

	finish(nil)

	return prev, found, nil
}

// Expire sets the remaining lifetime of key, keeping its value.
func (c *memoryCache[T]) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, errors.New("cache: expire requires a positive ttl")
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	finish := c.observe(ctx, InstrumentationSet, key)

	// Ristretto cannot re-time an entry in place, so the value is written back unchanged.
	val, found, err := c.peek(key)
	if err != nil || !found {
		finish(err)

		return false, err
	}

	if err := c.write(key, val, TTL[T](ttl)); err != nil {
		finish(err)

		return false, err
	}

	finish(nil)

	return true, nil
}

// TTL returns how much lifetime key has left and whether it is present at all.
func (c *memoryCache[T]) TTL(ctx context.Context, key string) (time.Duration, bool, error) {
	finish := c.observe(ctx, InstrumentationGet, key)

	if c.serialize {
		if c.serializedCache == nil {
			finish(ErrCacheClosed)

			return 0, false, ErrCacheClosed
		}

		ttl, found := c.serializedCache.GetTTL(key)

		finish(nil)

		return ttl, found, nil
	}

	if c.cache == nil {
		finish(ErrCacheClosed)

		return 0, false, ErrCacheClosed
	}

	ttl, found := c.cache.GetTTL(key)

	finish(nil)

	return ttl, found, nil
}

// peek returns the stored value without recording a hit or consulting the loader.
func (c *memoryCache[T]) peek(key string) (T, bool, error) {
	var val T

	if c.serialize {
		if c.serializedCache == nil {
			return val, false, ErrCacheClosed
		}

		b, found := c.serializedCache.Get(key)
		if !found {
			return val, false, nil
		}

		v, err := c.unmarshal(b)

		return v, err == nil, err
	}

	if c.cache == nil {
		return val, false, ErrCacheClosed
	}

	v, found := c.cache.Get(key)

	return v, found, nil
}

// write stores value and blocks until it is visible, so a caller holding the lock is never
// overtaken by its own pending write.
func (c *memoryCache[T]) write(key string, value T, opts ...ItemOption[T]) error {
	ttl := newItemOptions(opts...).TTL
	if ttl == 0 {
		ttl = c.ttl
	}

	if err := c.set(key, value, ttl); err != nil {
		return err
	}

	if c.serialize {
		c.serializedCache.Wait()
	} else {
		c.cache.Wait()
	}

	return nil
}

func (c *memoryCache[T]) Delete(ctx context.Context, key string) error {
	finish := c.observe(ctx, InstrumentationDelete, key)
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
