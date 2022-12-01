# Azugo Core

Azugo framework core.

## Features

* Structured logger [go.uber.org/zap](https://github.com/uber-go/zap)
* Extendable configuration [viper](https://github.com/spf13/viper) and command line [cobra](https://github.com/spf13/cobra) support
* Caching using memory or Redis
* Logger based on [zap](go.uber.org/zap) with output compatible with ECS

## Special Environment variables used by the Azugo framework

* `ENVIRONMENT` - An App environment setting (allowed values are `Development`, `Staging` and `Production`).
