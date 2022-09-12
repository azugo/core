package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLogFields(t *testing.T) {
	a, stop, logs, err := newTestApp()
	require.NoError(t, err)
	t.Cleanup(stop)

	require.NoError(t, a.Start())

	a.Log().Info("test", zap.String("field", "value"))

	require.Len(t, logs.All(), 2)
	entry := logs.All()[1]
	assert.Equal(t, "test", entry.Message)
	assert.Equal(t, zap.InfoLevel, entry.Level)
	fields := entry.ContextMap()
	assert.Equal(t, "value", fields["field"])
	assert.Equal(t, "Test", fields["service.name"])
	assert.Equal(t, "1.0.0", fields["service.version"])
	assert.Equal(t, "development", fields["service.environment"])
}
