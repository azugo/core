// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"strings"
	"sync"

	"azugo.io/core/system"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logFormatConsole = "console"
	logFormatECSJSON = "ecsjson"
	logFormatJSON    = "json"
)

var (
	logOutputDriversMu sync.RWMutex
	logOutputDrivers   = make(map[string]func(app *App, format, output string, level zapcore.Level) (zapcore.Core, error))
)

// RegisterLogDriver registers a new log type driver.
func RegisterLogDriver(name string, fn func(app *App, format, output string, level zapcore.Level) (zapcore.Core, error)) {
	logOutputDriversMu.Lock()
	defer logOutputDriversMu.Unlock()

	logOutputDrivers[name] = fn
}

func (a *App) loggerFields() []zap.Field {
	info := system.CollectInfo()

	fields := make([]zap.Field, 0, 3)
	if a.AppName != "" {
		fields = append(fields, zap.String("service.name", a.AppName))
	}

	if a.AppVer != "" {
		fields = append(fields, zap.String("service.version", a.AppVer))
	}

	fields = append(fields, zap.String("service.environment", strings.ToLower(string(a.Env()))))

	if info != nil {
		if info.Hostname != "" {
			fields = append(fields, zap.String("host.hostname", info.Hostname))
		}

		if info.IsContainer() {
			if info.Container.ID != "" {
				fields = append(fields, zap.String("container.id", info.Container.ID))
			}

			if info.IsKubernetes() {
				if info.Container.Kubernetes.Namespace != "" {
					fields = append(fields, zap.String("kubernetes.namespace", info.Container.Kubernetes.Namespace))
				}

				if info.Container.Kubernetes.PodName != "" {
					fields = append(fields, zap.String("kubernetes.pod.name", info.Container.Kubernetes.PodName))
				}

				if info.Container.Kubernetes.PodUID != "" {
					fields = append(fields, zap.String("kubernetes.pod.uid", info.Container.Kubernetes.PodUID))
				}

				if info.Container.Kubernetes.NodeName != "" {
					fields = append(fields, zap.String("kubernetes.node.name", info.Container.Kubernetes.NodeName))
				}
			}
		}
	}

	return fields
}

func (a *App) initLogger() error {
	a.loglock.Lock()
	defer a.loglock.Unlock()

	if a.logger != nil {
		return nil
	}

	conf := a.Config().Log

	var logLevel zapcore.Level

	if a.Env().IsDevelopment() {
		logLevel = parseLogLevel(conf.Level, zap.DebugLevel)
	} else {
		logLevel = parseLogLevel(conf.Level, zap.InfoLevel)
	}

	driver, ok := logOutputDrivers[conf.Type]
	if !ok {
		return fmt.Errorf("unsupported log driver %q", conf.Type)
	}

	core, err := driver(a, conf.Format, conf.Output, logLevel)
	if err != nil {
		return fmt.Errorf("failed to initialize log driver %q: %w", conf.Type, err)
	}

	if conf.Secondary != nil {
		// Secondary logger is configured.
		driver, ok = logOutputDrivers[conf.Secondary.Type]
		if !ok {
			return fmt.Errorf("unsupported log driver %q", conf.Secondary.Type)
		}

		secondaryCore, err := driver(a, conf.Secondary.Format, conf.Secondary.Output, parseLogLevel(conf.Secondary.Level, zap.InfoLevel))
		if err != nil {
			return fmt.Errorf("failed to initialize log driver %q: %w", conf.Secondary.Type, err)
		}

		core = zapcore.NewTee(core, secondaryCore)
	}

	opts := []zap.Option{
		zap.AddCaller(),
	}

	if a.Env().IsDevelopment() {
		opts = append(opts,
			zap.Development(),
			zap.AddStacktrace(zapcore.ErrorLevel),
		)
	}

	a.logger = zap.New(core, opts...).With(a.loggerFields()...)

	return nil
}

// ReplaceLogger replaces current application logger with custom.
//
// Default fields are automatically added to the logger.
func (a *App) ReplaceLogger(logger *zap.Logger) error {
	a.logger = logger.With(a.loggerFields()...)

	return nil
}

// Log returns application logger.
func (a *App) Log() *zap.Logger {
	if a.logger == nil {
		if err := a.initLogger(); err != nil {
			panic(fmt.Sprintf("failed to initialize logger: %v", err))
		}
	}

	return a.logger
}

func parseLogLevel(level string, defaultLevel zapcore.Level) zapcore.Level {
	l, err := zapcore.ParseLevel(level)
	if err != nil || level == "" {
		return defaultLevel
	}

	return l
}
