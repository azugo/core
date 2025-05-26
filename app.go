// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"sync"

	"azugo.io/core/cache"
	"azugo.io/core/config"
	"azugo.io/core/http"
	"azugo.io/core/instrumenter"
	"azugo.io/core/validation"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type App struct {
	env Environment

	// Background context
	bgctx  context.Context
	bgstop context.CancelFunc

	// Validate instance
	validate *validation.Validate

	// Logger
	loglock sync.Mutex
	logger  *zap.Logger

	// Configuration
	config *config.Configuration

	// Cache
	cache *cache.Cache

	// HTTP client
	httpClient http.Client

	// Tasks
	stlock  sync.RWMutex
	tasks   []Tasker
	started bool

	// Instrumenter
	instrumenter instrumenter.Instrumenter

	// App settings
	AppVer       string
	AppBuiltWith string
	AppName      string
}

func New() *App {
	ctx, stop := context.WithCancel(context.Background())

	return &App{
		env: NewEnvironment(EnvironmentProduction),

		bgctx:  ctx,
		bgstop: stop,

		tasks: make([]Tasker, 0),

		instrumenter: instrumenter.NullInstrumenter,

		validate: validation.New(),
	}
}

// SetVersion sets application version and built with tags.
func (a *App) SetVersion(version, builtWith string) {
	a.AppVer = version
	a.AppBuiltWith = builtWith
}

// Env returns the current application environment.
func (a *App) Env() Environment {
	return a.env
}

// Validate returns validation service instance.
func (a *App) Validate() *validation.Validate {
	return a.validate
}

// BackgroundContext returns global background context.
func (a *App) BackgroundContext() context.Context {
	return a.bgctx
}

func (a *App) String() string {
	name := a.AppName
	if len(name) == 0 {
		name = "Azugo"
	}

	bw := a.AppBuiltWith
	if len(bw) > 0 {
		bw = fmt.Sprintf(" (built with %s)", bw)
	}

	return fmt.Sprintf("%s %s%s", name, a.AppVer, bw)
}

// SetConfig binds application configuration to the application.
func (a *App) SetConfig(_ *cobra.Command, conf *config.Configuration) {
	if a.config != nil && a.config.Ready() {
		return
	}

	a.config = conf
}

// Config returns application configuration.
//
// Panics if configuration is not loaded.
func (a *App) Config() *config.Configuration {
	if a.config == nil || !a.config.Ready() {
		panic("configuration is not loaded")
	}

	return a.config
}

// Instrumentation defines callback to be used as instrumenter.
func (a *App) Instrumentation(instr instrumenter.Instrumenter) {
	if instr == nil {
		a.instrumenter = instrumenter.NullInstrumenter

		return
	}

	a.instrumenter = instr
}

// Instrumenter returns application instrumentation callback.
func (a *App) Instrumenter() instrumenter.Instrumenter {
	return a.instrumenter
}

// Start web application.
func (a *App) Start() error {
	if err := a.initLogger(); err != nil {
		return err
	}

	if err := a.initCache(); err != nil {
		return err
	}

	if err := a.startTasks(); err != nil {
		return err
	}

	a.initHTTPClient()

	a.Log().Info(fmt.Sprintf("Starting %s...", a.String()))

	return nil
}

// Stop application and its services.
func (a *App) Stop() {
	a.bgstop()

	a.stopTasks()

	a.closeCache()
}
