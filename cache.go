// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"azugo.io/core/cache"
)

func (a *App) initCache() error {
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

	a.cache = cache.New(opts...)

	return a.cache.Start(a.BackgroundContext())
}

func (a *App) closeCache() {
	if a.cache == nil {
		return
	}

	a.cache.Close()
}

func (a *App) Cache() *cache.Cache {
	if a.cache == nil {
		if err := a.initCache(); err != nil {
			panic(err)
		}
	}

	return a.cache
}
