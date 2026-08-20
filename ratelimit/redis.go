// Copyright 2026 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"azugo.io/core/cache"

	"github.com/valkey-io/valkey-go"
)

// All scripts execute atomically on the Redis server and use the server
// clock (TIME) so that limits are enforced consistently across service
// instances regardless of their local clock skew. Requires Redis 6.2+
// or Valkey (matching the cache backend minimum supported server).
var (
	// KEYS[1] - counter key
	// ARGV[1] - limit, ARGV[2] - window ms, ARGV[3] - n, ARGV[4] - peek
	// Returns {allowed, remaining, retry_after_ms, reset_ms}.
	//nolint:dupword
	fixedWindowScript = valkey.NewLuaScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local n = tonumber(ARGV[3])
local peek = ARGV[4] == "true"

local current = redis.call("GET", key)
local parsed = tonumber(current)
local count = parsed or 0
local ttl = redis.call("PTTL", key)
if ttl == -2 then
	count = 0
	ttl = window
elseif ttl == -1 then
	ttl = window
end

local remaining = limit - count
if remaining < 0 then
	remaining = 0
end

if n > limit then
	return {0, remaining, 0, ttl}
end

if count + n > limit then
	return {0, remaining, ttl, ttl}
end

if peek then
	return {1, remaining, 0, ttl}
end

if current and parsed == nil then
	count = n
	redis.call("SET", key, tostring(count), "PX", window)
else
	count = redis.call("INCRBY", key, n)
	if redis.call("PTTL", key) < 0 then
		redis.call("PEXPIRE", key, window)
	end
end

remaining = limit - count
if remaining < 0 then
	remaining = 0
end

return {1, remaining, 0, redis.call("PTTL", key)}
`)

	// Generic cell rate algorithm (GCRA): the only state per key is the
	// theoretical arrival time (TAT) of the next event if the bucket were
	// drained at exactly the configured rate. The bucket is full when TAT is
	// in the past and empty when TAT is a full burst ahead of now.
	//
	// KEYS[1] - TAT key
	// ARGV[1] - emission interval ms, ARGV[2] - burst capacity ms,
	// ARGV[3] - n, ARGV[4] - burst, ARGV[5] - peek
	// Returns {allowed, remaining, retry_after_ms, reset_ms} with durations
	// rounded up to whole milliseconds.
	tokenBucketScript = valkey.NewLuaScript(`
local key = KEYS[1]
local emission = tonumber(ARGV[1])
local tau = tonumber(ARGV[2])
local n = tonumber(ARGV[3])
local burst = tonumber(ARGV[4])
local peek = ARGV[5] == "true"

local t = redis.call("TIME")
local now = t[1] * 1000 + t[2] / 1000

local tat = tonumber(redis.call("GET", key)) or 0
if tat < now then
	tat = now
end

local remaining = math.floor((now - (tat - tau)) / emission)
if remaining < 0 then
	remaining = 0
elseif remaining > burst then
	remaining = burst
end

local reset = tat - now
if reset < 0 then
	reset = 0
end

if n > burst then
	return {0, remaining, 0, math.ceil(reset)}
end

local new_tat = tat + n * emission
local allow_at = new_tat - tau

if allow_at > now then
	return {0, remaining, math.ceil(allow_at - now), math.ceil(reset)}
end

if peek then
	return {1, remaining, 0, math.ceil(reset)}
end

redis.call("SET", key, tostring(new_tat), "PX", math.ceil(new_tat - now))

remaining = math.floor((now - (new_tat - tau)) / emission)
if remaining < 0 then
	remaining = 0
end

return {1, remaining, 0, math.ceil(new_tat - now)}
`)

	// Slots are members of a sorted set scored by lease expiry time so that
	// leases of crashed holders expire automatically.
	//
	// KEYS[1] - lease set key
	// ARGV[1] - slots, ARGV[2] - lease TTL ms, ARGV[3] - lease token
	// Returns {acquired, retry_after_ms}.
	semaphoreAcquireScript = valkey.NewLuaScript(`
local key = KEYS[1]
local slots = tonumber(ARGV[1])
local lease = tonumber(ARGV[2])
local token = ARGV[3]

local t = redis.call("TIME")
local now = math.floor(t[1] * 1000 + t[2] / 1000)

redis.call("ZREMRANGEBYSCORE", key, "-inf", now)

if redis.call("ZCARD", key) < slots then
	redis.call("ZADD", key, now + lease, token)
	redis.call("PEXPIRE", key, lease)

	return {1, 0}
end

local first = redis.call("ZRANGE", key, 0, 0, "WITHSCORES")

return {0, math.ceil(tonumber(first[2]) - now)}
`)
)

type redisLimiter struct {
	con    func() valkey.Client
	prefix string
	opt    *options
	// Token bucket script arguments are constant per limiter and formatted
	// once to avoid float formatting on every call.
	emissionArg string
	tauArg      string
}

func newRedisLimiter(con func() valkey.Client, prefix string, opt *options) *redisLimiter {
	r := &redisLimiter{con: con, prefix: prefix, opt: opt}

	if opt.strategy == strategyTokenBucket {
		emission := 1000 / opt.rate
		r.emissionArg = strconv.FormatFloat(emission, 'f', -1, 64)
		r.tauArg = strconv.FormatFloat(emission*float64(opt.burst), 'f', -1, 64)
	}

	return r
}

func (r *redisLimiter) allowN(ctx context.Context, key string, n int, peek bool, limit int) (Result, error) {
	if r.opt.strategy == strategyFixedWindow {
		return r.fixedWindow(ctx, key, n, peek, limit)
	}

	return r.tokenBucket(ctx, key, n, peek, limit)
}

func (r *redisLimiter) fixedWindow(ctx context.Context, key string, n int, peek bool, limit int) (Result, error) {
	con := r.con()
	if con == nil {
		return Result{}, cache.ErrCacheUnavailable
	}

	if limit <= 0 {
		limit = r.opt.limit
	}

	vals, err := fixedWindowScript.Exec(ctx, con, []string{r.prefix + key}, []string{
		strconv.Itoa(limit),
		strconv.FormatInt(r.opt.window.Milliseconds(), 10),
		strconv.Itoa(n),
		strconv.FormatBool(peek),
	}).ToArray()
	if err != nil {
		return Result{}, err
	}

	return parseLimiterReply(vals)
}

func (r *redisLimiter) tokenBucket(ctx context.Context, key string, n int, peek bool, limit int) (Result, error) {
	con := r.con()
	if con == nil {
		return Result{}, cache.ErrCacheUnavailable
	}

	burst := r.opt.burst
	tauArg := r.tauArg

	if limit > 0 {
		burst = limit
		tauArg = strconv.FormatFloat((1000/r.opt.rate)*float64(burst), 'f', -1, 64)
	}

	vals, err := tokenBucketScript.Exec(ctx, con, []string{r.prefix + key}, []string{
		r.emissionArg,
		tauArg,
		strconv.Itoa(n),
		strconv.Itoa(burst),
		strconv.FormatBool(peek),
	}).ToArray()
	if err != nil {
		return Result{}, err
	}

	return parseLimiterReply(vals)
}

func (r *redisLimiter) reset(ctx context.Context, key string) error {
	con := r.con()
	if con == nil {
		return cache.ErrCacheUnavailable
	}

	return con.Do(ctx, con.B().Del().Key(r.prefix+key).Build()).Error()
}

type redisSemaphore struct {
	con    func() valkey.Client
	prefix string
	opt    *options
}

func newRedisSemaphore(con func() valkey.Client, prefix string, opt *options) *redisSemaphore {
	return &redisSemaphore{con: con, prefix: prefix, opt: opt}
}

func (r *redisSemaphore) acquire(ctx context.Context, key, token string) (bool, time.Duration, error) {
	con := r.con()
	if con == nil {
		return false, 0, cache.ErrCacheUnavailable
	}

	vals, err := semaphoreAcquireScript.Exec(ctx, con, []string{r.prefix + key}, []string{
		strconv.Itoa(r.opt.slots),
		strconv.FormatInt(r.opt.leaseTTL.Milliseconds(), 10),
		token,
	}).ToArray()
	if err != nil {
		return false, 0, err
	}

	if len(vals) != 2 {
		return false, 0, fmt.Errorf("unexpected semaphore script reply: %v", vals)
	}

	acquired, err := vals[0].AsInt64()
	if err != nil {
		return false, 0, fmt.Errorf("unexpected semaphore script reply: %w", err)
	}

	retryAfter, err := vals[1].AsInt64()
	if err != nil {
		return false, 0, fmt.Errorf("unexpected semaphore script reply: %w", err)
	}

	return acquired == 1, time.Duration(retryAfter) * time.Millisecond, nil
}

func (r *redisSemaphore) release(ctx context.Context, key, token string) error {
	con := r.con()
	if con == nil {
		return cache.ErrCacheUnavailable
	}

	return con.Do(ctx, con.B().Zrem().Key(r.prefix+key).Member(token).Build()).Error()
}

func parseLimiterReply(vals []valkey.ValkeyMessage) (Result, error) {
	if len(vals) != 4 {
		return Result{}, fmt.Errorf("unexpected rate limit script reply: %v", vals)
	}

	var nums [4]int64

	for i := range vals {
		n, err := vals[i].AsInt64()
		if err != nil {
			return Result{}, fmt.Errorf("unexpected rate limit script reply: %w", err)
		}

		nums[i] = n
	}

	return Result{
		Allowed:    nums[0] == 1,
		Remaining:  int(nums[1]),
		RetryAfter: time.Duration(nums[2]) * time.Millisecond,
		ResetAt:    time.Now().Add(time.Duration(nums[3]) * time.Millisecond),
	}, nil
}
