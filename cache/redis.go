// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"context"
	"fmt"
	"time"

	"azugo.io/core/instrumenter"

	"github.com/go-redis/redis/v9"
	"github.com/goccy/go-json"
)

type redisCache[T any] struct {
	con          *redis.Client
	prefix       string
	ttl          time.Duration
	loader       func(ctx context.Context, key string) (interface{}, error)
	instrumenter instrumenter.Instrumenter
}

func newRedisCache[T any](prefix string, con *redis.Client, opts ...CacheOption) (CacheInstance[T], error) {
	opt := newCacheOptions(opts...)

	keyPrefix := opt.KeyPrefix
	if keyPrefix != "" {
		keyPrefix += ":"
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

	return &redisCache[T]{
		con:          con,
		prefix:       keyPrefix + prefix + ":",
		ttl:          opt.TTL,
		loader:       loader,
		instrumenter: opt.Instrumenter,
	}, nil
}

func newRedisClient(constr, password string) (*redis.Client, error) {
	redisOptions, err := redis.ParseURL(constr)
	if err != nil {
		return nil, err
	}
	// If password is provided override provided in connection string.
	if len(password) != 0 {
		redisOptions.Password = password
	}

	return redis.NewClient(redisOptions), nil
}

func (c *redisCache[T]) Get(ctx context.Context, key string, opts ...ItemOption[T]) (T, error) {
	val := new(T)
	if c.con == nil {
		return *val, ErrCacheClosed
	}

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheGet, c.prefix+key)

	s := c.con.Get(ctx, c.prefix+key)
	if s.Err() == redis.Nil {
		if c.loader != nil {
			v, err := c.loader(ctx, key)
			if err != nil {
				finish(err)
				return *val, err
			}
			vv, ok := v.(T)
			if !ok {
				err = fmt.Errorf("invalid value from loader: %v", v)
				finish(err)
				return *val, err
			}
			if err := c.Set(ctx, key, vv, opts...); err != nil {
				finish(err)
				return *val, err
			}
			return vv, nil
		}
		return *val, nil
	}
	if s.Err() != nil {
		finish(s.Err())
		return *val, s.Err()
	}
	if err := json.Unmarshal([]byte(s.Val()), val); err != nil {
		err = fmt.Errorf("invalid cache value: %w", err)
		finish(err)
		return *val, err
	}
	finish(nil)
	return *val, nil
}

func (c *redisCache[T]) Pop(ctx context.Context, key string) (T, error) {
	val := new(T)
	if c.con == nil {
		return *val, ErrCacheClosed
	}

	finishG := c.instrumenter.Observe(ctx, InstrumentationCacheGet, c.prefix+key)
	finishD := c.instrumenter.Observe(ctx, InstrumentationCacheDelete, c.prefix+key)

	s := c.con.GetDel(ctx, c.prefix+key)
	if s.Err() == redis.Nil {
		finishD(nil)
		finishG(nil)
		return *val, ErrKeyNotFound{Key: key}
	}
	if s.Err() != nil {
		finishD(s.Err())
		finishG(s.Err())
		return *val, s.Err()
	}
	if err := json.Unmarshal([]byte(s.Val()), val); err != nil {
		err = fmt.Errorf("invalid cache value: %w", err)
		finishD(err)
		finishG(err)
		return *val, err
	}
	finishD(nil)
	finishG(nil)
	return *val, nil
}

func (c *redisCache[T]) Set(ctx context.Context, key string, value T, opts ...ItemOption[T]) error {
	if c.con == nil {
		return ErrCacheClosed
	}

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheSet, c.prefix+key)

	buf, err := json.Marshal(value)
	if err != nil {
		err = fmt.Errorf("invalid cache value: %w", err)
		finish(err)
		return err
	}
	opt := newItemOptions(opts...)
	ttl := c.ttl
	if opt.TTL != 0 {
		ttl = opt.TTL
	}
	s := c.con.Set(ctx, c.prefix+key, string(buf), ttl)
	if s.Err() != nil {
		finish(s.Err())
		return s.Err()
	}
	finish(nil)
	return nil
}

func (c *redisCache[T]) Delete(ctx context.Context, key string) error {
	if c.con == nil {
		return ErrCacheClosed
	}

	finish := c.instrumenter.Observe(ctx, InstrumentationCacheSet, c.prefix+key)

	s := c.con.Del(ctx, c.prefix+key)
	if s.Err() != nil {
		finish(s.Err())
		return s.Err()
	}
	finish(nil)
	return nil
}

func (c *redisCache[T]) Ping(ctx context.Context) error {
	if c.con == nil {
		return nil
	}
	s := c.con.Ping(ctx)
	if s.Err() != nil {
		return s.Err()
	}
	return nil
}

func (c *redisCache[T]) Close() {
	if c.con == nil {
		return
	}
	_ = c.con.Close()
	c.con = nil
}
