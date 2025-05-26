package config

import (
	"azugo.io/core/validation"

	"github.com/spf13/viper"
)

// Log configuration section.
type Log struct {
	// Level of the logging output (defaults to debug in development environment and info in staging and production).
	Level string `mapstructure:"level" validate:"omitempty,oneof=debug info warn error dpanic panic fatal"`
	// Format of the logging output (defaults to console in development environment and ecsjson in staging and production).
	Format string `mapstructure:"format" validate:"omitempty,oneof=console json ecsjson"`
	// Output location (defaults to stderr)
	Output string `mapstructure:"output"`
}

// Validate logger configuration section.
func (c *Log) Validate(valid *validation.Validate) error {
	return valid.Struct(c)
}

// Bind logger configuration section.
func (c *Log) Bind(prefix string, v *viper.Viper) {
	v.SetDefault(prefix+".output", "stderr")

	_ = v.BindEnv(prefix+".level", "LOG_LEVEL")
	_ = v.BindEnv(prefix+".format", "LOG_FORMAT")
	_ = v.BindEnv(prefix+".output", "LOG_OUTPUT")
}
