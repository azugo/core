// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"os"
)

// Environment type.
type Environment string

const (
	EnvironmentDevelopment Environment = "Development"
	EnvironmentStaging     Environment = "Staging"
	EnvironmentProduction  Environment = "Production"
)

// NewEnvironment creates new Environment instance.
func NewEnvironment(defaultMode Environment) Environment {
	env := Environment(os.Getenv("ENVIRONMENT"))
	if len(env) == 0 {
		env = defaultMode
	}

	if env == EnvironmentProduction || env == EnvironmentStaging || env == EnvironmentDevelopment {
		return env
	}

	return defaultMode
}

// IsProduction checks if current environment is production.
func (e Environment) IsProduction() bool {
	return e == EnvironmentProduction
}

// IsStaging checks if current environment is staging.
func (e Environment) IsStaging() bool {
	return e == EnvironmentStaging
}

// IsDevelopment checks if current environment is development.
func (e Environment) IsDevelopment() bool {
	return e == EnvironmentDevelopment
}
