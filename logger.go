// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"azugo.io/core/system"

	"github.com/mattn/go-colorable"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logOutputDriversMu sync.RWMutex
	logOutputDrivers   = make(map[string]func(output *url.URL) (zapcore.WriteSyncer, error))
)

func RegisterLogOutput(name string, fn func(output *url.URL) (zapcore.WriteSyncer, error)) {
	logOutputDriversMu.Lock()
	defer logOutputDriversMu.Unlock()

	logOutputDrivers[name] = fn
}

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
	a.loglock.Lock()
	defer a.loglock.Unlock()

	if a.logger != nil {
		return nil
	}

	info := system.CollectInfo()
	conf := a.Config().Log

	var (
		logLevel zapcore.Level
		core     zapcore.Core
		output   zapcore.WriteSyncer
	)

	if a.Env().IsDevelopment() {
		logLevel = parseLogLevel(conf.Level, zap.DebugLevel)
	} else {
		logLevel = parseLogLevel(conf.Level, zap.InfoLevel)
	}

	devOutput := a.Env().IsDevelopment() && !info.IsContainer()

	switch conf.Output {
	case "":
		fallthrough
	case "stderr":
		if devOutput {
			output = zapcore.AddSync(colorable.NewColorableStderr())
		} else {
			output = zapcore.AddSync(os.Stderr)
		}
	case "stdout":
		if devOutput {
			output = zapcore.AddSync(colorable.NewColorableStdout())
		} else {
			output = zapcore.AddSync(os.Stdout)
		}
	default:
		var path *url.URL

		if filepath.IsAbs(conf.Output) {
			path = &url.URL{
				Scheme: "file",
				Path:   conf.Output,
			}
		} else {
			u, err := url.Parse(conf.Output)
			if err != nil {
				return fmt.Errorf("failed to parse log output %q: %w", conf.Output, err)
			}

			path = u
		}

		driver, ok := logOutputDrivers[path.Scheme]
		if !ok {
			return fmt.Errorf("unsupported log output %q", path.Scheme)
		}

		f, err := driver(path)
		if err != nil {
			return fmt.Errorf("failed to create log output %q: %w", path, err)
		}

		output = f
	}

	switch {
	case (conf.Format == "console" || conf.Format == "") && devOutput:
		devenc := zap.NewDevelopmentEncoderConfig()

		if conf.Output == "" || conf.Output == "stdout" || conf.Output == "stderr" {
			devenc.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}

		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(devenc),
			output,
			logLevel,
		)
	case conf.Format == "ecsjson" || conf.Format == "":
		// Use ECS JSON format.
		core = ecszap.NewCore(
			ecszap.NewDefaultEncoderConfig(),
			output,
			logLevel,
		)
	case conf.Format == "console":
		// Use console format.
		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
			output,
			logLevel,
		)
	case conf.Format == "json":
		// Use JSON format.
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			output,
			logLevel,
		)
	default:
		return fmt.Errorf("unsupported log format %q", conf.Format)
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

	a.logger = zap.New(core, opts...).With(a.loggerFields(info)...)

	return nil
}

// ReplaceLogger replaces current application logger with custom.
//
// Default fields are automatically added to the logger.
func (a *App) ReplaceLogger(logger *zap.Logger) error {
	a.logger = logger.With(a.loggerFields(system.CollectInfo())...)

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

func init() {
	RegisterLogOutput("file", func(u *url.URL) (zapcore.WriteSyncer, error) {
		f, err := os.OpenFile(u.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %q: %w", u.Path, err)
		}

		return zapcore.AddSync(f), nil
	})
}
