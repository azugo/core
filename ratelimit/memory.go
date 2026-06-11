// Copyright 2026 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ratelimit

import (
	"context"
	"sync"
	"time"
)

// cleanupInterval is the number of operations between sweeps of expired keys.
const cleanupInterval = 4096

// capEntries bounds the map to maxKeys entries before a new key is inserted.
// Expired entries are removed first; if the map is still full, the soonest
// expiring entry of a small random sample is evicted.
func capEntries[V any](entries map[string]V, maxKeys int, now time.Time, expiresAt func(V) time.Time) {
	if len(entries) < maxKeys {
		return
	}

	for k, v := range entries {
		if !expiresAt(v).After(now) {
			delete(entries, k)
		}
	}

	if len(entries) < maxKeys {
		return
	}

	var (
		victim  string
		expires time.Time
		sampled int
	)

	for k, v := range entries {
		if e := expiresAt(v); sampled == 0 || e.Before(expires) {
			victim, expires = k, e
		}

		sampled++

		// remove up to 8 entries to clean up some space
		if sampled >= 8 {
			break
		}
	}

	delete(entries, victim)
}

type windowEntry struct {
	count   int
	resetAt time.Time
}

type memoryLimiter struct {
	opt *options

	mu  sync.Mutex
	ops int
	// Fixed window counters by key.
	windows map[string]*windowEntry
	// Token bucket times by key.
	tats map[string]time.Time
}

func newMemoryLimiter(opt *options) *memoryLimiter {
	return &memoryLimiter{
		opt:     opt,
		windows: make(map[string]*windowEntry),
		tats:    make(map[string]time.Time),
	}
}

func (m *memoryLimiter) allowN(_ context.Context, key string, n int, peek bool) (Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cleanup()

	if m.opt.strategy == strategyFixedWindow {
		return m.fixedWindow(key, n, peek), nil
	}

	return m.tokenBucket(key, n, peek), nil
}

func (m *memoryLimiter) fixedWindow(key string, n int, peek bool) Result {
	now := time.Now()

	e := m.windows[key]
	if e != nil && !e.resetAt.After(now) {
		delete(m.windows, key)

		e = nil
	}

	count := 0
	resetAt := now.Add(m.opt.window)

	if e != nil {
		count = e.count
		resetAt = e.resetAt
	}

	remaining := max(m.opt.limit-count, 0)

	if n > m.opt.limit-count {
		return Result{
			Remaining:  remaining,
			RetryAfter: resetAt.Sub(now),
			ResetAt:    resetAt,
		}
	}

	if peek {
		return Result{Allowed: true, Remaining: remaining, ResetAt: resetAt}
	}

	if e == nil {
		capEntries(m.windows, m.opt.maxKeys, now, func(e *windowEntry) time.Time { return e.resetAt })

		e = &windowEntry{resetAt: resetAt}
		m.windows[key] = e
	}

	e.count += n

	return Result{Allowed: true, Remaining: m.opt.limit - e.count, ResetAt: resetAt}
}

// tokenBucket implements the generic cell rate algorithm (GCRA).
func (m *memoryLimiter) tokenBucket(key string, n int, peek bool) Result {
	now := time.Now()

	emission := max(time.Duration(float64(time.Second)/m.opt.rate), 1)
	tau := time.Duration(m.opt.burst) * emission

	tat, ok := m.tats[key]
	if !ok || tat.Before(now) {
		tat = now
	}

	remaining := int(now.Sub(tat.Add(-tau)) / emission)
	remaining = min(max(remaining, 0), m.opt.burst)

	if n > m.opt.burst {
		return Result{
			Remaining:  remaining,
			RetryAfter: tat.Sub(now) + emission,
			ResetAt:    tat,
		}
	}

	newTat := tat.Add(time.Duration(n) * emission)
	allowAt := newTat.Add(-tau)

	if allowAt.After(now) {
		return Result{
			Remaining:  remaining,
			RetryAfter: allowAt.Sub(now),
			ResetAt:    tat,
		}
	}

	if peek {
		return Result{Allowed: true, Remaining: remaining, ResetAt: tat}
	}

	if !ok {
		capEntries(m.tats, m.opt.maxKeys, now, func(tat time.Time) time.Time { return tat })
	}

	m.tats[key] = newTat

	return Result{
		Allowed:   true,
		Remaining: max(int(now.Sub(newTat.Add(-tau))/emission), 0),
		ResetAt:   newTat,
	}
}

func (m *memoryLimiter) reset(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.windows, key)
	delete(m.tats, key)

	return nil
}

// cleanup expired keys.
// Must be called with the lock held.
func (m *memoryLimiter) cleanup() {
	m.ops++
	if m.ops < cleanupInterval {
		return
	}

	m.ops = 0
	now := time.Now()

	for k, e := range m.windows {
		if !e.resetAt.After(now) {
			delete(m.windows, k)
		}
	}

	for k, tat := range m.tats {
		if tat.Before(now) {
			delete(m.tats, k)
		}
	}
}

type memorySemaphore struct {
	opt *options

	mu sync.Mutex
	// Lease expiry times by key and lease token.
	leases map[string]map[string]time.Time
}

func newMemorySemaphore(opt *options) *memorySemaphore {
	return &memorySemaphore{
		opt:    opt,
		leases: make(map[string]map[string]time.Time),
	}
}

func (m *memorySemaphore) acquire(_ context.Context, key, token string) (bool, time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	ls := m.leases[key]

	var earliest time.Time

	for t, expiresAt := range ls {
		if !expiresAt.After(now) {
			delete(ls, t)

			continue
		}

		if earliest.IsZero() || expiresAt.Before(earliest) {
			earliest = expiresAt
		}
	}

	if len(ls) < m.opt.slots {
		if ls == nil {
			// A key is reclaimable only when its last lease has expired.
			capEntries(m.leases, m.opt.maxKeys, now, func(ls map[string]time.Time) time.Time {
				var latest time.Time
				for _, expiresAt := range ls {
					if expiresAt.After(latest) {
						latest = expiresAt
					}
				}

				return latest
			})

			ls = make(map[string]time.Time)
			m.leases[key] = ls
		}

		ls[token] = now.Add(m.opt.leaseTTL)

		return true, 0, nil
	}

	return false, earliest.Sub(now), nil
}

func (m *memorySemaphore) release(_ context.Context, key, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ls := m.leases[key]; ls != nil {
		delete(ls, token)

		if len(ls) == 0 {
			delete(m.leases, key)
		}
	}

	return nil
}
