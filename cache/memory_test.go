package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCacheGetSet(t *testing.T) {
	c := New(CacheType(MemoryCache))
	err := c.Start(context.TODO())
	require.NoError(t, err)
	defer c.Close()

	i, err := Create[string](c, "test")
	require.NoError(t, err)

	err = i.Set(context.TODO(), "key", "value")
	assert.NoError(t, err)

	val, err := i.Get(context.TODO(), "key")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestMemoryCachePop(t *testing.T) {
	c := New(CacheType(MemoryCache))
	err := c.Start(context.TODO())
	require.NoError(t, err)
	defer c.Close()

	i, err := Create[string](c, "test")
	require.NoError(t, err)

	err = i.Set(context.TODO(), "key", "value")
	assert.NoError(t, err)

	val, err := i.Pop(context.TODO(), "key")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)

	val, err = i.Pop(context.TODO(), "key")
	assert.Error(t, err)
	assert.Empty(t, val)
}

func TestMemoryCacheDelete(t *testing.T) {
	c := New(CacheType(MemoryCache))
	err := c.Start(context.TODO())
	require.NoError(t, err)
	defer c.Close()

	i, err := Create[string](c, "test")
	require.NoError(t, err)

	err = i.Set(context.TODO(), "key", "value")
	assert.NoError(t, err)

	err = i.Delete(context.TODO(), "key")
	assert.NoError(t, err)

	val, err := i.Get(context.TODO(), "key")
	assert.NoError(t, err)
	assert.Empty(t, val)
}

func TestMemoryCacheExpire(t *testing.T) {
	c := New(CacheType(MemoryCache))
	err := c.Start(context.TODO())
	require.NoError(t, err)
	defer c.Close()

	i, err := Create[string](c, "test", DefaultTTL(100*time.Millisecond))
	require.NoError(t, err)

	err = i.Set(context.TODO(), "key", "value")
	assert.NoError(t, err)

	time.Sleep(150 * time.Millisecond)

	val, err := i.Get(context.TODO(), "key")
	assert.NoError(t, err)
	assert.Empty(t, val)
}

func TestMemoryCacheItemExpire(t *testing.T) {
	c := New(CacheType(MemoryCache))
	err := c.Start(context.TODO())
	require.NoError(t, err)
	defer c.Close()

	i, err := Create[string](c, "test")
	require.NoError(t, err)

	err = i.Set(context.TODO(), "key", "value", TTL[string](100*time.Millisecond))
	assert.NoError(t, err)

	time.Sleep(150 * time.Millisecond)

	val, err := i.Get(context.TODO(), "key")
	assert.NoError(t, err)
	assert.Empty(t, val)
}
