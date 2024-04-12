// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package server

import (
	"errors"

	"azugo.io/core"
	"azugo.io/core/config"

	"github.com/spf13/cobra"
)

type Options struct {
	// AppName is the name of the application.
	AppName string
	// AppVer is the version of the application.
	AppVer string
	// AppBuiltWith is the server build tags.
	AppBuiltWith string

	// Configuration object that implements config.Configurable interface.
	Configuration any
}

// New returns new Azugo pre-configured core server with default configuration.
func New(cmd *cobra.Command, opt Options) (*core.App, error) {
	a := core.New()
	a.AppName = opt.AppName
	a.SetVersion(opt.AppVer, opt.AppBuiltWith)

	// Support extended configuration.
	var conf *config.Configuration

	c := opt.Configuration
	if c == nil {
		conf = config.New()
		c = conf
	} else if configurable, ok := c.(config.Configurable); ok {
		conf = configurable.Core()
	} else {
		return nil, errors.New("configuration must implement Configurable interface")
	}

	a.SetConfig(cmd, conf)

	// Load configuration
	if err := conf.Load(cmd, c, string(a.Env())); err != nil {
		return nil, err
	}

	return a, nil
}
