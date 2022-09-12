// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"os"
	"strings"

	"github.com/mattn/go-colorable"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var canColorStdout = false

func (a *App) loggerFields() []zap.Field {
	// TODO: add additional fields for logger

	fields := make([]zap.Field, 0, 3)
	if a.AppName != "" {
		fields = append(fields, zap.String("service.name", a.AppName))
	}
	if a.AppVer != "" {
		fields = append(fields, zap.String("service.version", a.AppVer))
	}
	fields = append(fields, zap.String("service.environment", strings.ToLower(string(a.Env()))))

	return fields
}

func (a *App) initLogger() error {
	if a.logger != nil {
		return nil
	}

	if canColorStdout && a.Env().IsDevelopment() {
		conf := zap.NewDevelopmentEncoderConfig()
		conf.EncodeLevel = zapcore.CapitalColorLevelEncoder

		a.logger = zap.New(
			zapcore.NewCore(
				zapcore.NewConsoleEncoder(conf),
				zapcore.AddSync(colorable.NewColorableStdout()),
				zap.DebugLevel,
			),
			zap.AddCaller(),
			zap.AddStacktrace(zap.ErrorLevel),
		).With(a.loggerFields()...)

		return nil
	}

	encoderConfig := ecszap.NewDefaultEncoderConfig()
	core := ecszap.NewCore(encoderConfig, os.Stdout, zap.InfoLevel)

	a.logger = zap.New(core, zap.AddCaller()).With(a.loggerFields()...)

	return nil
}

// ReplaceLogger replaces current application logger with custom.
//
// Default fields are automatically added to the logger.
func (a *App) ReplaceLogger(logger *zap.Logger) error {
	a.logger = logger.With(a.loggerFields()...)
	return a.initLogger()
}

// Log returns application logger.
func (a *App) Log() *zap.Logger {
	if a.logger == nil {
		if err := a.initLogger(); err != nil {
			panic(err)
		}
	}
	return a.logger
}
