package ratelimit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"azugo.io/core/cache"

	"github.com/go-quicktest/qt"
)

func redisCacheTest(t *testing.T) *cache.Cache {
	t.Helper()

	cs := os.Getenv("REDIS_CONNSTR")
	if cs == "" {
		t.Skip("REDIS_CONNSTR is not set")
	}

	c := cache.New(cache.RedisCache, cache.KeyPrefix("ratelimit-test"), cache.ConnectionString(cs))
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	t.Cleanup(c.Close)

	return c
}

// uniqueName isolates test state from previous runs since limiter keys
// persist in Redis for the duration of the window.
func uniqueName(name string) string {
	return fmt.Sprintf("%s-%d", name, time.Now().UnixNano())
}

func limiterRedisKey(c *cache.Cache, name, key string) string {
	prefix := c.ConfiguredKeyPrefix()
	if prefix != "" {
		prefix += ":"
	}

	return prefix + strings.Join([]string{"ratelimit", name, key}, ":")
}

func TestRedisFixedWindow(t *testing.T) {
	c := redisCacheTest(t)

	l, err := NewFixedWindow(c, uniqueName("fw"), 3, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	for i := range 3 {
		res, err := l.Allow(ctx, "key")
		qt.Assert(t, qt.IsNil(err))
		qt.Check(t, qt.IsTrue(res.Allowed), qt.Commentf("event %d", i))
		qt.Check(t, qt.Equals(res.Remaining, 2-i))
	}

	res, err := l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))
	qt.Check(t, qt.IsTrue(res.RetryAfter > 0))

	// Peek does not consume.
	res, err = l.Peek(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))
	qt.Check(t, qt.Equals(res.Remaining, 0))

	// Reset clears the counter.
	err = l.Reset(ctx, "key")
	qt.Assert(t, qt.IsNil(err))

	res, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
}

func TestRedisFixedWindowSharedState(t *testing.T) {
	c := redisCacheTest(t)
	name := uniqueName("fw-shared")

	// Two limiter instances simulate two service instances sharing the
	// same Redis backend.
	l1, err := NewFixedWindow(c, name, 2, time.Minute)
	qt.Assert(t, qt.IsNil(err))
	l2, err := NewFixedWindow(c, name, 2, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	res, err := l1.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))

	res, err = l2.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
	qt.Check(t, qt.Equals(res.Remaining, 0))

	res, err = l1.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))
}

func TestRedisFixedWindowExpiry(t *testing.T) {
	c := redisCacheTest(t)

	l, err := NewFixedWindow(c, uniqueName("fw-expiry"), 1, 200*time.Millisecond)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	res, err := l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))

	res, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))

	time.Sleep(250 * time.Millisecond)

	res, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
}

func TestRedisTokenBucket(t *testing.T) {
	c := redisCacheTest(t)

	l, err := NewTokenBucket(c, uniqueName("tb"), 10, 3)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	for i := range 3 {
		res, err := l.Allow(ctx, "key")
		qt.Assert(t, qt.IsNil(err))
		qt.Check(t, qt.IsTrue(res.Allowed), qt.Commentf("burst event %d", i))
	}

	res, err := l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))
	qt.Check(t, qt.IsTrue(res.RetryAfter > 0))
	qt.Check(t, qt.IsTrue(res.RetryAfter <= 150*time.Millisecond))

	// One token replenishes after the emission interval.
	time.Sleep(150 * time.Millisecond)

	res, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
}

func TestRedisTokenBucketWait(t *testing.T) {
	c := redisCacheTest(t)

	l, err := NewTokenBucket(c, uniqueName("tb-wait"), 50, 1)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	_, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))

	err = l.Wait(ctx, "key")
	qt.Check(t, qt.IsNil(err))

	// Drained bucket recovers in 20ms which exceeds the wait limit.
	_, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))

	lw, err := NewTokenBucket(c, uniqueName("tb-wait-limit"), 1, 1, WaitLimit(50*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))

	_, err = lw.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))

	err = lw.Wait(ctx, "key")
	qt.Check(t, qt.ErrorIs(err, ErrWaitLimitExceeded))
}

func TestRedisFixedWindowAllowNMoreThanLimit(t *testing.T) {
	c := redisCacheTest(t)

	l, err := NewFixedWindow(c, uniqueName("fw-too-many"), 2, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	res, err := l.AllowN(context.TODO(), "key", 3)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))
	qt.Check(t, qt.Equals(res.RetryAfter, time.Duration(0)))
}

func TestRedisTokenBucketAllowNMoreThanBurst(t *testing.T) {
	c := redisCacheTest(t)

	l, err := NewTokenBucket(c, uniqueName("tb-too-many"), 20, 2)
	qt.Assert(t, qt.IsNil(err))

	res, err := l.AllowN(context.TODO(), "key", 3)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))
	qt.Check(t, qt.Equals(res.RetryAfter, time.Duration(0)))
}

func TestRedisFixedWindowPreservesCounterWithoutTTL(t *testing.T) {
	c := redisCacheTest(t)
	name := uniqueName("fw-persist")

	l, err := NewFixedWindow(c, name, 3, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	err = c.Connection().Set(ctx, limiterRedisKey(c, name, "key"), "3", 0).Err()
	qt.Assert(t, qt.IsNil(err))

	res, err := l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))
	qt.Check(t, qt.Equals(res.Remaining, 0))
	qt.Check(t, qt.IsTrue(res.RetryAfter > 0))
}

func TestRedisScriptsHandleNonNumericState(t *testing.T) {
	c := redisCacheTest(t)
	ctx := context.TODO()

	fwName := uniqueName("fw-bad")
	fw, err := NewFixedWindow(c, fwName, 3, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	err = c.Connection().Set(ctx, limiterRedisKey(c, fwName, "key"), "bad", 0).Err()
	qt.Assert(t, qt.IsNil(err))

	res, err := fw.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
	qt.Check(t, qt.Equals(res.Remaining, 2))

	tbName := uniqueName("tb-bad")
	tb, err := NewTokenBucket(c, tbName, 10, 2)
	qt.Assert(t, qt.IsNil(err))

	err = c.Connection().Set(ctx, limiterRedisKey(c, tbName, "key"), "bad", 0).Err()
	qt.Assert(t, qt.IsNil(err))

	res, err = tb.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
}

func TestRedisSemaphore(t *testing.T) {
	c := redisCacheTest(t)
	name := uniqueName("sem")

	// Two semaphore instances simulate two service instances sharing the
	// same Redis backend.
	s1, err := NewSemaphore(c, name, 2, PollInterval(5*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))
	s2, err := NewSemaphore(c, name, 2, PollInterval(5*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	rel1, ok, err := s1.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsTrue(ok))

	rel2, ok, err := s2.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsTrue(ok))

	// All slots are held across both instances.
	_, ok, err = s1.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(ok))

	// A release on one instance frees the slot for the other.
	rel1()

	rel3, ok, err := s2.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))

	rel2()
	rel3()
}

func TestRedisSemaphoreAcquireWaitsForRelease(t *testing.T) {
	c := redisCacheTest(t)

	s, err := NewSemaphore(c, uniqueName("sem-wait"), 1, PollInterval(5*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	rel, err := s.Acquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))

	go func() {
		time.Sleep(30 * time.Millisecond)
		rel()
	}()

	start := time.Now()
	rel2, err := s.Acquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(time.Since(start) >= 25*time.Millisecond))
	rel2()
}

func TestRedisSemaphoreLeaseExpires(t *testing.T) {
	c := redisCacheTest(t)

	s, err := NewSemaphore(c, uniqueName("sem-lease"), 1, LeaseTTL(50*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	// Slot is taken and never released, e.g. the holder crashed.
	_, ok, err := s.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsTrue(ok))

	_, ok, err = s.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(ok))

	time.Sleep(70 * time.Millisecond)

	rel, ok, err := s.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))
	rel()
}
