# Azugo Core

Azugo framework core.

## Features

* Structured logger [go.uber.org/zap](https://github.com/uber-go/zap)
* Extendable configuration [viper](https://github.com/spf13/viper) and command line [cobra](https://github.com/spf13/cobra) support
* Caching using memory or Redis
* Logger based on [zap](go.uber.org/zap) with output compatible with ECS

## Special Environment variables used by the Azugo framework

### Core

* `ENVIRONMENT` - An App environment setting (allowed values are `Development`, `Staging` and `Production`).
* `LOG_LEVEL` - Minimal log level (defaults to `info`, allowed values are `debug`, `info`, `warn`, `error`, `fatal`, `panic`).

### Cache

* `CACHE_TYPE` - Cache type to use in service (defaults to `memory`, allowed values are `memory`, `redis`, `redis-cluster`).
* `CACHE_TTL` - Duration on how long to keep items in cache. Defaults to 0 meaning to never expire.
* `CACHE_KEY_PREFIX` - Prefix all cache keys with specified value.
* `CACHE_CONNECTION` - If other than memory cache is used specifies connection string on how to connect to cache storage.
* `CACHE_PASSWORD` - Password to use in connection string.
* `CACHE_PASSWORD_FILE` - File to read value for `CACHE_PASSWORD` from.
