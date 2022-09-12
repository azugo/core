package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func envModeSet(mode Environment) func() {
	_ = os.Setenv("ENVIRONMENT", string(mode))

	return func() {
		_ = os.Unsetenv("ENVIRONMENT")
	}
}

func TestProductionMode(t *testing.T) {
	t.Cleanup(envModeSet(EnvironmentProduction))

	env := NewEnvironment(EnvironmentDevelopment)
	assert.True(t, env.IsProduction())
}

func TestStagingMode(t *testing.T) {
	t.Cleanup(envModeSet(EnvironmentStaging))

	env := NewEnvironment(EnvironmentDevelopment)
	assert.True(t, env.IsStaging())
}

func TestDevelopmentMode(t *testing.T) {
	t.Cleanup(envModeSet(EnvironmentDevelopment))

	env := NewEnvironment(EnvironmentProduction)
	assert.True(t, env.IsDevelopment())
}

func TestDefaultMode(t *testing.T) {
	env := NewEnvironment(EnvironmentProduction)
	assert.True(t, env.IsProduction())
}

func TestInvalidMode(t *testing.T) {
	t.Cleanup(envModeSet("invalid"))

	env := NewEnvironment(EnvironmentProduction)
	assert.True(t, env.IsProduction())
}
