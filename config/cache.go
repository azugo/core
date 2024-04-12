// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"time"

	"azugo.io/core/cache"
	"azugo.io/core/validation"

	"github.com/spf13/viper"
)

type Cache struct {
	Type             cache.Type    `mapstructure:"type" validate:"required,oneof=memory redis redis-cluster"`
	TTL              time.Duration `mapstructure:"ttl" validate:"omitempty,min=0"`
	ConnectionString string        `mapstructure:"connection" validate:"omitempty"`
	Password         string        `mapstructure:"password" validate:"omitempty"`
	KeyPrefix        string        `mapstructure:"key_prefix" validate:"omitempty"`
}

// Validate cache configuration section.
func (c *Cache) Validate(valid *validation.Validate) error {
	if err := valid.Struct(c); err != nil {
		return err
	}

	if err := cache.ValidateConnectionString(c.Type, c.ConnectionString); err != nil {
		return err
	}

	return nil
}

// Bind cache configuration section.
func (c *Cache) Bind(prefix string, v *viper.Viper) {
	psw, _ := LoadRemoteSecret("CACHE_PASSWORD")

	v.SetDefault(prefix+".type", "memory")
	v.SetDefault(prefix+".password", psw)

	_ = v.BindEnv(prefix+".type", "CACHE_TYPE")
	_ = v.BindEnv(prefix+".ttl", "CACHE_TTL")
	_ = v.BindEnv(prefix+".connection", "CACHE_CONNECTION")
	_ = v.BindEnv(prefix+".key_prefix", "CACHE_KEY_PREFIX")
}
