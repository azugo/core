// Copyright 2025 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"os"

	"azugo.io/core/system"

	"github.com/mattn/go-colorable"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	RegisterLogDriver("console", func(app *App, format string, output string, level zapcore.Level) (zapcore.Core, error) {
		info := system.CollectInfo()
		devOutput := app.Env().IsDevelopment() && !info.IsContainer()

		var target zapcore.WriteSyncer

		switch output {
		case "":
			fallthrough
		case "stderr":
			if devOutput {
				target = zapcore.AddSync(colorable.NewColorableStderr())
			} else {
				target = zapcore.AddSync(os.Stderr)
			}
		case "stdout":
			if devOutput {
				target = zapcore.AddSync(colorable.NewColorableStdout())
			} else {
				target = zapcore.AddSync(os.Stdout)
			}
		}

		switch {
		case (format == logFormatConsole || format == "") && devOutput:
			devenc := zap.NewDevelopmentEncoderConfig()

			if output == "" || output == "stdout" || output == "stderr" {
				devenc.EncodeLevel = zapcore.CapitalColorLevelEncoder
			}

			return zapcore.NewCore(
				zapcore.NewConsoleEncoder(devenc),
				target,
				level,
			), nil
		case format == logFormatECSJSON || format == "":
			// Use ECS JSON format.
			return ecszap.NewCore(
				ecszap.NewDefaultEncoderConfig(),
				target,
				level,
			), nil
		case format == logFormatConsole:
			// Use console format.
			return zapcore.NewCore(
				zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
				target,
				level,
			), nil
		case format == logFormatJSON:
			// Use JSON format.
			return zapcore.NewCore(
				zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
				target,
				level,
			), nil
		default:
			return nil, fmt.Errorf("unsupported log format %q", format)
		}
	})
}
