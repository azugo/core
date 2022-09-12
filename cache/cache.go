// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v9"
)

const (
	InstrumentationCacheStart  = "cache-start"
	InstrumentationCacheClose  = "cache-close"
	InstrumentationCachePing   = "cache-ping"
	InstrumentationCacheGet    = "cache-get"
	InstrumentationCacheLoader = "cache-loader"
	InstrumentationCacheSet    = "cache-set"
	InstrumentationCacheDelete = "cache-delete"
)

var ErrCacheClosed = errors.New("cache closed")

type ErrKeyNotFound struct {
	Key string
}

func (e ErrKeyNotFound) Error() string {
	return fmt.Sprintf("Key '%s' not found in cache", e.Key)
}

// Cache represents a cache.
type Cache struct {
	options     []CacheOption
	cache       map[string]any
	redisCon    *redis.Client
	redisConStr string
}

// New creates a new cache with specified type.
func New(opts ...CacheOption) *Cache {
	c := &Cache{
		options: opts,
		cache:   make(map[string]any),
	}
	return c
}

// CacheInstance represents a cache instance.
type CacheInstance[T any] interface {
	// Get value from cache. If value is not found, it will return default value.
	Get(ctx context.Context, key string, opts ...ItemOption[T]) (T, error)
	// Pop returns value from tha cache and deletes it. If value is not found, it will return ErrKeyNotFound error.
	Pop(ctx context.Context, key string) (T, error)
	// Set value in cache.
	Set(ctx context.Context, key string, value T, opts ...ItemOption[T]) error
	// Delete value from cache.
	Delete(ctx context.Context, key string) error
}

// CacheInstanceCloser represents a cache instance close method.
type CacheInstanceCloser interface {
	// Close cache instance.
	Close()
}

// CacheInstancePinger represents a cache instance ping method.
type CacheInstancePinger interface {
	Ping(ctx context.Context) error
}

// Start cache.
func (c *Cache) Start(ctx context.Context) error {
	opt := newCacheOptions(c.options...)

	finish := opt.Instrumenter.Observe(ctx, InstrumentationCacheStart)

	if opt.Type == RedisCache {
		con, err := newRedisClient(opt.ConnectionString, opt.ConnectionPassword)
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

	finish := opt.Instrumenter.Observe(context.Background(), InstrumentationCacheClose)
	defer finish(nil)

	if opt.Type == RedisCache {
		_ = c.redisCon.Close()
		c.redisCon = nil
	}
	for _, i := range c.cache {
		if c, ok := i.(CacheInstanceCloser); ok {
			c.Close()
		}
	}
	c.cache = nil
}

// Ping cache and all its instances.
func (c *Cache) Ping(ctx context.Context) error {
	opt := newCacheOptions(c.options...)

	finish := opt.Instrumenter.Observe(ctx, InstrumentationCachePing)

	if opt.Type == RedisCache && c.redisCon != nil {
		if s := c.redisCon.Ping(ctx); s != nil && s.Err() != nil {
			finish(s.Err())
			return s.Err()
		}
	}
	for _, i := range c.cache {
		if c, ok := i.(CacheInstancePinger); ok {
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
func Get[T any](cache *Cache, name string) (CacheInstance[T], error) {
	i, ok := cache.cache[name]
	if !ok {
		return nil, errors.New("cache not found")
	}
	r, ok := i.(CacheInstance[T])
	if !ok {
		return nil, errors.New("invalid cache type")
	}
	return r, nil
}

// Create new cache instance with specified name and options.
func Create[T any](cache *Cache, name string, opts ...CacheOption) (CacheInstance[T], error) {
	opt := append(append([]CacheOption{}, cache.options...), opts...)

	o := newCacheOptions(opt...)

	var c CacheInstance[T]
	var err error

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
		c, err = newRedisCache[T](name, con, opt...)
		if err != nil {
			return nil, err
		}
	}
	if c != nil {
		cache.cache[name] = c
		return c, nil
	}
	return nil, errors.New("unsupported cache type")
}

// ValidateConnectionString validates connection string for specific cache type.
func ValidateConnectionString(typ CacheType, connStr string) error {
	if typ == RedisCache {
		if len(connStr) == 0 {
			return errors.New("connection string can not be empty")
		}
		if _, err := redis.ParseURL(connStr); err != nil {
			return err
		}
		return nil
	}
	return nil
}
