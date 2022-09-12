package test

import (
	"azugo.io/core"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func ObservedLogs(a *core.App) *observer.ObservedLogs {
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	a.ReplaceLogger(zap.New(observedZapCore))

	return observedLogs
}
