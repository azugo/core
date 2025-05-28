// Copyright 2025 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	RegisterLogDriver("file", func(_ *App, format string, output string, level zapcore.Level) (zapcore.Core, error) {
		var target *url.URL

		if filepath.IsAbs(output) {
			target = &url.URL{
				Scheme: "file",
				Path:   output,
			}
		} else {
			u, err := url.Parse(output)
			if err != nil {
				return nil, fmt.Errorf("failed to parse log output %q: %w", output, err)
			}

			target = u
		}

		if target.Scheme != "file" {
			return nil, fmt.Errorf("unsupported log output scheme %q", target.Scheme)
		}

		f, err := os.OpenFile(target.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %q: %w", target.Path, err)
		}

		w := zapcore.AddSync(f)

		switch format {
		case logFormatECSJSON:
			// Use ECS JSON format.
			return ecszap.NewCore(
				ecszap.NewDefaultEncoderConfig(),
				w,
				level,
			), nil
		case logFormatJSON:
			return zapcore.NewCore(
				zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
				w,
				level,
			), nil
		case logFormatConsole:
			return zapcore.NewCore(
				zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
				w,
				level,
			), nil
		default:
			return nil, fmt.Errorf("unsupported log format %q", format)
		}
	})
}
