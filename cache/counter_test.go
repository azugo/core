package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-quicktest/qt"
)

func newMemoryCounter(t *testing.T, opts ...Option) Counter {
	t.Helper()

	c := New(append([]Option{MemoryCache}, opts...)...)
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))
	t.Cleanup(c.Close)

	counter, err := CreateCounter(c, "test")
	qt.Assert(t, qt.IsNil(err))

	return counter
}

func TestMemoryCounterIncrement(t *testing.T) {
	counter := newMemoryCounter(t)

	count, err := counter.Increment(context.TODO(), "key", 1)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(count, int64(1)))

	count, err = counter.Increment(context.TODO(), "key", 4)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(count, int64(5)))

	// A negative delta counts down.
	count, err = counter.Increment(context.TODO(), "key", -2)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(count, int64(3)))

	// The embedded Instance sees the same value.
	val, err := counter.Get(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, int64(3)))
}

func TestMemoryCounterKeepsFixedWindow(t *testing.T) {
	counter := newMemoryCounter(t)

	_, err := counter.Increment(context.TODO(), "key", 1, TTL[int64](50*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))

	// Incrementing again must not restart the window.
	time.Sleep(30 * time.Millisecond)

	count, err := counter.Increment(context.TODO(), "key", 1)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(count, int64(2)))

	time.Sleep(40 * time.Millisecond)

	count, err = counter.Increment(context.TODO(), "key", 1)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(count, int64(1)), qt.Commentf("the window expired, so the counter restarts"))
}

func TestMemoryCounterLosesNoConcurrentIncrement(t *testing.T) {
	counter := newMemoryCounter(t)

	const racers = 32

	var wg sync.WaitGroup

	wg.Add(racers)

	for range racers {
		go func() {
			defer wg.Done()

			_, err := counter.Increment(context.TODO(), "key", 1)
			qt.Check(t, qt.IsNil(err))
		}()
	}

	wg.Wait()

	count, err := counter.Get(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(count, int64(racers)))
}

func TestRedisCounterIncrement(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skip("REDIS_CONNSTR is not set")
	}

	c := New(RedisCache, KeyPrefix("prefix"), ConnectionString(cs))
	qt.Assert(t, qt.IsNil(c.Start(context.TODO())))

	defer c.Close()

	counter, err := CreateCounter(c, "test")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsNil(counter.Delete(context.TODO(), "count")))

	count, err := counter.Increment(context.TODO(), "count", 1, TTL[int64](time.Minute))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(count, int64(1)))

	count, err = counter.Increment(context.TODO(), "count", 4)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(count, int64(5)))

	// The embedded Instance reads the same value, so the stored form is shared.
	val, err := counter.Get(context.TODO(), "count")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, int64(5)))

	qt.Assert(t, qt.IsNil(counter.Delete(context.TODO(), "count")))
}
