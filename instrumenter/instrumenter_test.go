package instrumenter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstrumenter(t *testing.T) {
	var i Instrumenter
	var start, end bool
	i = func(ctx context.Context, op string, args ...any) func(err error) {
		start = true
		assert.Equal(t, "op", op)
		return func(err error) {
			assert.ErrorContains(t, err, "test error")
			end = true
		}
	}

	cb := i.Observe(context.Background(), "op")
	assert.True(t, start)
	assert.False(t, end)
	cb(errors.New("test error"))
	assert.True(t, end)
}

func TestCombinedInstrumenters(t *testing.T) {
	var i Instrumenter

	calls := make([]string, 0, 4)

	i = CombinedInstrumenter(
		func(ctx context.Context, op string, args ...any) func(err error) {
			assert.Equal(t, "op", op)
			calls = append(calls, "1s")
			return func(err error) {
				assert.ErrorContains(t, err, "test error")
				calls = append(calls, "1e")
			}
		},
		func(ctx context.Context, op string, args ...any) func(err error) {
			assert.Equal(t, "op", op)
			calls = append(calls, "2s")
			return func(err error) {
				assert.ErrorContains(t, err, "test error")
				calls = append(calls, "2e")
			}
		},
	)

	cb := i.Observe(context.Background(), "op")
	assert.Equal(t, []string{"1s", "2s"}, calls)
	cb(errors.New("test error"))
	assert.Equal(t, []string{"1s", "2s", "2e", "1e"}, calls)
}

func TestNilInstrumenter(t *testing.T) {
	var i Instrumenter
	cb := i.Observe(context.Background(), "op")
	assert.NotNil(t, cb)
}
