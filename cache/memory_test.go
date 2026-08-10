package cache

import (
	"context"
	"testing"
	"time"

	"azugo.io/core/instrumenter"

	"github.com/go-quicktest/qt"
)

func TestMemoryCacheGetSet(t *testing.T) {
	c := New(MemoryCache)
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key", "value")
	qt.Check(t, qt.IsNil(err))

	// Even if a Set gets applied, it might take a few milliseconds after the call has returned to the user.
	// In database terms, it is an eventual consistency model.
	time.Sleep(10 * time.Millisecond)
	val, err := i.Get(context.TODO(), "key")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "value"))
}

func TestMemoryCachePop(t *testing.T) {
	c := New(MemoryCache)
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key", "value")
	qt.Check(t, qt.IsNil(err))
	// Even if a Set gets applied, it might take a few milliseconds after the call has returned to the user.
	// In database terms, it is an eventual consistency model.
	time.Sleep(10 * time.Millisecond)
	val, err := i.Pop(context.TODO(), "key")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "value"))

	val, err = i.Pop(context.TODO(), "key")
	qt.Check(t, qt.IsNotNil(err))
	qt.Check(t, qt.Equals(val, ""))
}

func TestMemoryCacheDelete(t *testing.T) {
	c := New(MemoryCache)
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key", "value")
	qt.Check(t, qt.IsNil(err))

	err = i.Delete(context.TODO(), "key")
	qt.Check(t, qt.IsNil(err))

	val, err := i.Get(context.TODO(), "key")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, ""))
}

func TestMemoryCacheExpire(t *testing.T) {
	c := New(MemoryCache)
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test", DefaultTTL(100*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key", "value")
	qt.Check(t, qt.IsNil(err))

	time.Sleep(150 * time.Millisecond)

	val, err := i.Get(context.TODO(), "key")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, ""))
}

func TestMemoryCacheItemExpire(t *testing.T) {
	c := New(MemoryCache)
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key", "value", TTL[string](100*time.Millisecond))
	qt.Check(t, qt.IsNil(err))

	time.Sleep(150 * time.Millisecond)

	val, err := i.Get(context.TODO(), "key")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, ""))
}

func TestMemoryCacheGetHit(t *testing.T) {
	for _, serialize := range []bool{true, false} {
		hits := 0
		instr := func(_ context.Context, op string, args ...any) func(error) {
			key, ok := InstrGetHit(op, args...)
			if backend, bok := InstrBackend(args...); ok && bok && key == "test:key" && backend == MemoryCache {
				hits++
			}

			return instrumenter.NullFinish
		}

		c := New(MemoryCache, Serialize(serialize), Instrumenter(instr))
		err := c.Start(context.TODO())
		qt.Assert(t, qt.IsNil(err))

		i, err := Create[string](c, "test")
		qt.Assert(t, qt.IsNil(err))

		err = i.Set(context.TODO(), "key", "value")
		qt.Check(t, qt.IsNil(err))

		time.Sleep(10 * time.Millisecond)

		val, err := i.Get(context.TODO(), "key")
		qt.Check(t, qt.IsNil(err))
		qt.Check(t, qt.Equals(val, "value"))
		qt.Check(t, qt.Equals(hits, 1))

		// Missing key is not a hit.
		_, err = i.Get(context.TODO(), "missing")
		qt.Check(t, qt.IsNil(err))
		qt.Check(t, qt.Equals(hits, 1))

		c.Close()
	}
}
