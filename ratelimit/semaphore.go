package ratelimit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"azugo.io/core/cache"
)

// Instrumentation operation names for semaphore events.
const (
	InstrumentationAcquire = "semaphore-acquire"
	InstrumentationRelease = "semaphore-release"
)

// Semaphore limits the number of concurrently held slots per key.
type Semaphore interface {
	// Acquire blocks until a slot for key is available and takes it. The
	// returned release function must be called to free the slot; the slot is
	// also freed automatically when the lease TTL expires. It returns
	// ErrWaitLimitExceeded when no slot becomes available before the
	// WaitLimit option or context deadline, or ctx.Err() if the context is
	// cancelled while waiting.
	Acquire(ctx context.Context, key string) (release func(), err error)
	// TryAcquire takes a slot for key if one is available right now.
	TryAcquire(ctx context.Context, key string) (release func(), ok bool, err error)
}

type semaphoreBackend interface {
	acquire(ctx context.Context, key, token string) (ok bool, retryAfter time.Duration, err error)
	release(ctx context.Context, key, token string) error
}

// NewSemaphore creates a concurrency limiter backed by the cache.
// slots is the maximum number of concurrently held slots per key.
func NewSemaphore(c *cache.Cache, name string, slots int, opts ...SemaphoreOption) (Semaphore, error) {
	if name == "" {
		return nil, errors.New("semaphore name is required")
	}

	opt := newSemaphoreOptions(opts...)
	opt.slots = slots

	if err := opt.validateSemaphore(); err != nil {
		return nil, err
	}

	con, distributed, err := connection(c)
	if err != nil {
		return nil, err
	}

	var b semaphoreBackend
	if distributed {
		b = newRedisSemaphore(con, keyPrefix(c, "semaphore", name), opt)
	} else {
		b = newMemorySemaphore(opt)
	}

	return &semaphore{backend: b, opt: opt}, nil
}

type semaphore struct {
	backend semaphoreBackend
	opt     *options
}

func (s *semaphore) Acquire(ctx context.Context, key string) (func(), error) {
	key = normalizeKey(key)

	finish := observe(ctx, s.opt.instrumenter, InstrumentationAcquire, key)

	token, err := newToken()
	if err != nil {
		finish(err)

		return nil, err
	}

	err = waitLoop(ctx, s.opt.waitLimit, s.opt.pollInterval, func(ctx context.Context) (bool, time.Duration, error) {
		ok, retryAfter, err := s.backend.acquire(ctx, key, token)
		if err != nil {
			if s.opt.failOpen {
				return true, 0, nil
			}

			return false, 0, err
		}

		return ok, retryAfter, nil
	})
	finish(err)

	if err != nil {
		return nil, err
	}

	return s.releaseFunc(ctx, key, token), nil
}

func (s *semaphore) TryAcquire(ctx context.Context, key string) (func(), bool, error) {
	key = normalizeKey(key)

	finish := observe(ctx, s.opt.instrumenter, InstrumentationAcquire, key)

	token, err := newToken()
	if err != nil {
		finish(err)

		return nil, false, err
	}

	ok, _, err := s.backend.acquire(ctx, key, token)
	finish(err)

	if err != nil {
		if s.opt.failOpen {
			return func() {}, true, nil
		}

		return nil, false, err
	}

	if !ok {
		return nil, false, nil
	}

	return s.releaseFunc(ctx, key, token), true, nil
}

func (s *semaphore) releaseFunc(ctx context.Context, key, token string) func() {
	var once sync.Once

	ctx = context.WithoutCancel(ctx)

	return func() {
		once.Do(func() {
			finish := observe(ctx, s.opt.instrumenter, InstrumentationRelease, key)
			finish(s.backend.release(ctx, key, token))
		})
	}
}

func newToken() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("failed to generate semaphore lease token: %w", err)
	}

	return hex.EncodeToString(buf[:]), nil
}
