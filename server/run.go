// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package server provides helpers for running Azugo applications as servers.
package server

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
)

// Runnable provides methods to run application that will gracefully stop.
type Runnable interface {
	Start() error
	Stop()

	String() string
	Log() *zap.Logger
}

// Run starts an application and waits for the context to be cancelled before
// stopping it gracefully.
func Run(ctx context.Context, a Runnable) {
	go func() {
		if err := a.Start(); err != nil {
			a.Log().With(zap.Error(err)).Fatal("Failed to start service")

			os.Exit(1)
		}

		a.Log().Info(fmt.Sprintf("Starting %s...", a.String()))
	}()

	<-ctx.Done()

	a.Log().Info(fmt.Sprintf("Stopping %s...", a.String()))

	a.Stop()
}
