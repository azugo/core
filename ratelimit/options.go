package ratelimit

import (
	"errors"
	"time"

	"azugo.io/core/instrumenter"
)

type strategy int

const (
	strategyFixedWindow strategy = iota + 1
	strategyTokenBucket
)

type options struct {
	strategy strategy
	// Fixed window strategy.
	limit  int
	window time.Duration
	// Token bucket strategy.
	rate  float64
	burst int
	// Semaphore.
	slots        int
	leaseTTL     time.Duration
	pollInterval time.Duration
	// Shared.
	waitLimit    time.Duration
	failOpen     bool
	maxKeys      int
	instrumenter instrumenter.Instrumenter
}

// LimiterOption configures a rate limiter.
type LimiterOption interface {
	applyLimiter(opt *options)
}

// SemaphoreOption configures a semaphore.
type SemaphoreOption interface {
	applySemaphore(opt *options)
}

func defaultOptions() *options {
	opt := &options{
		leaseTTL:     time.Minute,
		pollInterval: 100 * time.Millisecond,
		maxKeys:      100_000,
	}

	return opt
}

func newLimiterOptions(opts ...LimiterOption) *options {
	opt := defaultOptions()

	for _, o := range opts {
		o.applyLimiter(opt)
	}

	return opt
}

func newFixedWindowOptions(limit int, window time.Duration, opts ...LimiterOption) (*options, error) {
	opt := newLimiterOptions(opts...)
	opt.strategy = strategyFixedWindow
	opt.limit = limit
	opt.window = window

	if err := opt.validateFixedWindow(); err != nil {
		return nil, err
	}

	return opt, nil
}

func newTokenBucketOptions(rate float64, burst int, opts ...LimiterOption) (*options, error) {
	opt := newLimiterOptions(opts...)
	opt.strategy = strategyTokenBucket
	opt.rate = rate
	opt.burst = burst

	if err := opt.validateTokenBucket(); err != nil {
		return nil, err
	}

	return opt, nil
}

func newSemaphoreOptions(opts ...SemaphoreOption) *options {
	opt := defaultOptions()

	for _, o := range opts {
		o.applySemaphore(opt)
	}

	return opt
}

func (o *options) validateFixedWindow() error {
	if o.limit <= 0 {
		return errors.New("fixed window limit must be positive")
	}

	if o.window <= 0 {
		return errors.New("fixed window duration must be positive")
	}

	if o.waitLimit < 0 {
		return errors.New("wait limit must not be negative")
	}

	if o.maxKeys <= 0 {
		return errors.New("max keys must be positive")
	}

	return nil
}

func (o *options) validateTokenBucket() error {
	if o.rate <= 0 {
		return errors.New("token bucket rate must be positive")
	}

	if o.burst <= 0 {
		return errors.New("token bucket burst must be positive")
	}

	if o.waitLimit < 0 {
		return errors.New("wait limit must not be negative")
	}

	if o.maxKeys <= 0 {
		return errors.New("max keys must be positive")
	}

	return nil
}

func (o *options) validateSemaphore() error {
	if o.slots <= 0 {
		return errors.New("semaphore slot count must be positive")
	}

	if o.leaseTTL <= 0 {
		return errors.New("lease TTL must be positive")
	}

	if o.pollInterval <= 0 {
		return errors.New("poll interval must be positive")
	}

	if o.waitLimit < 0 {
		return errors.New("wait limit must not be negative")
	}

	if o.maxKeys <= 0 {
		return errors.New("max keys must be positive")
	}

	return nil
}

// WaitLimit is the maximum total time Limiter.Wait or Semaphore.Acquire may
// spend waiting. When not set, waiting is bounded only by the context.
type WaitLimit time.Duration

func (w WaitLimit) applyLimiter(o *options) {
	o.waitLimit = time.Duration(w)
}

func (w WaitLimit) applySemaphore(o *options) {
	o.waitLimit = time.Duration(w)
}

// LeaseTTL is the time after which a held semaphore slot is freed
// automatically if it is not released (e.g. the holder crashed). It must be
// longer than the longest expected hold time. Defaults to 1 minute.
type LeaseTTL time.Duration

func (l LeaseTTL) applySemaphore(o *options) {
	o.leaseTTL = time.Duration(l)
}

// PollInterval is the maximum time between checks for a free semaphore slot
// while waiting in Semaphore.Acquire. Defaults to 100 milliseconds.
type PollInterval time.Duration

func (p PollInterval) applySemaphore(o *options) {
	o.pollInterval = time.Duration(p)
}

// MaxKeys bounds the number of distinct keys tracked by the memory backend
// so that high cardinality (possibly attacker controlled) keys can not
// exhaust memory. When the bound is reached, expired entries are removed
// first; if all entries are still live, the soonest-expiring one is evicted.
// Evicting a live entry forgets its counter, so prefer a Redis backed cache
// when hostile key cardinality above this bound is expected. Has no effect
// on the Redis backend. Defaults to 100000.
type MaxKeys int

func (m MaxKeys) applyLimiter(o *options) {
	o.maxKeys = int(m)
}

func (m MaxKeys) applySemaphore(o *options) {
	o.maxKeys = int(m)
}

// FailOpen controls the behavior when the backend is unreachable. When set,
// events are allowed and slots are granted on backend errors instead of
// returning the error. Use for throttling that protects an external service
// where halting all traffic is worse than briefly exceeding the limit; keep
// disabled (the default) for security limits such as login throttling.
type FailOpen bool

func (f FailOpen) applyLimiter(o *options) {
	o.failOpen = bool(f)
}

func (f FailOpen) applySemaphore(o *options) {
	o.failOpen = bool(f)
}

// Instrumenter is a function that instruments rate limiter operations.
type Instrumenter instrumenter.Instrumenter

func (i Instrumenter) applyLimiter(o *options) {
	o.instrumenter = instrumenter.Instrumenter(i)
}

func (i Instrumenter) applySemaphore(o *options) {
	o.instrumenter = instrumenter.Instrumenter(i)
}
