// Package test provides testing helpers for Azugo applications.
package test

import (
	"azugo.io/core"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// ObservedLogs replaces the app logger with an in-memory observer and returns
// the observed log entries for use in tests.
func ObservedLogs(a *core.App) *observer.ObservedLogs {
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	_ = a.ReplaceLogger(zap.New(observedZapCore))

	return observedLogs
}
