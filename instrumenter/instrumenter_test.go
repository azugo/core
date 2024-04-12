package instrumenter

import (
	"context"
	"errors"
	"testing"

	"github.com/go-quicktest/qt"
)

func TestInstrumenter(t *testing.T) {
	var i Instrumenter
	var start, end bool
	i = func(ctx context.Context, op string, args ...any) func(err error) {
		start = true
		qt.Check(t, qt.Equals(op, "op"))
		return func(err error) {
			qt.Check(t, qt.ErrorMatches(err, "test error"))
			end = true
		}
	}

	cb := i.Observe(context.Background(), "op")
	qt.Check(t, qt.IsTrue(start))
	qt.Check(t, qt.IsFalse(end))
	cb(errors.New("test error"))
	qt.Check(t, qt.IsTrue(end))
}

func TestCombinedInstrumenters(t *testing.T) {
	var i Instrumenter

	calls := make([]string, 0, 4)

	i = CombinedInstrumenter(
		func(ctx context.Context, op string, args ...any) func(err error) {
			qt.Check(t, qt.Equals(op, "op"))
			calls = append(calls, "1s")
			return func(err error) {
				qt.Check(t, qt.ErrorMatches(err, "test error"))
				calls = append(calls, "1e")
			}
		},
		func(ctx context.Context, op string, args ...any) func(err error) {
			qt.Check(t, qt.DeepEquals(op, "op"))
			calls = append(calls, "2s")
			return func(err error) {
				qt.Check(t, qt.ErrorMatches(err, "test error"))
				calls = append(calls, "2e")
			}
		},
	)

	cb := i.Observe(context.Background(), "op")
	qt.Check(t, qt.DeepEquals(calls, []string{"1s", "2s"}))
	cb(errors.New("test error"))
	qt.Check(t, qt.DeepEquals(calls, []string{"1s", "2s", "2e", "1e"}))
}

func TestNilInstrumenter(t *testing.T) {
	var i Instrumenter
	cb := i.Observe(context.Background(), "op")
	qt.Check(t, qt.IsNotNil(cb))
}
