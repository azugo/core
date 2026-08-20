// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package cache provides a unified caching layer with memory and Redis backends.
package cache

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"

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

// ErrCacheUnavailable is returned when the cache backend connection is not
// established or is lost.
var ErrCacheUnavailable = errors.New("cache connection unavailable")

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
	mu          sync.RWMutex
	redisCon    valkey.Client
	redisConStr string
	closed      bool
	bgctx       context.Context
}

// New creates a new cache with specified type.
func New(opts ...Option) *Cache {
	return &Cache{
		options:     opts,
		cache:       make(map[string]any),
		redisConStr: newCacheOptions(opts...).ConnectionString,
		bgctx:       context.Background(),
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
//
// An unreachable backend does not fail the start; the connection is retried
// in the background and operations return ErrCacheUnavailable until it is
// established.
func (c *Cache) Start(ctx context.Context) error {
	c.bgctx = ctx

	opt := newCacheOptions(c.options...)

	finish := opt.Instrumenter.Observe(ctx, InstrumentationStart)

	if opt.Type != RedisCache && opt.Type != RedisClusterCache && opt.Type != RedisSentinelCache {
		finish(nil)

		return nil
	}

	if err := ValidateConnectionString(opt.Type, opt.ConnectionString); err != nil {
		finish(err)

		return err
	}

	con, err := newRedisConnection(opt)
	if err != nil {
		finish(err)

		go c.reconnect(ctx, opt)

		return nil
	}

	c.mu.Lock()
	c.redisCon = con
	c.mu.Unlock()

	finish(nil)

	return nil
}

// reconnect retries the shared connection with exponential backoff until it
// succeeds, the cache is closed or ctx is cancelled.
func (c *Cache) reconnect(ctx context.Context, opt *cacheOptions) {
	con, err := connectRetry(ctx, opt)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		con.Close()

		return
	}

	c.redisCon = con
}

// Close cache and all its instances.
func (c *Cache) Close() {
	opt := newCacheOptions(c.options...)

	finish := opt.Instrumenter.Observe(context.Background(), InstrumentationClose)
	defer finish(nil)

	c.mu.Lock()
	c.closed = true
	instances := c.cache
	c.cache = nil
	con := c.redisCon
	c.redisCon = nil
	c.mu.Unlock()

	for _, i := range instances {
		if c, ok := i.(InstanceCloser); ok {
			c.Close()
		}
	}

	if con != nil {
		con.Close()
	}
}

// Connection returns the underlying Redis client shared by the cache or nil
// if the connection is not established.
func (c *Cache) Connection() valkey.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()

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

	if opt.Type != MemoryCache {
		con := c.Connection()
		if con == nil {
			finish(ErrCacheUnavailable)

			return ErrCacheUnavailable
		}

		if err := connError(con.Do(ctx, con.B().Ping().Build()).Error()); err != nil {
			finish(err)

			return err
		}
	}

	c.mu.RLock()

	instances := make([]any, 0, len(c.cache))
	for _, i := range c.cache {
		instances = append(instances, i)
	}

	c.mu.RUnlock()

	for _, i := range instances {
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
	cache.mu.RLock()
	i, ok := cache.cache[name]
	cache.mu.RUnlock()

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
	case RedisCache, RedisClusterCache, RedisSentinelCache:
		c, err = newRedisCache[T](name, cache, opt...)
		if err != nil {
			return nil, err
		}
	}

	if c != nil {
		cache.mu.Lock()
		defer cache.mu.Unlock()

		if cache.closed {
			return nil, ErrCacheClosed
		}

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

	var (
		copt valkey.ClientOption
		err  error
	)

	//nolint:exhaustive // memory cache type require no validation
	switch typ {
	case RedisCache, RedisClusterCache:
		copt, err = valkey.ParseURL(connStr)
	case RedisSentinelCache:
		copt, err = parseRedisSentinelURL(connStr)
	default:
		return fmt.Errorf("unsupported cache type: %v", typ)
	}

	if err != nil {
		return err
	}

	if copt.TLSConfig == nil {
		if u, err := url.Parse(connStr); err == nil {
			if ok, _ := strconv.ParseBool(u.Query().Get("skip_verify")); ok {
				return errors.New("skip_verify requires a TLS connection string scheme")
			}
		}
	}

	return nil
}
