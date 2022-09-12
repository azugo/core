package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getRedisConnStr() string {
	return os.Getenv("REDIS_CONNSTR")
}

func TestReidsCacheGetSet(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skipped()
		return
	}
	c := New(CacheType(RedisCache), KeyPrefix("prefix"), ConnectionString(cs))
	err := c.Start(context.TODO())
	require.NoError(t, err)
	defer c.Close()

	i, err := Create[string](c, "test")
	require.NoError(t, err)

	err = i.Set(context.TODO(), "key1", "value")
	assert.NoError(t, err)

	val, err := i.Get(context.TODO(), "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestRedisCachePop(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skipped()
		return
	}
	c := New(CacheType(RedisCache), ConnectionString(cs))
	err := c.Start(context.TODO())
	require.NoError(t, err)
	defer c.Close()

	i, err := Create[string](c, "test")
	require.NoError(t, err)

	err = i.Set(context.TODO(), "key2", "value")
	assert.NoError(t, err)

	val, err := i.Pop(context.TODO(), "key2")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)

	val, err = i.Pop(context.TODO(), "key2")
	assert.Error(t, err)
	assert.Empty(t, val)
}

func TestRedisCacheDelete(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skipped()
		return
	}
	c := New(CacheType(RedisCache), ConnectionString(cs))
	err := c.Start(context.TODO())
	require.NoError(t, err)
	defer c.Close()

	i, err := Create[string](c, "test")
	require.NoError(t, err)

	err = i.Set(context.TODO(), "key3", "value")
	assert.NoError(t, err)

	err = i.Delete(context.TODO(), "key3")
	assert.NoError(t, err)

	val, err := i.Get(context.TODO(), "key3")
	assert.NoError(t, err)
	assert.Empty(t, val)
}

func TestRedisCacheExpire(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skipped()
		return
	}
	c := New(CacheType(RedisCache), ConnectionString(cs))
	err := c.Start(context.TODO())
	require.NoError(t, err)
	defer c.Close()

	i, err := Create[string](c, "test", DefaultTTL(100*time.Millisecond))
	require.NoError(t, err)

	err = i.Set(context.TODO(), "key4", "value")
	assert.NoError(t, err)

	time.Sleep(150 * time.Millisecond)

	val, err := i.Get(context.TODO(), "key4")
	assert.NoError(t, err)
	assert.Empty(t, val)
}
