package core

import (
	"os"
	"testing"

	"github.com/go-quicktest/qt"
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
	qt.Check(t, qt.IsTrue(env.IsProduction()))
}

func TestStagingMode(t *testing.T) {
	t.Cleanup(envModeSet(EnvironmentStaging))

	env := NewEnvironment(EnvironmentDevelopment)
	qt.Check(t, qt.IsTrue(env.IsStaging()))
}

func TestDevelopmentMode(t *testing.T) {
	t.Cleanup(envModeSet(EnvironmentDevelopment))

	env := NewEnvironment(EnvironmentProduction)
	qt.Check(t, qt.IsTrue(env.IsDevelopment()))
}

func TestDefaultMode(t *testing.T) {
	env := NewEnvironment(EnvironmentProduction)
	qt.Check(t, qt.IsTrue(env.IsProduction()))
}

func TestInvalidMode(t *testing.T) {
	t.Cleanup(envModeSet("invalid"))

	env := NewEnvironment(EnvironmentProduction)
	qt.Check(t, qt.IsTrue(env.IsProduction()))
}
