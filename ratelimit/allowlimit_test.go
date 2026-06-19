package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/go-quicktest/qt"
)

func TestAllowLimitFixedWindow(t *testing.T) {
	l, err := NewFixedWindow(memCache(), "fw", 5, time.Minute)
	qt.Assert(t, qt.IsNil(err))

	ctx := context.Background()

	// Override to 2 for key "a": 2 allowed, 3rd denied.
	r, err := l.AllowLimit(ctx, "a", 2)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(r.Allowed))
	r, _ = l.AllowLimit(ctx, "a", 2)
	qt.Check(t, qt.IsTrue(r.Allowed))
	r, _ = l.AllowLimit(ctx, "a", 2)
	qt.Check(t, qt.IsFalse(r.Allowed))

	// A different key, same limiter, higher override: its own counter.
	r, _ = l.AllowLimit(ctx, "b", 10)
	qt.Check(t, qt.IsTrue(r.Allowed))

	// Non-positive override falls back to the configured limit (5).
	for range 5 {
		r, _ = l.AllowLimit(ctx, "c", 0)
		qt.Check(t, qt.IsTrue(r.Allowed))
	}
	r, _ = l.AllowLimit(ctx, "c", 0)
	qt.Check(t, qt.IsFalse(r.Allowed))
}

func TestAllowLimitTokenBucket(t *testing.T) {
	l, err := NewTokenBucket(memCache(), "tb", 1, 3) // rate 1/s, burst 3
	qt.Assert(t, qt.IsNil(err))

	ctx := context.Background()

	// Override burst to 2 for key "x": 2 allowed, 3rd denied.
	r, err := l.AllowLimit(ctx, "x", 2)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(r.Allowed))
	r, _ = l.AllowLimit(ctx, "x", 2)
	qt.Check(t, qt.IsTrue(r.Allowed))
	r, _ = l.AllowLimit(ctx, "x", 2)
	qt.Check(t, qt.IsFalse(r.Allowed))

	// Configured burst (3) for a different key.
	for range 3 {
		r, _ = l.AllowLimit(ctx, "y", 0)
		qt.Check(t, qt.IsTrue(r.Allowed))
	}
	r, _ = l.AllowLimit(ctx, "y", 0)
	qt.Check(t, qt.IsFalse(r.Allowed))
}
