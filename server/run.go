// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// Runnable provides methods to run application that will gracefully stop.
type Runnable interface {
	Start() error
	Stop()

	String() string
	Log() *zap.Logger
}

// Run starts an application and waits for it to finish.
func Run(a Runnable) {
	// Catch interrupts for gracefully stopping background node proecess
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := a.Start(); err != nil {
			a.Log().With(zap.Error(err)).Fatal("Failed to start service")

			return
		}

		a.Log().Info(fmt.Sprintf("Starting %s...", a.String()))
	}()

	s := <-done
	signal.Stop(done)

	a.Log().Info(fmt.Sprintf("Received %s signal, stopping service", s))

	a.Stop()
}
