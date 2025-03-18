// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	InstrumentationStart  = "cache-start"
	InstrumentationClose  = "cache-close"
	InstrumentationPing   = "cache-ping"
	InstrumentationGet    = "cache-get"
	InstrumentationLoader = "cache-loader"
	InstrumentationSet    = "cache-set"
	InstrumentationDelete = "cache-delete"
)

var ErrCacheClosed = errors.New("cache closed")

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
	redisCon    redis.Cmdable
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

	if opt.Type == RedisCache || opt.Type == RedisClusterCache {
		var (
			con redis.Cmdable
			err error
		)

		if opt.Type == RedisCache {
			con, err = newRedisClient(opt.ConnectionString, opt.ConnectionPassword)
		} else if opt.Type == RedisClusterCache {
			con, err = newRedisClusterClient(opt.ConnectionString, opt.ConnectionPassword)
		} else {
			con, err = newRedisSentinelClient(opt.ConnectionString, opt.ConnectionPassword)
		}

		if err != nil {
			finish(err)

			return err
		}

		c.redisCon = con
		c.redisConStr = opt.ConnectionString
	}

	finish(nil)

	return nil
}

// Close cache and all its instances.
func (c *Cache) Close() {
	opt := newCacheOptions(c.options...)

	finish := opt.Instrumenter.Observe(context.Background(), InstrumentationClose)
	defer finish(nil)

	if opt.Type == RedisCache || opt.Type == RedisSentinelCache {
		if v, ok := c.redisCon.(*redis.Client); ok {
			_ = v.Close()
		}

		c.redisCon = nil
	} else if opt.Type == RedisClusterCache {
		if v, ok := c.redisCon.(*redis.ClusterClient); ok {
			_ = v.Close()
		}

		c.redisCon = nil
	}

	for _, i := range c.cache {
		if c, ok := i.(InstanceCloser); ok {
			c.Close()
		}
	}

	c.cache = nil
}

// Ping cache and all its instances.
func (c *Cache) Ping(ctx context.Context) error {
	opt := newCacheOptions(c.options...)

	finish := opt.Instrumenter.Observe(ctx, InstrumentationPing)

	if (opt.Type == RedisCache || opt.Type == RedisClusterCache) && c.redisCon != nil {
		if s := c.redisCon.Ping(ctx); s != nil && s.Err() != nil {
			finish(s.Err())

			return s.Err()
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
		c, err = newMemoryCache[T](opt...)
		if err != nil {
			return nil, err
		}
	case RedisCache:
		con := cache.redisCon
		if o.ConnectionString != cache.redisConStr {
			con, err = newRedisClient(o.ConnectionString, o.ConnectionPassword)
			if err != nil {
				return nil, err
			}
		}

		c = newRedisCache[T](name, con, opt...)
	case RedisClusterCache:
		con := cache.redisCon
		if o.ConnectionString != cache.redisConStr {
			con, err = newRedisClusterClient(o.ConnectionString, o.ConnectionPassword)
			if err != nil {
				return nil, err
			}
		}

		c = newRedisCache[T](name, con, opt...)
	case RedisSentinelCache:
		con := cache.redisCon
		if o.ConnectionString != cache.redisConStr {
			con, err = newRedisSentinelClient(o.ConnectionString, o.ConnectionPassword)
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
	if len(connStr) == 0 && typ != MemoryCache {
		return errors.New("connection string can not be empty")
	}

	var err error

	switch typ {
	case MemoryCache:
		// No validation needed for MemoryCache
		return nil
	case RedisCache:
		_, err = ParseRedisURL(connStr)
	case RedisClusterCache:
		_, err = ParseRedisClusterURL(connStr)
	case RedisSentinelCache:
		_, err = ParseRedisSentinelURL(connStr)
	default:
		return fmt.Errorf("unsupported cache type: %v", typ)
	}

	return err
}

// InstrGet returns cache key if the operation is cache get event.
func InstrGet(op string, args ...any) (string, bool) {
	if op != InstrumentationGet || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrSet returns cache key if the operation is cache set event.
func InstrSet(op string, args ...any) (string, bool) {
	if op != InstrumentationSet || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrDelete returns cache key if the operation is cache delete event.
func InstrDelete(op string, args ...any) (string, bool) {
	if op != InstrumentationDelete || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}

// InstrLoader returns cache key if the operation is cache loader event.
func InstrLoader(op string, args ...any) (string, bool) {
	if op != InstrumentationLoader || len(args) != 1 {
		return "", false
	}

	key, ok := args[0].(string)

	return key, ok
}
