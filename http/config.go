package http

// NamedClient represents the configuration for the named client instance.
type NamedClient struct {
	BaseURI string `mapstructure:"base_uri" validate:"required"`
}

// Configuration represents the configuration for the HTTP client.
type Configuration struct {
	Clients map[string]NamedClient `mapstructure:"clients"`
}

// apply the configuration to the options.
func (c *Configuration) apply(o *options) {
	o.Configuration = c
}
