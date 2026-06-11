// Package instrumenter provides an interface for observing cache, ratelimit and HTTP operations.
package instrumenter

import (
	"context"
)

// NullFinish helper when no error handling is needed.
func NullFinish(error) {}

// Instrumenter defines function type that can be used for instrumentation.
// This function should return a function with no argument as a callback for finished execution.
type Instrumenter func(ctx context.Context, op string, args ...any) func(err error)

// Observe operation.
func (i Instrumenter) Observe(ctx context.Context, op string, args ...any) func(err error) {
	if i == nil {
		return NullFinish
	}

	return i(ctx, op, args...)
}

// ObserveKey helper for instrumenting operations with a single key argument.
func ObserveKey(ctx context.Context, i Instrumenter, op, key string) func(error) {
	if i == nil {
		return NullFinish
	}

	return i(ctx, op, key)
}

// NullInstrumenter is a no-op instrumenter.
func NullInstrumenter(_ context.Context, _ string, _ ...any) func(err error) {
	return NullFinish
}

// CombinedInstrumenter is an instrumenter that combines multiple instrumenters.
func CombinedInstrumenter(instr ...Instrumenter) Instrumenter {
	return func(ctx context.Context, op string, args ...any) func(err error) {
		l := len(instr)

		cb := make([]func(error), l)
		for i, ii := range instr {
			cb[l-i-1] = ii.Observe(ctx, op, args...)
		}

		return func(err error) {
			for _, c := range cb {
				c(err)
			}
		}
	}
}
