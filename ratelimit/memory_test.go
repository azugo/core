package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"azugo.io/core/cache"

	"github.com/go-quicktest/qt"
)

func memCache() *cache.Cache {
	return cache.New()
}

func TestNewValidation(t *testing.T) {
	c := memCache()

	_, err := NewFixedWindow(c, "", 5, time.Minute)
	qt.Check(t, qt.ErrorMatches(err, "rate limiter name is required"))

	_, err = NewFixedWindow(c, "test", 0, time.Minute)
	qt.Check(t, qt.ErrorMatches(err, "fixed window limit must be positive"))

	_, err = NewFixedWindow(c, "test", 5, 0)
	qt.Check(t, qt.ErrorMatches(err, "fixed window duration must be positive"))

	_, err = NewTokenBucket(c, "test", 0, 1)
	qt.Check(t, qt.ErrorMatches(err, "token bucket rate must be positive"))

	_, err = NewTokenBucket(c, "test", 1, 0)
	qt.Check(t, qt.ErrorMatches(err, "token bucket burst must be positive"))

	_, err = NewSemaphore(c, "test", 0)
	qt.Check(t, qt.ErrorMatches(err, "semaphore slot count must be positive"))

	_, err = NewSemaphore(c, "", 1)
	qt.Check(t, qt.ErrorMatches(err, "semaphore name is required"))

	_, err = NewSemaphore(c, "test", 1, LeaseTTL(0))
	qt.Check(t, qt.ErrorMatches(err, "lease TTL must be positive"))
}

func TestNewRedisCacheNotStarted(t *testing.T) {
	c := cache.New(cache.RedisCache, cache.ConnectionString("redis://localhost:6379/0"))

	l, err := NewFixedWindow(c, "test", 5, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	_, err = l.Allow(context.TODO(), "key")
	qt.Check(t, qt.ErrorIs(err, cache.ErrCacheUnavailable))

	s, err := NewSemaphore(c, "test", 1)
	qt.Assert(t, qt.IsNil(err))

	_, _, err = s.TryAcquire(context.TODO(), "key")
	qt.Check(t, qt.ErrorIs(err, cache.ErrCacheUnavailable))
}

func TestFixedWindowAllow(t *testing.T) {
	l, err := NewFixedWindow(memCache(), "test", 3, 200*time.Millisecond)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	for i := range 3 {
		res, err := l.Allow(ctx, "key")
		qt.Assert(t, qt.IsNil(err))
		qt.Check(t, qt.IsTrue(res.Allowed))
		qt.Check(t, qt.Equals(res.Remaining, 2-i))
	}

	res, err := l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))
	qt.Check(t, qt.Equals(res.Remaining, 0))
	qt.Check(t, qt.IsTrue(res.RetryAfter > 0))

	// Other keys are not affected.
	res, err = l.Allow(ctx, "other")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))

	// The window expires and attempts are allowed again.
	time.Sleep(250 * time.Millisecond)

	res, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
}

func TestFixedWindowAllowN(t *testing.T) {
	l, err := NewFixedWindow(memCache(), "test", 5, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	res, err := l.AllowN(ctx, "key", 3)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
	qt.Check(t, qt.Equals(res.Remaining, 2))

	res, err = l.AllowN(ctx, "key", 3)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))
	qt.Check(t, qt.Equals(res.Remaining, 2))

	// More than the limit can never be allowed.
	res, err = l.AllowN(ctx, "fresh", 6)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))

	_, err = l.AllowN(ctx, "key", 0)
	qt.Check(t, qt.ErrorMatches(err, "event count must be positive"))
}

func TestFixedWindowPeek(t *testing.T) {
	l, err := NewFixedWindow(memCache(), "test", 2, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	// Peek does not consume events.
	for range 5 {
		res, err := l.Peek(ctx, "key")
		qt.Assert(t, qt.IsNil(err))
		qt.Check(t, qt.IsTrue(res.Allowed))
		qt.Check(t, qt.Equals(res.Remaining, 2))
	}

	_, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	_, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))

	res, err := l.Peek(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))
	qt.Check(t, qt.Equals(res.Remaining, 0))
	qt.Check(t, qt.IsTrue(res.RetryAfter > 0))
}

func TestFixedWindowReset(t *testing.T) {
	l, err := NewFixedWindow(memCache(), "test", 1, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	res, err := l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))

	res, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))

	err = l.Reset(ctx, "key")
	qt.Assert(t, qt.IsNil(err))

	res, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
}

func TestTokenBucketBurst(t *testing.T) {
	// 10 tokens per second, burst of 3.
	l, err := NewTokenBucket(memCache(), "test", 10, 3)
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
	qt.Check(t, qt.IsTrue(res.RetryAfter <= 100*time.Millisecond))

	// One token replenishes after the emission interval.
	time.Sleep(120 * time.Millisecond)

	res, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
}

func TestTokenBucketPeek(t *testing.T) {
	l, err := NewTokenBucket(memCache(), "test", 10, 2)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	res, err := l.Peek(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
	qt.Check(t, qt.Equals(res.Remaining, 2))

	_, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	_, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))

	res, err = l.Peek(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))
	qt.Check(t, qt.Equals(res.Remaining, 0))
}

func TestWait(t *testing.T) {
	// 50 tokens per second so a drained bucket recovers in 20ms.
	l, err := NewTokenBucket(memCache(), "test", 50, 1)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	res, err := l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))

	start := time.Now()
	err = l.Wait(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(time.Since(start) >= 10*time.Millisecond))
}

func TestWaitLimitExceeded(t *testing.T) {
	// Bucket recovers in 1s but waiting is limited to 50ms.
	l, err := NewTokenBucket(memCache(), "test", 1, 1, WaitLimit(50*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	_, err = l.Allow(ctx, "key")
	qt.Assert(t, qt.IsNil(err))

	start := time.Now()
	err = l.Wait(ctx, "key")
	qt.Check(t, qt.ErrorIs(err, ErrWaitLimitExceeded))
	// Fails fast instead of waiting out the limit.
	qt.Check(t, qt.IsTrue(time.Since(start) < 50*time.Millisecond))
}

func TestWaitContextCancel(t *testing.T) {
	l, err := NewTokenBucket(memCache(), "test", 1, 1)
	qt.Assert(t, qt.IsNil(err))

	_, err = l.Allow(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))

	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err = l.Wait(ctx, "key")
	qt.Check(t, qt.ErrorIs(err, context.Canceled))
}

func TestWaitContextDeadline(t *testing.T) {
	l, err := NewTokenBucket(memCache(), "test", 1, 1)
	qt.Assert(t, qt.IsNil(err))

	_, err = l.Allow(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))

	ctx, cancel := context.WithTimeout(context.TODO(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err = l.Wait(ctx, "key")
	qt.Check(t, qt.ErrorIs(err, ErrWaitLimitExceeded))
	qt.Check(t, qt.IsTrue(time.Since(start) < 50*time.Millisecond))
}

func TestSemaphoreTryAcquire(t *testing.T) {
	s, err := NewSemaphore(memCache(), "test", 2)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	rel1, ok, err := s.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsTrue(ok))

	rel2, ok, err := s.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsTrue(ok))

	_, ok, err = s.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(ok))

	// Other keys have their own slots.
	relOther, ok, err := s.TryAcquire(ctx, "other")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))
	relOther()

	rel1()
	// Releasing twice is a no-op.
	rel1()

	rel3, ok, err := s.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))

	rel2()
	rel3()
}

func TestSemaphoreAcquireWaitsForRelease(t *testing.T) {
	s, err := NewSemaphore(memCache(), "test", 1, PollInterval(5*time.Millisecond))
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
	qt.Check(t, qt.IsTrue(time.Since(start) >= 30*time.Millisecond))
	rel2()
}

func TestSemaphoreAcquireWaitLimit(t *testing.T) {
	s, err := NewSemaphore(memCache(), "test", 1,
		WaitLimit(50*time.Millisecond), PollInterval(5*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	rel, err := s.Acquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	defer rel()

	err = func() error {
		_, err := s.Acquire(ctx, "key")
		return err
	}()
	qt.Check(t, qt.ErrorIs(err, ErrWaitLimitExceeded))
}

func TestSemaphoreLeaseExpires(t *testing.T) {
	s, err := NewSemaphore(memCache(), "test", 1, LeaseTTL(30*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	// Slot is taken and never released, e.g. the holder crashed.
	_, ok, err := s.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsTrue(ok))

	_, ok, err = s.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(ok))

	time.Sleep(40 * time.Millisecond)

	rel, ok, err := s.TryAcquire(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))
	rel()
}

func TestFailOpen(t *testing.T) {
	opt, err := newFixedWindowOptions(1, time.Minute, FailOpen(true))
	qt.Assert(t, qt.IsNil(err))
	l := &limiter{backend: failingBackend{}, opt: opt}

	res, err := l.Allow(context.TODO(), "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))

	err = l.Wait(context.TODO(), "key")
	qt.Check(t, qt.IsNil(err))

	// Fail closed by default.
	opt, err = newFixedWindowOptions(1, time.Minute)
	qt.Assert(t, qt.IsNil(err))
	l = &limiter{backend: failingBackend{}, opt: opt}

	_, err = l.Allow(context.TODO(), "key")
	qt.Check(t, qt.ErrorMatches(err, "backend unavailable"))
}

func BenchmarkMemoryFixedWindowAllow(b *testing.B) {
	l, err := NewFixedWindow(memCache(), "bench", math.MaxInt32, time.Hour)
	qt.Assert(b, qt.IsNil(err))

	ctx := context.TODO()

	b.ReportAllocs()

	for b.Loop() {
		if _, err := l.Allow(ctx, "key"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryTokenBucketAllow(b *testing.B) {
	l, err := NewTokenBucket(memCache(), "bench", math.MaxInt32, math.MaxInt32)
	qt.Assert(b, qt.IsNil(err))

	ctx := context.TODO()

	b.ReportAllocs()

	for b.Loop() {
		if _, err := l.Allow(ctx, "key"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemorySemaphore(b *testing.B) {
	s, err := NewSemaphore(memCache(), "bench", 1)
	qt.Assert(b, qt.IsNil(err))

	ctx := context.TODO()

	b.ReportAllocs()

	for b.Loop() {
		release, ok, err := s.TryAcquire(ctx, "key")
		if err != nil || !ok {
			b.Fatal(err, ok)
		}

		release()
	}
}

type failingBackend struct{}

func (failingBackend) allowN(_ context.Context, _ string, _ int, _ bool, _ int) (Result, error) {
	return Result{}, errors.New("backend unavailable")
}

func (failingBackend) reset(_ context.Context, _ string) error {
	return errors.New("backend unavailable")
}

func TestAllowNOverflow(t *testing.T) {
	ctx := context.TODO()

	// A huge n must not wrap the window counter sum and corrupt state.
	l, err := NewFixedWindow(memCache(), "test", 5, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	res, err := l.AllowN(ctx, "key", math.MaxInt)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))

	res, err = l.AllowN(ctx, "key", 5)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))

	// A huge n must not overflow the token bucket arrival time.
	l, err = NewTokenBucket(memCache(), "test", 10, 3)
	qt.Assert(t, qt.IsNil(err))

	res, err = l.AllowN(ctx, "key", math.MaxInt)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))

	res, err = l.AllowN(ctx, "key", 3)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
}

func TestMemoryLimiterMaxKeys(t *testing.T) {
	ctx := context.TODO()

	l, err := NewFixedWindow(memCache(), "test", 5, time.Minute, MaxKeys(3))
	qt.Assert(t, qt.IsNil(err))

	for i := range 10 {
		_, err = l.Allow(ctx, fmt.Sprintf("key%d", i))
		qt.Assert(t, qt.IsNil(err))
	}

	ml, ok := l.(*limiter).backend.(*memoryLimiter)
	qt.Assert(t, qt.IsTrue(ok))
	qt.Check(t, qt.IsTrue(len(ml.windows) <= 3), qt.Commentf("windows: %d", len(ml.windows)))

	l, err = NewTokenBucket(memCache(), "test", 10, 3, MaxKeys(3))
	qt.Assert(t, qt.IsNil(err))

	for i := range 10 {
		_, err = l.Allow(ctx, fmt.Sprintf("key%d", i))
		qt.Assert(t, qt.IsNil(err))
	}

	ml, ok = l.(*limiter).backend.(*memoryLimiter)
	qt.Assert(t, qt.IsTrue(ok))
	qt.Check(t, qt.IsTrue(len(ml.tats) <= 3), qt.Commentf("tats: %d", len(ml.tats)))

	_, err = NewFixedWindow(memCache(), "test", 5, time.Minute, MaxKeys(0))
	qt.Check(t, qt.ErrorMatches(err, "max keys must be positive"))
}

func TestMemorySemaphoreMaxKeys(t *testing.T) {
	ctx := context.TODO()

	s, err := NewSemaphore(memCache(), "test", 1, MaxKeys(3))
	qt.Assert(t, qt.IsNil(err))

	for i := range 10 {
		_, ok, err := s.TryAcquire(ctx, fmt.Sprintf("key%d", i))
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.IsTrue(ok))
	}

	ms, ok := s.(*semaphore).backend.(*memorySemaphore)
	qt.Assert(t, qt.IsTrue(ok))
	qt.Check(t, qt.IsTrue(len(ms.leases) <= 3), qt.Commentf("leases: %d", len(ms.leases)))
}

func TestLongKeysNormalized(t *testing.T) {
	ctx := context.TODO()

	l, err := NewFixedWindow(memCache(), "test", 1, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	longA := strings.Repeat("a", 100_000)
	longB := strings.Repeat("b", 100_000)

	res, err := l.Allow(ctx, longA)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))

	// Same long key maps to the same counter.
	res, err = l.Allow(ctx, longA)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(res.Allowed))

	// Different long keys map to different counters.
	res, err = l.Allow(ctx, longB)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))

	// Only the hash is stored, not the full key.
	ml, ok := l.(*limiter).backend.(*memoryLimiter)
	qt.Assert(t, qt.IsTrue(ok))

	for k := range ml.windows {
		qt.Check(t, qt.IsTrue(len(k) <= maxKeyLength), qt.Commentf("stored key length: %d", len(k)))
	}

	// Reset reaches the same normalized key.
	err = l.Reset(ctx, longA)
	qt.Assert(t, qt.IsNil(err))

	res, err = l.Allow(ctx, longA)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(res.Allowed))
}
