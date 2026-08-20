// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"azugo.io/core/cache"

	"go.uber.org/zap"
)

func (a *App) initCache() error {
	a.cachelock.Lock()
	defer a.cachelock.Unlock()

	if a.cache != nil {
		return nil
	}

	conf := a.Config().Cache
	opts := []cache.Option{
		conf.Type,
		cache.Instrumenter(a.Instrumenter()),
	}

	if conf.TTL > 0 {
		opts = append(opts, cache.DefaultTTL(conf.TTL))
	}

	if len(conf.ConnectionString) != 0 {
		opts = append(opts, cache.ConnectionString(conf.ConnectionString))
	}

	if len(conf.Password) != 0 {
		opts = append(opts, cache.ConnectionPassword(conf.Password))
	}

	if len(conf.KeyPrefix) != 0 {
		opts = append(opts, cache.KeyPrefix(conf.KeyPrefix))
	}

	if !conf.ClientCache {
		opts = append(opts, cache.ClientCache(false))
	}

	if conf.ClientCache && conf.ClientCacheTTL > 0 {
		opts = append(opts, cache.ClientCacheTTL(conf.ClientCacheTTL))
	}

	if conf.ClientCacheSize > 0 {
		opts = append(opts, cache.ClientCacheSize(conf.ClientCacheSize))
	}

	c := cache.New(opts...)

	if err := c.Start(a.BackgroundContext()); err != nil {
		return err
	}

	if err := c.Ping(a.BackgroundContext()); err != nil {
		a.Log().Warn("cache backend is unreachable, reconnecting in background", zap.Error(err))
	}

	a.cache = c

	return nil
}

func (a *App) closeCache() {
	a.cachelock.Lock()
	defer a.cachelock.Unlock()

	if a.cache == nil {
		return
	}

	a.cache.Close()
}

// Cache returns the application cache instance, initializing it on first use.
func (a *App) Cache() *cache.Cache {
	if err := a.initCache(); err != nil {
		panic(err)
	}

	a.cachelock.Lock()
	defer a.cachelock.Unlock()

	return a.cache
}
