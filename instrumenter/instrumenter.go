package instrumenter

import (
	"context"
)

// Instrumenter defines function type that can be used for instrumentation.
// This function should return a function with no argument as a callback for finished execution.
type Instrumenter func(ctx context.Context, op string, args ...any) func(err error)

// Observe operation.
func (i Instrumenter) Observe(ctx context.Context, op string, args ...any) func(err error) {
	if i != nil {
		return i(ctx, op, args...)
	}

	return func(_ error) {}
}

// NullInstrumenter is a no-op instrumenter.
func NullInstrumenter(_ context.Context, _ string, _ ...any) func(err error) {
	return func(_ error) {}
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
