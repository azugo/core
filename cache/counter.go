package cache

import (
	"context"
	"errors"
	"time"

	"github.com/valkey-io/valkey-go"
)

// Counter is a cache instance of int64 counters that change atomically.
type Counter interface {
	Instance[int64]

	// Increment adds delta to key and returns the resulting count.
	Increment(ctx context.Context, key string, delta int64, opts ...ItemOption[int64]) (int64, error)
}

// CreateCounter creates a counter cache instance with the given name and options.
func CreateCounter(cache *Cache, name string, opts ...Option) (Counter, error) {
	inst, err := Create[int64](cache, name, opts...)
	if err != nil {
		return nil, err
	}

	switch c := inst.(type) {
	case *memoryCache[int64]:
		return &memoryCounter{memoryCache: c}, nil
	case *redisCache[int64]:
		return &redisCounter{redisCache: c}, nil
	default:
		return nil, errors.New("unsupported cache type")
	}
}

type memoryCounter struct {
	*memoryCache[int64]
}

// Increment adds delta to key as atomic operation.
func (c *memoryCounter) Increment(ctx context.Context, key string, delta int64, opts ...ItemOption[int64]) (int64, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	finish := c.observe(ctx, InstrumentationSet, key)

	count, found, err := c.peek(key)
	if err != nil {
		finish(err)

		return 0, err
	}

	ttl := newItemOptions(opts...).TTL
	if ttl == 0 {
		ttl = c.ttl
	}

	if found {
		var (
			remaining time.Duration
			timed     bool
		)

		if c.serialize {
			remaining, timed = c.serializedCache.GetTTL(key)
		} else {
			remaining, timed = c.cache.GetTTL(key)
		}

		if timed {
			ttl = remaining
		}
	}

	count += delta

	if err := c.write(key, count, TTL[int64](ttl)); err != nil {
		finish(err)

		return 0, err
	}

	finish(nil)

	return count, nil
}

type redisCounter struct {
	*redisCache[int64]
}

// Increment adds delta to key as atomic operation.
func (c *redisCounter) Increment(ctx context.Context, key string, delta int64, opts ...ItemOption[int64]) (int64, error) {
	con, err := c.connection()
	if err != nil {
		return 0, err
	}

	finish := c.observe(ctx, InstrumentationSet, key)

	ttl := c.ttl
	if opt := newItemOptions(opts...); opt.TTL != 0 {
		ttl = opt.TTL
	}

	create := con.B().Set().Key(c.prefix + key).Value("0").Nx()

	var completed valkey.Completed
	if ttl > 0 {
		completed = create.Px(ttl).Build()
	} else {
		completed = create.Build()
	}

	res := con.DoMulti(ctx, completed,
		con.B().Incrby().Key(c.prefix+key).Increment(delta).Build())

	if err := res[0].Error(); err != nil && !valkey.IsValkeyNil(err) {
		err = connError(err)
		finish(err)

		return 0, err
	}

	count, err := res[1].AsInt64()
	if err != nil {
		err = connError(err)
		finish(err)

		return 0, err
	}

	finish(nil)

	return count, nil
}
