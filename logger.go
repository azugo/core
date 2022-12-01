// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"os"
	"strings"

	"azugo.io/core/system"

	"github.com/mattn/go-colorable"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func (a *App) loggerFields(info *system.Info) []zap.Field {
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
	if a.logger != nil {
		return nil
	}

	info := system.CollectInfo()

	if a.Env().IsDevelopment() && !info.IsContainer() {
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
		).With(a.loggerFields(info)...)

		return nil
	}

	encoderConfig := ecszap.NewDefaultEncoderConfig()
	core := ecszap.NewCore(encoderConfig, os.Stdout, zap.InfoLevel)

	a.logger = zap.New(core, zap.AddCaller()).With(a.loggerFields(info)...)

	return nil
}

// ReplaceLogger replaces current application logger with custom.
//
// Default fields are automatically added to the logger.
func (a *App) ReplaceLogger(logger *zap.Logger) error {
	a.logger = logger.With(a.loggerFields(system.CollectInfo())...)
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
