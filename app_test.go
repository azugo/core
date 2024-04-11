package core

import (
	"os"
	"testing"

	"azugo.io/core/config"

	"github.com/go-quicktest/qt"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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
	if err := a.ReplaceLogger(zap.New(observedZapCore)); err != nil {
		return nil, func() {}, nil, err
	}

	return a, func() {
		a.Stop()
		_ = os.Unsetenv("ENVIRONMENT")
	}, observedLogs, nil
}

func TestNewApp(t *testing.T) {
	a, cleanup, logs, err := newTestApp()
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsNotNil(a))

	t.Cleanup(cleanup)

	err = a.Start()
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(a.Env(), EnvironmentDevelopment))
	qt.Check(t, qt.HasLen(logs.All(), 1))
	qt.Check(t, qt.Equals(logs.All()[0].Message, "Starting Test 1.0.0 (built with test)..."))
	qt.Check(t, qt.Equals(a.String(), "Test 1.0.0 (built with test)"))
}
