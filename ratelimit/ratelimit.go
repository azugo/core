// Package ratelimit provides rate limiting and concurrency limiting
// primitives.
//
// When the cache is Redis backed, limits are enforced atomically across all
// service instances sharing the same cache. With the memory cache limits
// apply to a single service instance only.
package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	mrand "math/rand/v2"
	"time"

	"azugo.io/core/cache"
	"azugo.io/core/instrumenter"

	"github.com/valkey-io/valkey-go"
)

// Instrumentation operation names for rate limiter events.
const (
	InstrumentationAllow = "ratelimit-allow"
	InstrumentationPeek  = "ratelimit-peek"
	InstrumentationWait  = "ratelimit-wait"
	InstrumentationReset = "ratelimit-reset"
)

// ErrWaitLimitExceeded is returned by Limiter.Wait and Semaphore.Acquire when
// the event can not be permitted before the wait limit or context deadline.
var ErrWaitLimitExceeded = errors.New("rate limit wait time exceeded")

// minRetryInterval prevents hot-looping when the backend reports a zero or
// sub-millisecond retry interval.
const minRetryInterval = 5 * time.Millisecond

// maxKeyLength bounds the per-key memory and storage cost of possibly
// attacker controlled keys (e.g. a user supplied login name); longer keys
// are replaced by their SHA-256 hash.
const maxKeyLength = 64

func noopFinish(error) {}

// observe wraps instrumenter.Observe so that the variadic argument slice and
// interface boxing of key are not allocated when no instrumenter is
// configured. extra arguments (e.g. a *Result the operation fills in before
// finish is called) are forwarded after the key.
func observe(ctx context.Context, instr instrumenter.Instrumenter, op, key string, extra ...any) func(error) {
	if instr.Empty() {
		return noopFinish
	}

	if len(extra) == 0 {
		return instr(ctx, op, key)
	}

	args := make([]any, 0, len(extra)+1)
	args = append(args, key)
	args = append(args, extra...)

	return instr(ctx, op, args...)
}

func normalizeKey(key string) string {
	if len(key) <= maxKeyLength {
		return key
	}

	sum := sha256.Sum256([]byte(key))

	return hex.EncodeToString(sum[:])
}

// Result describes the outcome of a rate limiter check.
type Result struct {
	// Allowed reports whether the event is permitted.
	Allowed bool
	// Remaining is the number of events left in the current window or burst.
	Remaining int
	// RetryAfter is the time to wait until the next event would be permitted.
	// It is zero when the event is allowed.
	RetryAfter time.Duration
	// ResetAt is the approximate time when the limit fully replenishes.
	ResetAt time.Time
}

// Limiter limits the rate of events per key.
type Limiter interface {
	// Allow consumes one event for key if it is permitted right now.
	Allow(ctx context.Context, key string) (Result, error)
	// AllowN consumes n events at once for key if all of them are permitted
	// right now. Requesting more events than the configured limit or burst
	// is always denied.
	AllowN(ctx context.Context, key string, n int) (Result, error)
	// AllowLimit consumes one event for key if it is permitted right now
	// using custom limit.
	// Provide zero limit to use configured default limit.
	AllowLimit(ctx context.Context, key string, limit int) (Result, error)
	// Peek reports the current state for key without consuming any events.
	Peek(ctx context.Context, key string) (Result, error)
	// Wait blocks until one event for key is permitted and consumes it.
	// It returns ErrWaitLimitExceeded as soon as it is known that the event
	// can not be permitted before the WaitLimit option or context deadline,
	// or ctx.Err() if the context is cancelled while waiting.
	Wait(ctx context.Context, key string) error
	// Reset clears the rate limiter state for key.
	Reset(ctx context.Context, key string) error
}

type limiterBackend interface {
	allowN(ctx context.Context, key string, n int, peek bool, limit int) (Result, error)
	reset(ctx context.Context, key string) error
}

func newLimiter(c *cache.Cache, name string, opt *options) (Limiter, error) {
	if name == "" {
		return nil, errors.New("rate limiter name is required")
	}

	var b limiterBackend
	if con, distributed := connection(c); distributed {
		b = newRedisLimiter(con, keyPrefix(c, "ratelimit", name), opt)
	} else {
		b = newMemoryLimiter(opt)
	}

	return &limiter{backend: b, opt: opt}, nil
}

// NewFixedWindow creates a fixed-window rate limiter named name backed by the
// cache. It allows at most limit events per window.
func NewFixedWindow(c *cache.Cache, name string, limit int, window time.Duration, opts ...LimiterOption) (Limiter, error) {
	opt, err := newFixedWindowOptions(limit, window, opts...)
	if err != nil {
		return nil, err
	}

	return newLimiter(c, name, opt)
}

// NewTokenBucket creates a token-bucket rate limiter named name backed by the
// cache. It allows a sustained rate events/second with bursts up to burst.
func NewTokenBucket(c *cache.Cache, name string, rate float64, burst int, opts ...LimiterOption) (Limiter, error) {
	opt, err := newTokenBucketOptions(rate, burst, opts...)
	if err != nil {
		return nil, err
	}

	return newLimiter(c, name, opt)
}

// connection returns a lazy connection provider so that limiters can be
// created before the cache backend connection is established.
func connection(c *cache.Cache) (func() valkey.Client, bool) {
	if c.ConfiguredType() == cache.MemoryCache {
		return nil, false
	}

	return c.Connection, true
}

func keyPrefix(c *cache.Cache, kind, name string) string {
	prefix := c.ConfiguredKeyPrefix()
	if prefix != "" {
		prefix += ":"
	}

	return prefix + kind + ":" + name + ":"
}

type limiter struct {
	backend limiterBackend
	opt     *options
}

// failOpen converts backend errors to an allowed result when the FailOpen
// option is set.
func (l *limiter) failOpen(res Result, err error) (Result, error) {
	if err != nil && l.opt.failOpen {
		return Result{Allowed: true}, nil
	}

	return res, err
}

func (l *limiter) Allow(ctx context.Context, key string) (Result, error) {
	return l.AllowN(ctx, key, 1)
}

func (l *limiter) AllowN(ctx context.Context, key string, n int) (Result, error) {
	if n <= 0 {
		return Result{}, errors.New("event count must be positive")
	}

	key = normalizeKey(key)

	var res Result

	finish := observe(ctx, l.opt.instrumenter, InstrumentationAllow, key, &res)

	res, err := l.backend.allowN(ctx, key, n, false, 0)
	finish(err)

	return l.failOpen(res, err)
}

func (l *limiter) AllowLimit(ctx context.Context, key string, limit int) (Result, error) {
	key = normalizeKey(key)

	var res Result

	finish := observe(ctx, l.opt.instrumenter, InstrumentationAllow, key, &res)

	res, err := l.backend.allowN(ctx, key, 1, false, limit)
	finish(err)

	return l.failOpen(res, err)
}

func (l *limiter) Peek(ctx context.Context, key string) (Result, error) {
	key = normalizeKey(key)

	var res Result

	finish := observe(ctx, l.opt.instrumenter, InstrumentationPeek, key, &res)

	res, err := l.backend.allowN(ctx, key, 1, true, 0)
	finish(err)

	return l.failOpen(res, err)
}

func (l *limiter) Wait(ctx context.Context, key string) error {
	key = normalizeKey(key)

	finish := observe(ctx, l.opt.instrumenter, InstrumentationWait, key)

	err := waitLoop(ctx, l.opt.waitLimit, 0, func(ctx context.Context) (bool, time.Duration, error) {
		res, err := l.failOpen(l.backend.allowN(ctx, key, 1, false, 0))
		if err != nil {
			return false, 0, err
		}

		return res.Allowed, res.RetryAfter, nil
	})
	finish(err)

	return err
}

func (l *limiter) Reset(ctx context.Context, key string) error {
	key = normalizeKey(key)

	finish := observe(ctx, l.opt.instrumenter, InstrumentationReset, key)

	err := l.backend.reset(ctx, key)
	finish(err)

	return err
}

func waitLoop(ctx context.Context, waitLimit, maxInterval time.Duration, attempt func(context.Context) (bool, time.Duration, error)) error {
	var deadline time.Time
	if waitLimit > 0 {
		deadline = time.Now().Add(waitLimit)
	}

	if d, ok := ctx.Deadline(); ok && (deadline.IsZero() || d.Before(deadline)) {
		deadline = d
	}

	var timer *time.Timer

	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		ok, retryAfter, err := attempt(ctx)
		if err != nil {
			return err
		}

		if ok {
			return nil
		}

		if maxInterval > 0 && retryAfter > maxInterval {
			retryAfter = maxInterval
		}

		if retryAfter < minRetryInterval {
			retryAfter = minRetryInterval
		}

		// retry jitter
		retryAfter += time.Duration(mrand.Int64N(int64(retryAfter)/10 + 1)) //nolint:gosec

		if !deadline.IsZero() && time.Now().Add(retryAfter).After(deadline) {
			return ErrWaitLimitExceeded
		}

		if timer == nil {
			timer = time.NewTimer(retryAfter)
		} else {
			timer.Reset(retryAfter)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
}
