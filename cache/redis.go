// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"azugo.io/core/instrumenter"

	"github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"
)

type redisCache[T any] struct {
	con            valkey.Client
	typ            Type
	name           string
	prefix         string
	ttl            time.Duration
	clientCacheTTL time.Duration
	loader         func(ctx context.Context, key string) (any, error)
	instrumenter   instrumenter.Instrumenter
}

func newRedisCache[T any](prefix string, con valkey.Client, opts ...Option) Instance[T] {
	opt := newCacheOptions(opts...)

	keyPrefix := opt.KeyPrefix
	if keyPrefix != "" {
		keyPrefix += ":"
	}

	loader := opt.Loader
	if loader != nil {
		loader = func(ctx context.Context, key string) (any, error) {
			finish := instrumenter.ObserveKey(ctx, opt.Instrumenter, InstrumentationLoader, key)
			v, err := opt.Loader(ctx, key)
			finish(err)

			return v, err
		}
	}

	return &redisCache[T]{
		con:            con,
		typ:            opt.Type,
		name:           prefix,
		prefix:         keyPrefix + prefix + ":",
		ttl:            opt.TTL,
		clientCacheTTL: opt.ClientCacheTTL,
		loader:         loader,
		instrumenter:   opt.Instrumenter,
	}
}

func (c *redisCache[T]) observe(ctx context.Context, op, key string) func(error) {
	return c.instrumenter.Observe(ctx, op, c.prefix+key, string(c.typ), c.name)
}

func parseRedisSentinelURL(urlStr string) (valkey.ClientOption, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return valkey.ClientOption{}, err
	}

	var tlsConfig *tls.Config

	switch u.Scheme {
	case "sentinel":
	case "sentinels":
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	default:
		return valkey.ClientOption{}, errors.New("redis sentinel URL must start with sentinel:// or sentinels:// scheme")
	}

	// Extract username if present
	username := ""
	if u.User != nil {
		username = u.User.Username()
	}

	masterName := strings.TrimPrefix(u.Path, "/")
	if masterName == "" {
		return valkey.ClientOption{}, errors.New("master name is required in sentinel URL path")
	}

	if u.Host == "" {
		return valkey.ClientOption{}, errors.New("sentinel addresses are required")
	}

	options := valkey.ClientOption{
		InitAddress: strings.Split(u.Host, ","),
		Username:    username,
		TLSConfig:   tlsConfig,
		Sentinel: valkey.SentinelOption{
			MasterSet: masterName,
			TLSConfig: tlsConfig,
		},
	}

	// Parse query parameters
	if u.RawQuery != "" {
		q := u.Query()

		if dbStr := q.Get("db"); dbStr != "" {
			db, err := strconv.Atoi(dbStr)
			if err != nil {
				return valkey.ClientOption{}, fmt.Errorf("invalid db value: %w", err)
			}

			options.SelectDB = db
		}

		if tlsConfig != nil && q.Get("skip_verify") == "true" {
			tlsConfig.InsecureSkipVerify = true
		}
	}

	return options, nil
}

func newValkeyClient(copt valkey.ClientOption, o *cacheOptions) (valkey.Client, error) {
	// If password is provided override provided in connection string.
	if len(o.ConnectionPassword) != 0 {
		copt.Password = o.ConnectionPassword
	}

	if !o.ClientCache {
		copt.DisableCache = true
	}

	if !copt.DisableCache && o.ClientCacheSize > 0 {
		copt.CacheSizeEachConn = o.ClientCacheSize
	}

	return valkey.NewClient(copt)
}

func newRedisClient(o *cacheOptions) (valkey.Client, error) {
	copt, err := valkey.ParseURL(o.ConnectionString)
	if err != nil {
		return nil, err
	}

	return newValkeyClient(copt, o)
}

func newRedisSentinelClient(o *cacheOptions) (valkey.Client, error) {
	copt, err := parseRedisSentinelURL(o.ConnectionString)
	if err != nil {
		return nil, err
	}

	return newValkeyClient(copt, o)
}

func (c *redisCache[T]) Get(ctx context.Context, key string, opts ...ItemOption[T]) (T, error) {
	val := new(T)
	if c.con == nil {
		return *val, ErrCacheClosed
	}

	finish := c.observe(ctx, InstrumentationGet, key)

	var res valkey.ValkeyResult
	if c.clientCacheTTL > 0 {
		res = c.con.DoCache(ctx, c.con.B().Get().Key(c.prefix+key).Cache(), c.clientCacheTTL)
	} else {
		res = c.con.Do(ctx, c.con.B().Get().Key(c.prefix+key).Build())
	}

	if res.IsCacheHit() {
		c.observe(ctx, InstrumentationGetHit, key)(nil)
	}

	v, err := res.ToString()

	if valkey.IsValkeyNil(err) {
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

	if err != nil {
		finish(err)

		return *val, err
	}

	if err := json.Unmarshal([]byte(v), val); err != nil {
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

	finishG := c.observe(ctx, InstrumentationGet, key)
	finishD := c.observe(ctx, InstrumentationDelete, key)

	v, err := c.con.Do(ctx, c.con.B().Getdel().Key(c.prefix+key).Build()).ToString()
	if valkey.IsValkeyNil(err) {
		finishD(nil)
		finishG(nil)

		return *val, KeyNotFoundError{Key: key}
	}

	if err != nil {
		finishD(err)
		finishG(err)

		return *val, err
	}

	if err := json.Unmarshal([]byte(v), val); err != nil {
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

	finish := c.observe(ctx, InstrumentationSet, key)

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

	cmd := c.con.B().Set().Key(c.prefix + key).Value(string(buf))

	var completed valkey.Completed
	if ttl > 0 {
		completed = cmd.Px(ttl).Build()
	} else {
		completed = cmd.Build()
	}

	if err := c.con.Do(ctx, completed).Error(); err != nil {
		finish(err)

		return err
	}

	finish(nil)

	return nil
}

func (c *redisCache[T]) Delete(ctx context.Context, key string) error {
	if c.con == nil {
		return ErrCacheClosed
	}

	finish := c.observe(ctx, InstrumentationDelete, key)

	if err := c.con.Do(ctx, c.con.B().Del().Key(c.prefix+key).Build()).Error(); err != nil {
		finish(err)

		return err
	}

	finish(nil)

	return nil
}

func (c *redisCache[T]) Ping(ctx context.Context) error {
	if c.con == nil {
		return nil
	}

	return c.con.Do(ctx, c.con.B().Ping().Build()).Error()
}

func (c *redisCache[T]) Close() error {
	if c.con == nil {
		return nil
	}

	c.con.Close()
	c.con = nil

	return nil
}
