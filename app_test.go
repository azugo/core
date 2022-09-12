package core

import (
	"os"
	"testing"

	"azugo.io/core/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestApp() (*App, func(), *observer.ObservedLogs, error) {
	os.Setenv("ENVIRONMENT", string(EnvironmentDevelopment))
	a := New()

	conf := config.New()
	if err := conf.Load(nil, conf, string(a.Env())); err != nil {
		return nil, func() {}, nil, err
	}
	a.AppName = "Test"
	a.SetVersion("1.0.0", "test")
	a.SetConfig(nil, conf)

	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	a.ReplaceLogger(zap.New(observedZapCore))

	return a, func() {
		a.Stop()
		_ = os.Unsetenv("ENVIRONMENT")
	}, observedLogs, nil
}

func TestNewApp(t *testing.T) {
	a, cleanup, logs, err := newTestApp()
	require.NoError(t, err)
	require.NotNil(t, a)

	t.Cleanup(cleanup)

	assert.NoError(t, a.Start())

	assert.Equal(t, EnvironmentDevelopment, a.Env())
	assert.Len(t, logs.All(), 1)
	assert.Equal(t, "Starting Test 1.0.0 (built with test)...", logs.All()[0].Message)
	assert.Equal(t, "Test 1.0.0 (built with test)", a.String())
}
