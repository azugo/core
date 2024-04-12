package http

import (
	"azugo.io/core/validation"
)

// NamedClient represents the configuration for the named client instance.
type NamedClient struct {
	BaseURI string `mapstructure:"base_uri" validate:"required http_url"`
}

// Configuration represents the configuration for the HTTP client.
type Configuration struct {
	Clients map[string]NamedClient `mapstructure:"clients"`
}

// apply the configuration to the options.
func (c *Configuration) apply(o *options) {
	o.Configuration = c
}

// Validate Metrics configuration section.
func (c *Configuration) Validate(valid *validation.Validate) error {
	for _, client := range c.Clients {
		if err := valid.Struct(client); err != nil {
			return err
		}
	}

	return nil
}
