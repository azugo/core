package core

import (
	"context"
	"testing"

	"azugo.io/core/cache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheInstrumentation(t *testing.T) {
	a, cleanup, _, err := newTestApp()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	actions := make([]string, 0, 4)

	a.Instrumentation(func(ctx context.Context, op string, args ...any) func(err error) {
		actions = append(actions, op)
		return func(err error) {
			actions = append(actions, op+":end")
		}
	})

	require.NoError(t, a.Start())

	c, err := cache.Create[string](a.Cache(), "test", cache.Loader(func(_ context.Context, key string) (any, error) {
		return "loaded", nil
	}))
	require.NoError(t, err)

	assert.NoError(t, c.Set(context.TODO(), "key", "value"))
	_, err = c.Get(context.TODO(), "key")
	assert.NoError(t, err)
	err = c.Delete(context.TODO(), "key")

	val, err := c.Get(context.TODO(), "key")
	assert.Equal(t, "loaded", val)
	assert.NoError(t, err)

	a.Cache().Ping(context.TODO())

	a.Stop()

	assert.Equal(t, []string{
		cache.InstrumentationCacheStart,
		cache.InstrumentationCacheStart + ":end",

		cache.InstrumentationCacheSet,
		cache.InstrumentationCacheSet + ":end",

		cache.InstrumentationCacheGet,
		cache.InstrumentationCacheGet + ":end",

		cache.InstrumentationCacheDelete,
		cache.InstrumentationCacheDelete + ":end",

		cache.InstrumentationCacheGet,
		cache.InstrumentationCacheLoader,
		cache.InstrumentationCacheLoader + ":end",
		cache.InstrumentationCacheGet + ":end",

		cache.InstrumentationCachePing,
		cache.InstrumentationCachePing + ":end",

		cache.InstrumentationCacheClose,
		cache.InstrumentationCacheClose + ":end",
	}, actions)
}
