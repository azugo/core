// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"os"
	"runtime/debug"
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

func (a *App) initLogger() {
	a.loglock.Lock()
	defer a.loglock.Unlock()

	fmt.Println("initLogger")
	debug.PrintStack()

	if a.logger != nil {
		fmt.Printf("logger: %#v\n", a.logger)
		return
	}

	info := system.CollectInfo()

	if a.Env().IsDevelopment() && !info.IsContainer() {
		fmt.Println("initLogger: development mode")
		conf := zap.NewDevelopmentEncoderConfig()
		conf.EncodeLevel = zapcore.CapitalColorLevelEncoder

		a.logger = zap.New(
			zapcore.NewCore(
				zapcore.NewConsoleEncoder(conf),
				zapcore.AddSync(colorable.NewColorableStdout()),
				parseLogLevel(os.Getenv("LOG_LEVEL"), zap.DebugLevel),
			),
			zap.AddCaller(),
			zap.AddStacktrace(zap.ErrorLevel),
		).With(a.loggerFields(info)...)
		fmt.Printf("logger: %#v\n", a.logger)

		return
	}

	fmt.Println("initLogger: staging/prod mode")

	core := ecszap.NewCore(
		ecszap.NewDefaultEncoderConfig(),
		os.Stdout,
		parseLogLevel(os.Getenv("LOG_LEVEL"), zap.InfoLevel),
	)

	a.logger = zap.New(core, zap.AddCaller(), zap.Fields(a.loggerFields(info)...))
	fmt.Printf("logger: %#v\n", a.logger)
}

// ReplaceLogger replaces current application logger with custom.
//
// Default fields are automatically added to the logger.
func (a *App) ReplaceLogger(logger *zap.Logger) error {
	fmt.Printf("replace logger: %#v\n", logger)
	a.logger = logger.With(a.loggerFields(system.CollectInfo())...)
	fmt.Printf("logger: %#v\n", logger)
	return nil
}

// Log returns application logger.
func (a *App) Log() *zap.Logger {
	if a.logger == nil {
		a.initLogger()
	}
	fmt.Printf("logger: %#v\n", a.logger)
	return a.logger
}

func parseLogLevel(level string, defaultLevel zapcore.Level) zapcore.Level {
	l, err := zapcore.ParseLevel(level)
	if err != nil {
		return defaultLevel
	}
	return l
}
