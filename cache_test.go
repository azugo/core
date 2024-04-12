package core

import (
	"context"
	"testing"
	"time"

	"azugo.io/core/cache"

	"github.com/go-quicktest/qt"
)

func TestCacheInstrumentation(t *testing.T) {
	a, cleanup, _, err := newTestApp()
	qt.Assert(t, qt.IsNil(err))
	t.Cleanup(cleanup)

	actions := make([]string, 0, 4)

	a.Instrumentation(func(ctx context.Context, op string, args ...any) func(err error) {
		actions = append(actions, op)
		return func(err error) {
			actions = append(actions, op+":end")
		}
	})

	err = a.Start()
	qt.Assert(t, qt.IsNil(err))

	c, err := cache.Create[string](a.Cache(), "test", cache.Loader(func(_ context.Context, key string) (any, error) {
		return "loaded", nil
	}))
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.IsNil(c.Set(context.TODO(), "key", "value")))
	// Even if a Set gets applied, it might take a few milliseconds after the call has returned to the user.
	// In database terms, it is an eventual consistency model.
	time.Sleep(10 * time.Millisecond)
	_, err = c.Get(context.TODO(), "key")
	qt.Check(t, qt.IsNil(err))
	err = c.Delete(context.TODO(), "key")
	qt.Check(t, qt.IsNil(err))

	val, err := c.Get(context.TODO(), "key")
	qt.Check(t, qt.Equals(val, "loaded"))
	qt.Check(t, qt.IsNil(err))

	a.Cache().Ping(context.TODO())

	a.Stop()

	qt.Check(t, qt.DeepEquals(actions, []string{
		cache.InstrumentationStart,
		cache.InstrumentationStart + ":end",

		cache.InstrumentationSet,
		cache.InstrumentationSet + ":end",

		cache.InstrumentationGet,
		cache.InstrumentationGet + ":end",

		cache.InstrumentationDelete,
		cache.InstrumentationDelete + ":end",

		cache.InstrumentationGet,
		cache.InstrumentationLoader,
		cache.InstrumentationLoader + ":end",
		cache.InstrumentationGet + ":end",

		cache.InstrumentationPing,
		cache.InstrumentationPing + ":end",

		cache.InstrumentationClose,
		cache.InstrumentationClose + ":end",
	}))
}
