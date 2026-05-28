// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"os"
	"strings"
)

// Environment type.
type Environment string

// Supported environment values.
const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTest        Environment = "test"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

// NewEnvironment creates new Environment instance.
func NewEnvironment(defaultMode Environment) Environment {
	env := Environment(strings.ToLower(os.Getenv("ENVIRONMENT")))
	if len(env) == 0 {
		env = defaultMode
	}

	if env == EnvironmentProduction || env == EnvironmentStaging || env == EnvironmentTest || env == EnvironmentDevelopment {
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

// IsTest checks if current environment is testing.
func (e Environment) IsTest() bool {
	return e == EnvironmentTest
}

// IsDevelopment checks if current environment is development.
func (e Environment) IsDevelopment() bool {
	return e == EnvironmentDevelopment
}
