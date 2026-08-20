# Azugo Core

Azugo framework core.

## Features

* Structured logger [go.uber.org/zap](https://github.com/uber-go/zap)
* Extendable configuration [viper](https://github.com/spf13/viper) and command line [cobra](https://github.com/spf13/cobra) support
* Caching using memory or Redis (Redis 6.2 or later, or any Valkey version)
* Logger based on [zap](go.uber.org/zap) with output compatible with ECS

## Special Environment variables used by the Azugo framework

### Core

* `ENVIRONMENT` - An App environment setting (allowed values are `development`, `test`, `staging` and `production`).
* `LOG_TYPE` - Log type (defaults to `console`, allowed values are `console`, `file` or other registered log drivers).
* `LOG_LEVEL` - Minimal log level (defaults to `info`, allowed values are `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal`).
* `LOG_FORMAT` - Log output format (defaults to `console` in development environment and `ecsjson` in staging and production).
* `LOG_OUTPUT` - Log output location (defaults to `stderr`, allowed values are `stderr`, `stdout`, file path or `file://` URL and other values supported by registered log drivers)
* `LOG_STACKTRACE` - Enable stack traces for error level and above regardless of environment (defaults to `false`).
* `LOG_TYPE_SECONDARY` - Secondary log type (see `LOG_TYPE`)
* `LOG_LEVEL_SECONDARY` - Secondary log level (defaults to `info`, see `LOG_LEVEL`)
* `LOG_FORMAT_SECONDARY` - Secondary log format (see `LOG_FORMAT`)
* `LOG_OUTPUT_SECONDARY` - Secondary log output location (See `LOG_OUTPUT`)

### Cache

Redis-backed cache types require Redis 6.2 or later, or any Valkey version.

* `CACHE_TYPE` - Cache type to use in service (defaults to `memory`, allowed values are `memory`, `redis`, `redis-cluster`, `redis-sentinel`).
* `CACHE_TTL` - Duration on how long to keep items in cache. Defaults to 0 meaning to never expire.
* `CACHE_KEY_PREFIX` - Prefix all cache keys with specified value.
* `CACHE_CONNECTION` - If other than memory cache is used specifies connection string on how to connect to cache storage. Use the `rediss://` scheme to connect over TLS and add `skip_verify=true` to skip server certificate verification. `skip_verify=true` on a plain `redis://` connection string is rejected, since it does not select the transport.
* `CACHE_PASSWORD` - Password to use in connection string.
* `CACHE_PASSWORD_FILE` - File to read value for `CACHE_PASSWORD` from.
* `CACHE_CLIENT_CACHE` - Enable server-assisted client-side caching support on Redis connections (defaults to `true`). Set to `false` for providers without `CLIENT TRACKING` support (e.g. Google Cloud Memorystore).
* `CACHE_CLIENT_CACHE_TTL` - Default duration to keep values in the in-process client-side cache for all cache instances (defaults to `0` meaning disabled). Individual cache instances can override it with the `cache.ClientCacheTTL` option.
* `CACHE_CLIENT_CACHE_SIZE` - Client-side cache memory in bytes per Redis connection (defaults to `16777216` - 16 MiB).

#### Redis Sentinel Connection String Format

When using `redis-sentinel` as the cache type, the connection string should be formatted as:

```
sentinel://[username@]host1:port,host2:port,host3:port/masterName?db=0
```

Where:

* `username` - Optional username for Redis authentication
* `host1:port,host2:port,host3:port` - Comma-separated list of Redis Sentinel addresses
* `masterName` - The name of the Redis master in the Sentinel configuration
* `db=0` - Optional database number (defaults to 0)

Use the `sentinels://` scheme to connect to sentinels and the master over TLS. Add
`skip_verify=true` to skip server certificate verification (only applies with `sentinels://`).

Example:

```bash
CACHE_TYPE: "redis-sentinel"
CACHE_CONNECTION: "sentinel://admin@redis-sentinel1:26379,redis-sentinel2:26379,redis-sentinel3:26379/mymaster?db=0"
CACHE_PASSWORD_FILE: /secret/redis-password
CACHE_KEY_PREFIX: "my-service"
```
