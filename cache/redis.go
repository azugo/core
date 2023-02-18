// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"reflect"
	"time"

	"azugo.io/core/instrumenter"

	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
)

type redisCache[T any] struct {
	con          redis.Cmdable
	prefix       string
	ttl          time.Duration
	loader       func(ctx context.Context, key string) (interface{}, error)
	instrumenter instrumenter.Instrumenter
}

func newRedisCache[T any](prefix string, con redis.Cmdable, opts ...CacheOption) (CacheInstance[T], error) {
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

func parseCustomURLAttr(v string) (string, bool, error) {
	u, err := url.Parse(v)
	if err != nil {
		return "", false, err
	}
	var insecureSkipVerify bool
	if u.RawQuery != "" {
		q := u.Query()
		if q.Get("skip_verify") == "true" {
			insecureSkipVerify = true
			q.Del("skip_verify")
			u.RawQuery = q.Encode()
		}
	}
	return u.String(), insecureSkipVerify, nil
}

func ParseRedisClusterURL(v string) (*redis.ClusterOptions, error) {
	v, insecureSkipVerify, err := parseCustomURLAttr(v)
	if err != nil {
		return nil, err
	}
	o, err := redis.ParseClusterURL(v)
	if err == nil && insecureSkipVerify {
		if o.TLSConfig == nil {
			o.TLSConfig = &tls.Config{}
		}
		o.TLSConfig.InsecureSkipVerify = insecureSkipVerify
	}
	return o, err
}

func ParseRedisURL(v string) (*redis.Options, error) {
	v, insecureSkipVerify, err := parseCustomURLAttr(v)
	if err != nil {
		return nil, err
	}
	o, err := redis.ParseURL(v)
	if err == nil && insecureSkipVerify {
		if o.TLSConfig == nil {
			o.TLSConfig = &tls.Config{}
		}
		o.TLSConfig.InsecureSkipVerify = insecureSkipVerify
	}
	return o, err
}

func newRedisClient(constr, password string) (redis.Cmdable, error) {
	redisOptions, err := ParseRedisURL(constr)
	if err != nil {
		return nil, err
	}
	// If password is provided override provided in connection string.
	if len(password) != 0 {
		redisOptions.Password = password
	}
	// longer timeout for redis client to avoid timeout errors for slow redis servers
	redisOptions.DialTimeout = 10 * time.Second
	return redis.NewClient(redisOptions), nil
}

func newRedisClusterClient(constr, password string) (redis.Cmdable, error) {
	redisOptions, err := ParseRedisClusterURL(constr)
	if err != nil {
		return nil, err
	}
	// If password is provided override provided in connection string.
	if len(password) != 0 {
		redisOptions.Password = password
	}
	return redis.NewClusterClient(redisOptions), nil
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

func (c *redisCache[T]) Close() error {
	if c.con == nil {
		return nil
	}
	var err error
	switch v := c.con.(type) {
	case *redis.Client:
		err = v.Close()
	case *redis.ClusterClient:
		err = v.Close()
	case nil:
		// do nothing
	default:
		// this will not happen anyway, unless we mishandle it on `Init`
		panic(fmt.Sprintf("invalid redis client: %v", reflect.TypeOf(v)))
	}
	c.con = nil
	return err
}
