package cache

import (
	"context"
	"strconv"
	"sync"
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
	qt.Assert(t, qt.IsNil(i.Sync(context.TODO())))
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
	qt.Assert(t, qt.IsNil(i.Sync(context.TODO())))
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
	qt.Assert(t, qt.IsNil(i.Sync(context.TODO())))
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
			backend, bok := InstrBackend(args...)
			instance, iok := InstrInstance(args...)

			if ok && bok && iok && key == "test:key" && backend == MemoryCache && instance == "test" {
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
		qt.Assert(t, qt.IsNil(i.Sync(context.TODO())))

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

func TestMemoryCacheSyncMakesWritesVisible(t *testing.T) {
	c := New(MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))

	i, err := Create[string](c, "sync")
	qt.Assert(t, qt.IsNil(err))

	for n := range 1000 {
		key := "k" + strconv.Itoa(n)
		qt.Assert(t, qt.IsNil(i.Set(context.TODO(), key, "v")))
		qt.Assert(t, qt.IsNil(i.Sync(context.TODO())))

		v, err := i.Get(context.TODO(), key)
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.Equals(v, "v"))
	}
}

func TestMemoryCacheAdd(t *testing.T) {
	c := New(MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))

	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	added, err := i.Add(context.TODO(), "key", "first")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(added))

	added, err = i.Add(context.TODO(), "key", "second")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(added))

	val, err := i.Get(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "first"))
}

func TestMemoryCacheAddHasOneWinner(t *testing.T) {
	c := New(MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))

	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	const racers = 16

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		wins int
	)

	wg.Add(racers)

	for n := range racers {
		go func() {
			defer wg.Done()

			added, err := i.Add(context.TODO(), "key", strconv.Itoa(n))
			qt.Check(t, qt.IsNil(err))

			if added {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	qt.Check(t, qt.Equals(wins, 1))
}

func TestMemoryCacheSwap(t *testing.T) {
	c := New(MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))

	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	prev, found, err := i.Swap(context.TODO(), "key", "first")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(found))
	qt.Check(t, qt.Equals(prev, ""))

	prev, found, err = i.Swap(context.TODO(), "key", "second")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(found))
	qt.Check(t, qt.Equals(prev, "first"))

	val, err := i.Get(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "second"))
}

func TestMemoryCacheSwapHandsOutEachPredecessorOnce(t *testing.T) {
	c := New(MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))

	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	const racers = 16

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		seen = make(map[string]int, racers)
	)

	wg.Add(racers)

	for n := range racers {
		go func() {
			defer wg.Done()

			prev, _, err := i.Swap(context.TODO(), "key", strconv.Itoa(n))
			qt.Check(t, qt.IsNil(err))

			mu.Lock()
			seen[prev]++
			mu.Unlock()
		}()
	}

	wg.Wait()

	// No predecessor is observed twice, so no update was lost.
	for _, count := range seen {
		qt.Check(t, qt.Equals(count, 1))
	}
}

func TestMemoryCacheExpireExtendsWithoutChangingValue(t *testing.T) {
	c := New(MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))

	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	// A missing key reports absent.
	applied, err := i.Expire(context.TODO(), "key", time.Minute)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(applied))

	qt.Assert(t, qt.IsNil(i.Set(context.TODO(), "key", "value", TTL[string](30*time.Millisecond))))
	qt.Assert(t, qt.IsNil(i.Sync(context.TODO())))

	applied, err = i.Expire(context.TODO(), "key", time.Minute)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(applied))

	// The original lifetime has passed, but the extended entry is still there, unchanged.
	time.Sleep(60 * time.Millisecond)

	val, err := i.Get(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "value"))

	_, err = i.Expire(context.TODO(), "key", 0)
	qt.Check(t, qt.IsNotNil(err))
}

func TestMemoryCacheTTL(t *testing.T) {
	c := New(MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))

	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	// A missing key is absent, not merely expiry-less.
	ttl, found, err := i.TTL(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(found))
	qt.Check(t, qt.Equals(ttl, time.Duration(0)))

	qt.Assert(t, qt.IsNil(i.Set(context.TODO(), "key", "value", TTL[string](time.Minute))))
	qt.Assert(t, qt.IsNil(i.Sync(context.TODO())))

	ttl, found, err = i.TTL(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(found))
	qt.Check(t, qt.IsTrue(ttl > 59*time.Second && ttl <= time.Minute), qt.Commentf("got %s", ttl))

	// Expire re-times it, and the read reflects that.
	applied, err := i.Expire(context.TODO(), "key", time.Hour)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(applied))

	ttl, _, err = i.TTL(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ttl > 59*time.Minute), qt.Commentf("got %s", ttl))
}

func TestMemoryCacheAddSeesPrecedingSet(t *testing.T) {
	c := New(MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	for n := range 1000 {
		key := strconv.Itoa(n)

		qt.Assert(t, qt.IsNil(i.Set(context.TODO(), key, "set")))

		added, err := i.Add(context.TODO(), key, "add")
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.IsFalse(added))

		val, err := i.Get(context.TODO(), key)
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.Equals(val, "set"))
	}
}

func TestMemoryCacheExpireDoesNotRestoreDeletedKey(t *testing.T) {
	c := New(MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	for n := range 1000 {
		key := strconv.Itoa(n)

		qt.Assert(t, qt.IsNil(i.Set(context.TODO(), key, "value")))
		qt.Assert(t, qt.IsNil(i.Sync(context.TODO())))

		var wg sync.WaitGroup

		wg.Go(func() {
			_ = i.Delete(context.TODO(), key)
		})
		wg.Go(func() {
			_, _ = i.Expire(context.TODO(), key, time.Minute)
		})
		wg.Wait()

		qt.Assert(t, qt.IsNil(i.Sync(context.TODO())))

		_, found, err := i.TTL(context.TODO(), key)
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.IsFalse(found))
	}
}

func TestMemoryCacheReplace(t *testing.T) {
	c := New(MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	replaced, err := i.Replace(context.TODO(), "key", "value")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(replaced))

	_, found, err := i.TTL(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(found))

	qt.Assert(t, qt.IsNil(i.Set(context.TODO(), "key", "first")))

	replaced, err = i.Replace(context.TODO(), "key", "second", TTL[string](time.Minute))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(replaced))

	val, err := i.Get(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "second"))

	ttl, found, err := i.TTL(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(found))
	qt.Check(t, qt.IsTrue(ttl > 0 && ttl <= time.Minute))
}

func TestMemoryCacheReplaceDoesNotRestoreDeletedKey(t *testing.T) {
	c := New(MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	for n := range 1000 {
		key := strconv.Itoa(n)

		qt.Assert(t, qt.IsNil(i.Set(context.TODO(), key, "value")))

		var wg sync.WaitGroup

		wg.Go(func() {
			_ = i.Delete(context.TODO(), key)
		})
		wg.Go(func() {
			_, _ = i.Replace(context.TODO(), key, "renewed")
		})
		wg.Wait()

		qt.Assert(t, qt.IsNil(i.Sync(context.TODO())))

		_, found, err := i.TTL(context.TODO(), key)
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.IsFalse(found))
	}
}

func TestMemoryCacheSensitiveKeys(t *testing.T) {
	var (
		mu   sync.Mutex
		keys []string
	)

	instr := func(_ context.Context, op string, args ...any) func(error) {
		if op == InstrumentationGet || op == InstrumentationSet || op == InstrumentationDelete || op == InstrumentationLoader {
			key, _ := args[0].(string)
			instance, _ := InstrInstance(args...)

			mu.Lock()
			keys = append(keys, op+"|"+key+"|"+instance)
			mu.Unlock()
		}

		return instrumenter.NullFinish
	}

	c := New(MemoryCache, Instrumenter(instr))
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))
	defer c.Close()

	loader := Loader(func(context.Context, string) (any, error) {
		return "loaded", nil
	})

	secret, err := Create[string](c, "sessions", SensitiveKeys(), loader)
	qt.Assert(t, qt.IsNil(err))

	plain, err := Create[string](c, "plain")
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.IsNil(secret.Set(context.TODO(), "sid", "value")))
	qt.Assert(t, qt.IsNil(secret.Delete(context.TODO(), "sid")))

	val, err := secret.Get(context.TODO(), "missing")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "loaded"))

	qt.Assert(t, qt.IsNil(plain.Set(context.TODO(), "key", "value")))

	mu.Lock()
	defer mu.Unlock()

	qt.Check(t, qt.DeepEquals(keys, []string{
		InstrumentationSet + "|sessions:<redacted>|sessions",
		InstrumentationDelete + "|sessions:<redacted>|sessions",
		InstrumentationGet + "|sessions:<redacted>|sessions",
		InstrumentationLoader + "|sessions:<redacted>|",
		InstrumentationSet + "|plain:key|plain",
	}))
}
