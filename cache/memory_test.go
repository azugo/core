package cache

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func loadCacheValueById(ctx context.Context, key string) (any, error) {
	// wait for 100ms to 700ms randomly to simulate slow loading
	b := make([]byte, 1)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Duration(100+int(b[0])%600) * time.Millisecond)
	return fmt.Sprintf("value-%s", key), nil
}

var testKeys = []string {
	"key1",
	"key2",
	"key3",
	"key4",
	"key5",
	"key6",
	"key7",
}

func TestMemoryCacheGetSetLoader(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skipped()
		return
	}
	c := New(CacheType(RedisCache), ConnectionString(cs))
	err := c.Start(context.TODO())
	require.NoError(t, err)
	defer c.Close()

	i, err := Create[string](c, "test",
		DefaultTTL(5*time.Minute),
		Loader(loadCacheValueById),
		MemoryCache,
	)
	i3, err := Create[string](c, "test23",
		DefaultTTL(5*time.Minute),
		Loader(loadCacheValueById),
		MemoryCache,
	)
	i2, err := Create[string](c, "testRedis",
		DefaultTTL(5*time.Minute),
		Loader(loadCacheValueById),
		RedisCache,
	)
	require.NoError(t, err)
	var wg sync.WaitGroup

	for _, key := range testKeys {
		wg.Add(1)
		go func(key string) {
			val, err := i.Get(context.TODO(), key)
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("value-%s", key), val)
			firstVal, err := i.Get(context.TODO(), testKeys[0])
			i.Set(context.TODO(), testKeys[0], fmt.Sprintf("value-%s", testKeys[0]))
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("value-%s", testKeys[0]), firstVal)
			i.Delete(context.TODO(), testKeys[0])
			val, err = i2.Get(context.TODO(), key)
			i.Set(context.TODO(), testKeys[0], fmt.Sprintf("value-%s", testKeys[0]))
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("value-%s", key), val)
			val, err = i3.Get(context.TODO(), key)
			i3.Set(context.TODO(), testKeys[0], fmt.Sprintf("value-%s", testKeys[0]))
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("value-%s", key), val)
			wg.Done()
		}(key)
		wg.Add(1)
		go func(key string) {
			i.Set(context.TODO(), testKeys[0], fmt.Sprintf("value-%s", testKeys[0]))
			assert.NoError(t, err)
			val, err := i2.Get(context.TODO(), testKeys[0])
			assert.Equal(t, fmt.Sprintf("value-%s", testKeys[0]), val)
			i.Set(context.TODO(), testKeys[0], fmt.Sprintf("value-%s", testKeys[0]))
			assert.NoError(t, err)
			i3.Set(context.TODO(), testKeys[0], fmt.Sprintf("value-%s", testKeys[0]))
			assert.NoError(t, err)
			val, err = i3.Get(context.TODO(), testKeys[0])
			assert.Equal(t, fmt.Sprintf("value-%s", testKeys[0]), val)
			i3.Set(context.TODO(), testKeys[0], fmt.Sprintf("value-%s", testKeys[0]))
			assert.NoError(t, err)
			wg.Done()
		}(key)
	}
	wg.Wait()

}

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
