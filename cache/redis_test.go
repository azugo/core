package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"azugo.io/core/instrumenter"

	"github.com/go-quicktest/qt"
)

func getRedisConnStr() string {
	return os.Getenv("REDIS_CONNSTR")
}

func TestParseRedisSentinelURL(t *testing.T) {
	opt, err := parseRedisSentinelURL("sentinel://user@s1:26379,s2:26379/mymaster?db=2")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.DeepEquals(opt.InitAddress, []string{"s1:26379", "s2:26379"}))
	qt.Check(t, qt.Equals(opt.Username, "user"))
	qt.Check(t, qt.Equals(opt.Sentinel.MasterSet, "mymaster"))
	qt.Check(t, qt.Equals(opt.SelectDB, 2))
	qt.Check(t, qt.IsNil(opt.TLSConfig))
	qt.Check(t, qt.IsNil(opt.Sentinel.TLSConfig))
}

func TestParseRedisSentinelURLTLS(t *testing.T) {
	opt, err := parseRedisSentinelURL("sentinels://s1:26379/mymaster")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsNotNil(opt.TLSConfig))
	qt.Check(t, qt.IsFalse(opt.TLSConfig.InsecureSkipVerify))
	qt.Check(t, qt.Equals(opt.Sentinel.TLSConfig, opt.TLSConfig))

	opt, err = parseRedisSentinelURL("sentinels://s1:26379/mymaster?skip_verify=true")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsNotNil(opt.TLSConfig))
	qt.Check(t, qt.IsTrue(opt.TLSConfig.InsecureSkipVerify))

	// skip_verify without TLS scheme is ignored.
	opt, err = parseRedisSentinelURL("sentinel://s1:26379/mymaster?skip_verify=true")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsNil(opt.TLSConfig))
}

func TestParseRedisSentinelURLErrors(t *testing.T) {
	_, err := parseRedisSentinelURL("redis://s1:26379/mymaster")
	qt.Check(t, qt.IsNotNil(err))

	_, err = parseRedisSentinelURL("sentinel://s1:26379")
	qt.Check(t, qt.IsNotNil(err))

	_, err = parseRedisSentinelURL("sentinel:///mymaster")
	qt.Check(t, qt.IsNotNil(err))

	_, err = parseRedisSentinelURL("sentinel://s1:26379/mymaster?db=abc")
	qt.Check(t, qt.IsNotNil(err))
}

func TestValidateConnectionStringSkipVerify(t *testing.T) {
	err := ValidateConnectionString(RedisCache, "redis://localhost:6379/0?skip_verify=true")
	qt.Check(t, qt.ErrorMatches(err, "skip_verify requires a TLS connection string scheme"))

	err = ValidateConnectionString(RedisSentinelCache, "sentinel://s1:26379/mymaster?skip_verify=true")
	qt.Check(t, qt.ErrorMatches(err, "skip_verify requires a TLS connection string scheme"))

	qt.Check(t, qt.IsNil(ValidateConnectionString(RedisCache, "rediss://localhost:6379/0?skip_verify=true")))
	qt.Check(t, qt.IsNil(ValidateConnectionString(RedisSentinelCache, "sentinels://s1:26379/mymaster?skip_verify=true")))
	qt.Check(t, qt.IsNil(ValidateConnectionString(RedisCache, "redis://localhost:6379/0?skip_verify=false")))
	qt.Check(t, qt.IsNil(ValidateConnectionString(RedisCache, "redis://localhost:6379/0")))
}

func TestRedisCacheStartUnreachable(t *testing.T) {
	ctx := t.Context()

	c := New(RedisCache, ConnectionString("redis://localhost:1/0"))
	err := c.Start(ctx)
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	qt.Check(t, qt.ErrorIs(c.Ping(ctx), ErrCacheUnavailable))

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	_, err = i.Get(ctx, "key")
	qt.Check(t, qt.ErrorIs(err, ErrCacheUnavailable))

	err = i.Set(ctx, "key", "value")
	qt.Check(t, qt.ErrorIs(err, ErrCacheUnavailable))

	c.Close()

	_, err = i.Get(ctx, "key")
	qt.Check(t, qt.ErrorIs(err, ErrCacheClosed))
}

func TestRedisCacheInstanceOwnConnectionUnreachable(t *testing.T) {
	ctx := t.Context()

	c := New(RedisCache, ConnectionString("redis://localhost:1/0"))
	qt.Assert(t, qt.IsNil(c.Start(ctx)))
	defer c.Close()

	i, err := Create[string](c, "test", ConnectionString("redis://localhost:2/0"))
	qt.Assert(t, qt.IsNil(err))

	_, err = i.Get(ctx, "key")
	qt.Check(t, qt.ErrorIs(err, ErrCacheUnavailable))

	_, err = Create[string](c, "test2", ConnectionString("http://localhost:6379"))
	qt.Check(t, qt.IsNotNil(err))
}

func TestRedisCacheStartInvalidConnectionString(t *testing.T) {
	c := New(RedisCache, ConnectionString("http://localhost:6379"))
	err := c.Start(context.TODO())
	qt.Check(t, qt.IsNotNil(err))
}

func TestRedisCacheGetSet(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skip("REDIS_CONNSTR is not set")
	}
	c := New(RedisCache, KeyPrefix("prefix"), ConnectionString(cs))
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key1", "value")
	qt.Check(t, qt.IsNil(err))

	val, err := i.Get(context.TODO(), "key1")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "value"))
}

func TestRedisCachePop(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skip("REDIS_CONNSTR is not set")
	}
	c := New(RedisCache, ConnectionString(cs))
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key2", "value")
	qt.Check(t, qt.IsNil(err))

	val, err := i.Pop(context.TODO(), "key2")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "value"))

	val, err = i.Pop(context.TODO(), "key2")
	qt.Check(t, qt.IsNotNil(err))
	qt.Check(t, qt.Equals(val, ""))
}

func TestRedisCacheDelete(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skip("REDIS_CONNSTR is not set")
	}
	c := New(RedisCache, ConnectionString(cs))
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test")
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key3", "value")
	qt.Check(t, qt.IsNil(err))

	err = i.Delete(context.TODO(), "key3")
	qt.Check(t, qt.IsNil(err))

	val, err := i.Get(context.TODO(), "key3")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, ""))
}

func TestRedisCacheExpire(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skip("REDIS_CONNSTR is not set")
	}
	c := New(RedisCache, ConnectionString(cs))
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "test", DefaultTTL(100*time.Millisecond))
	qt.Assert(t, qt.IsNil(err))

	err = i.Set(context.TODO(), "key4", "value")
	qt.Check(t, qt.IsNil(err))

	time.Sleep(150 * time.Millisecond)

	val, err := i.Get(context.TODO(), "key4")
	qt.Check(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, ""))
}

func TestClientCacheOptions(t *testing.T) {
	opt := newCacheOptions(ClientCacheTTL(time.Minute))
	qt.Check(t, qt.Equals(opt.ClientCacheTTL, time.Minute))
	qt.Check(t, qt.IsTrue(opt.ClientCache))
	qt.Check(t, qt.Equals(opt.ClientCacheSize, 0))

	// Instance level ClientCacheTTL(0) overrides the cache level default.
	opt = newCacheOptions(ClientCacheTTL(time.Minute), ClientCacheTTL(0))
	qt.Check(t, qt.Equals(opt.ClientCacheTTL, time.Duration(0)))

	opt = newCacheOptions(ClientCache(false), ClientCacheSize(8<<20))
	qt.Check(t, qt.IsFalse(opt.ClientCache))
	qt.Check(t, qt.Equals(opt.ClientCacheSize, 8<<20))
}

func TestRedisCacheClientCache(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skip("REDIS_CONNSTR is not set")
	}
	c := New(RedisCache, ConnectionString(cs))
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	hits := 0
	instr := func(_ context.Context, op string, _ ...any) func(error) {
		if op == InstrumentationGetHit {
			hits++
		}

		return instrumenter.NullFinish
	}

	i, err := Create[string](c, "cctest", ClientCacheTTL(time.Minute), Instrumenter(instr))
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	err = i.Set(ctx, "key", "value")
	qt.Assert(t, qt.IsNil(err))

	val, err := i.Get(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "value"))

	// Second read is served from the local client-side cache.
	val, err = i.Get(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "value"))
	qt.Check(t, qt.IsTrue(hits >= 1))

	// External write invalidates the locally cached value.
	con := c.Connection()
	err = con.Do(ctx, con.B().Set().Key("cctest:key").Value(`"value2"`).Build()).Error()
	qt.Assert(t, qt.IsNil(err))

	for range 40 {
		val, err = i.Get(ctx, "key")
		qt.Assert(t, qt.IsNil(err))

		if val == "value2" {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	qt.Check(t, qt.Equals(val, "value2"))
}

func TestRedisCacheClientCacheDisabledFallback(t *testing.T) {
	cs := getRedisConnStr()
	if cs == "" {
		t.Skip("REDIS_CONNSTR is not set")
	}
	c := New(RedisCache, ConnectionString(cs), ClientCache(false), ClientCacheTTL(time.Minute))
	err := c.Start(context.TODO())
	qt.Assert(t, qt.IsNil(err))
	defer c.Close()

	i, err := Create[string](c, "ccdisabled")
	qt.Assert(t, qt.IsNil(err))

	ctx := context.TODO()

	err = i.Set(ctx, "key", "value")
	qt.Assert(t, qt.IsNil(err))

	val, err := i.Get(ctx, "key")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(val, "value"))
}
