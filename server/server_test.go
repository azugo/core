package server

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"azugo.io/core"
	"azugo.io/core/cache"
	"azugo.io/core/test"

	"github.com/go-quicktest/qt"
)

type App struct {
	*core.App

	wg sync.WaitGroup
}

func (a *App) Start() error {
	if err := a.App.Start(); err != nil {
		return err
	}

	a.wg.Add(1)

	// Start should not exit
	<-(chan int)(nil)
	return nil
}

func (a *App) Stop() {
	a.App.Stop()

	a.wg.Done()
}

func TestApp(t *testing.T) {
	a, err := New(nil, Options{
		AppName: "Test",
		AppVer:  "1.0.0",
	})
	qt.Assert(t, qt.IsNil(err))
	_ = test.ObservedLogs(a)
	app := &App{
		App: a,
	}

	go Run(app)
	time.Sleep(100 * time.Millisecond)

	qt.Check(t, qt.IsTrue(app.Config().Ready()))
	qt.Check(t, qt.Equals(a.Config().Cache.Type, cache.MemoryCache))
	qt.Check(t, qt.Equals(app.Env(), core.EnvironmentProduction))
	qt.Check(t, qt.IsNil(a.Cache().Ping(context.TODO())))

	// Signal interrupt to stop app
	proc, err := os.FindProcess(os.Getpid())
	qt.Assert(t, qt.IsNil(err))
	proc.Signal(os.Interrupt)
	// Wait for app to finish
	app.wg.Wait()
}
