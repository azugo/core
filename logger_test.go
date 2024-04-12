package core

import (
	"testing"

	"github.com/go-quicktest/qt"
	"go.uber.org/zap"
)

func TestLogFields(t *testing.T) {
	a, stop, logs, err := newTestApp()
	qt.Assert(t, qt.IsNil(err))
	t.Cleanup(stop)

	err = a.Start()
	qt.Assert(t, qt.IsNil(err))

	a.Log().Info("test", zap.String("field", "value"))

	qt.Assert(t, qt.HasLen(logs.All(), 2))
	entry := logs.All()[1]
	qt.Check(t, qt.Equals(entry.Message, "test"))
	qt.Check(t, qt.Equals(entry.Level, zap.InfoLevel))
	fields := entry.ContextMap()
	qt.Check(t, qt.Equals(fields["field"], "value"))
	qt.Check(t, qt.Equals(fields["service.name"], "Test"))
	qt.Check(t, qt.Equals(fields["service.version"], "1.0.0"))
	qt.Check(t, qt.Equals(fields["service.environment"], "development"))
}
