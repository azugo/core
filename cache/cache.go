// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package cache provides a unified caching layer with memory and Redis backends.
package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/valkey-io/valkey-go"
)

// Instrumentation operation names for cache events.
const (
	InstrumentationStart  = "cache-start"
	InstrumentationClose  = "cache-close"
	InstrumentationPing   = "cache-ping"
	InstrumentationGet    = "cache-get"
	InstrumentationGetHit = "cache-get-hit"
	InstrumentationLoader = "cache-loader"
	InstrumentationSet    = "cache-set"
	InstrumentationDelete = "cache-delete"
)

// ErrCacheClosed is returned when an operation is attempted on a closed cache.
var ErrCacheClosed = errors.New("cache closed")

// KeyNotFoundError is returned when a cache key is not found.
type KeyNotFoundError struct {
	Key string
}

func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("Key '%s' not found in cache", e.Key)
}

// Cache represents a cache.
type Cache struct {
	options     []Option
	cache       map[string]any
	redisCon    valkey.Client
	redisConStr string
}

// New creates a new cache with specified type.
func New(opts ...Option) *Cache {
	return &Cache{
		options: opts,
		cache:   make(map[string]any),
	}
}

// Instance of a cache.
type Instance[T any] interface {
	// Get value from cache. If value is not found, it will return default value.
	Get(ctx context.Context, key string, opts ...ItemOption[T]) (T, error)
	// Pop returns value from tha cache and deletes it. If value is not found, it will return ErrKeyNotFound error.
	Pop(ctx context.Context, key string) (T, error)
	// Set value in cache.
	Set(ctx context.Context, key string, value T, opts ...ItemOption[T]) error
	// Delete value from cache.
	Delete(ctx context.Context, key string) error
}

// InstanceCloser represents a cache instance close method.
type InstanceCloser interface {
	// Close cache instance.
	Close()
}

// InstancePinger represents a cache instance ping method.
type InstancePinger interface {
	Ping(ctx context.Context) error
}

// Start cache.
func (c *Cache) Start(ctx context.Context) error {
	opt := newCacheOptions(c.options...)

	finish := opt.Instrumenter.Observe(ctx, InstrumentationStart)

	if opt.Type != RedisCache && opt.Type != RedisClusterCache && opt.Type != RedisSentinelCache {
		finish(nil)

		return nil
	}

	var (
		con valkey.Client
		err error
	)

	//nolint:exhaustive // check is already done above
	switch opt.Type {
	case RedisCache, RedisClusterCache:
		con, err = newRedisClient(opt)
	case RedisSentinelCache:
		con, err = newRedisSentinelClient(opt)
	}

	if err != nil {
		finish(err)

		return err
	}

	c.redisCon = con
	c.redisConStr = opt.ConnectionString

	finish(nil)

	return nil
}

// Close cache and all its instances.
func (c *Cache) Close() {
	opt := newCacheOptions(c.options...)

	finish := opt.Instrumenter.Observe(context.Background(), InstrumentationClose)
	defer finish(nil)

	switch opt.Type {
	case RedisCache, RedisClusterCache, RedisSentinelCache:
		if c.redisCon != nil {
			c.redisCon.Close()
		}

		c.redisCon = nil
	case MemoryCache:
		// nothing to close
	}

	for _, i := range c.cache {
		if c, ok := i.(InstanceCloser); ok {
			c.Close()
		}
	}

	c.cache = nil
}

// Connection returns the underlying Redis client shared by the cache.
func (c *Cache) Connection() valkey.Client {
	return c.redisCon
}

// ConfiguredType returns the cache type the cache was created with.
func (c *Cache) ConfiguredType() Type {
	opt := newCacheOptions(c.options...)
	if opt.Type == "" {
		return MemoryCache
	}

	return opt.Type
}

// ConfiguredKeyPrefix returns the global key prefix the cache was created with.
func (c *Cache) ConfiguredKeyPrefix() string {
	return newCacheOptions(c.options...).KeyPrefix
}

// Ping cache and all its instances.
func (c *Cache) Ping(ctx context.Context) error {
	opt := newCacheOptions(c.options...)

	finish := opt.Instrumenter.Observe(ctx, InstrumentationPing)

	if opt.Type != MemoryCache && c.redisCon != nil {
		if err := c.redisCon.Do(ctx, c.redisCon.B().Ping().Build()).Error(); err != nil {
			finish(err)

			return err
		}
	}

	for _, i := range c.cache {
		if c, ok := i.(InstancePinger); ok {
			if err := c.Ping(ctx); err != nil {
				finish(err)

				return err
			}
		}
	}

	finish(nil)

	return nil
}

// Get returns pre-configured cache instance by name.
func Get[T any](cache *Cache, name string) (Instance[T], error) {
	i, ok := cache.cache[name]
	if !ok {
		return nil, errors.New("cache not found")
	}

	r, ok := i.(Instance[T])
	if !ok {
		return nil, errors.New("invalid cache type")
	}

	return r, nil
}

// Create new cache instance with specified name and options.
func Create[T any](cache *Cache, name string, opts ...Option) (Instance[T], error) {
	opt := append(append([]Option{}, cache.options...), opts...)

	o := newCacheOptions(opt...)

	var (
		c   Instance[T]
		err error
	)

	switch o.Type {
	case MemoryCache:
		c, err = newMemoryCache[T](name, opt...)
		if err != nil {
			return nil, err
		}
	case RedisCache, RedisClusterCache:
		con := cache.redisCon
		if o.ConnectionString != cache.redisConStr {
			con, err = newRedisClient(o)
			if err != nil {
				return nil, err
			}
		}

		c = newRedisCache[T](name, con, opt...)
	case RedisSentinelCache:
		con := cache.redisCon
		if o.ConnectionString != cache.redisConStr {
			con, err = newRedisSentinelClient(o)
			if err != nil {
				return nil, err
			}
		}

		c = newRedisCache[T](name, con, opt...)
	}

	if c != nil {
		cache.cache[name] = c

		return c, nil
	}

	return nil, errors.New("unsupported cache type")
}

// ValidateConnectionString validates connection string for specific cache type.
func ValidateConnectionString(typ Type, connStr string) error {
	if typ == MemoryCache {
		return nil
	}

	if len(connStr) == 0 {
		return errors.New("connection string can not be empty")
	}

	var err error

	//nolint:exhaustive // memory cache type require no validation
	switch typ {
	case RedisCache, RedisClusterCache:
		_, err = parseRedisURL(connStr)
	case RedisSentinelCache:
		_, err = parseRedisSentinelURL(connStr)
	default:
		return fmt.Errorf("unsupported cache type: %v", typ)
	}

	return err
}
