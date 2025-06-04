// Copyright 2025 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"azugo.io/core/validation"

	"github.com/spf13/viper"
)

// Logger represents the logging output configuration.
type Logger struct {
	// Type of the logger (allowed values are `console`, `file` and other registered logging targets).
	Type string `mapstructure:"type" validate:"required"`
	// Level of the logging output (defaults to `info`).
	Level string `mapstructure:"level" validate:"omitempty,oneof=debug info warn error dpanic panic fatal"`
	// Format of the logging output (defaults to `console` in development environment and `ecsjson` in staging and production).
	Format string `mapstructure:"format" validate:"omitempty,oneof=console json ecsjson"`
	// Output location (type sepcific output location).
	Output string `mapstructure:"output" validate:"omitempty"`
}

// Log configuration section.
type Log struct {
	// Type of the logger (defaults to `console`, allowed also `file` or other registered logging targets).
	Type string `mapstructure:"type" validate:"omitempty"`
	// Level of the logging output (defaults to `debug` in development environment and `info` in staging and production).
	Level string `mapstructure:"level" validate:"omitempty,oneof=debug info warn error dpanic panic fatal"`
	// Format of the logging output (defaults to `console` in development environment and `ecsjson` in staging and production).
	Format string `mapstructure:"format" validate:"omitempty,oneof=console json ecsjson"`
	// Output location (defaults to stderr)
	Output string `mapstructure:"output"`

	// Secondary logging output configuration.
	Secondary *Logger `mapstructure:"secondary" validate:"omitempty"`
}

// Validate logger configuration section.
func (c *Log) Validate(valid *validation.Validate) error {
	return valid.Struct(c)
}

// Bind logger configuration section.
func (c *Log) Bind(prefix string, v *viper.Viper) {
	v.SetDefault(prefix+".type", "console")
	v.SetDefault(prefix+".output", "stderr")

	_ = v.BindEnv(prefix+".type", "LOG_TYPE")
	_ = v.BindEnv(prefix+".level", "LOG_LEVEL")
	_ = v.BindEnv(prefix+".format", "LOG_FORMAT")
	_ = v.BindEnv(prefix+".output", "LOG_OUTPUT")

	_ = v.BindEnv(prefix+".secondary.type", "LOG_TYPE_SECONDARY")
	_ = v.BindEnv(prefix+".secondary.level", "LOG_LEVEL_SECONDARY")
	_ = v.BindEnv(prefix+".secondary.format", "LOG_FORMAT_SECONDARY")
	_ = v.BindEnv(prefix+".secondary.output", "LOG_OUTPUT_SECONDARY")
}
