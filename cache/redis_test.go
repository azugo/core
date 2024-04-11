package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-quicktest/qt"
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
	c := New(RedisCache, KeyPrefix("prefix"), ConnectionString(cs))
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key1", "value")
	qt.Check(t, qt.IsNil(err))

	val, err := i.Get(context.TODO(), "key1")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "value"))
}

func TestRedisCachePop(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skipped()
		return
	}
	c := New(RedisCache, ConnectionString(cs))
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key2", "value")
	qt.Check(t, qt.IsNil(err))

	val, err := i.Pop(context.TODO(), "key2")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "value"))

	val, err = i.Pop(context.TODO(), "key2")
	qt.Check(t, qt.IsNotNil(err))
	qt.Check(t, qt.Equals(val, ""))
}

func TestRedisCacheDelete(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skipped()
		return
	}
	c := New(RedisCache, ConnectionString(cs))
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key3", "value")
	qt.Check(t, qt.IsNil(err))

	err = i.Delete(context.TODO(), "key3")
	qt.Check(t, qt.IsNil(err))

	val, err := i.Get(context.TODO(), "key3")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, ""))
}

func TestRedisCacheExpire(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skipped()
		return
	}
	c := New(RedisCache, ConnectionString(cs))
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test", DefaultTTL(100*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key4", "value")
	qt.Check(t, qt.IsNil(err))

	time.Sleep(150 * time.Millisecond)

	val, err := i.Get(context.TODO(), "key4")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, ""))
}
