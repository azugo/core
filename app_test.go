package core_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"azugo.io/core"
	"azugo.io/core/cache"
	"azugo.io/core/server"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	a, err := server.New(nil, server.ServerOptions{
		AppName: "Test",
		AppVer:  "1.0.0",
	})
	require.NoError(t, err)
	app := &App{
		App: a,
	}

	go core.Run(app)
	time.Sleep(100 * time.Millisecond)

	assert.True(t, app.Config().Ready())
	assert.Equal(t, cache.MemoryCache, a.Config().Cache.Type)
	assert.Equal(t, core.EnvironmentProduction, app.Env())
	assert.NoError(t, a.Cache().Ping(context.TODO()))

	// Signal interrupt to stop app
	proc, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	proc.Signal(os.Interrupt)
	// Wait for app to finish
	app.wg.Wait()
}
