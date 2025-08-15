
# wslogger

`wslogger` is a flexible and modern logging package for Go, supporting structured logs, multiple formats, log rotation, OpenTelemetry context, and more.

## Features

- Multiple log formats: JSON, text (customizable)
- Log levels: Info, Warn, Error, Debug
- Context-aware logging (OpenTelemetry support)
- Log rotation (via lumberjack)
- Color output (optional)
- Easy configuration via options

## Installation

```bash
go get github.com/thiagozs/go-wslogger
```

## Usage

```go
package main

import (
    "context"
    "github.com/thiagozs/go-wslogger"
)

func main() {
    log := wslogger.NewLogger(
        wslogger.WithAppName("MyApp"),
        wslogger.WithColor(true),
        wslogger.WithFormat("[{time}] [{app_name}] [{level}] {message} {extra}"),
    )

    log.Info("App started")
    log.Warn("Warning message", "user", "john")
    log.Errorf("Error: %s", "something failed")
    log.Debugf("Debug value: %d", 42)

    // Logging with context (OpenTelemetry)
    ctx := context.Background()
    log.InfoCtx(ctx, "Info with context")
    log.ErrorCtxf(ctx, "Error with wrap: %w", fmt.Errorf("original error"))
}
```

## API Highlights

### Main Methods

- `Info(args ...any)`
- `Warn(args ...any)`
- `Error(args ...any)`
- `Debug(args ...any)`
- `Infof(format string, args ...any)`
- `Warnf(format string, args ...any)`
- `Errorf(format string, args ...any)`
- `Debugf(format string, args ...any)`
- `InfoCtx(ctx, args ...any)`
- `WarnCtx(ctx, args ...any)`
- `ErrorCtx(ctx, args ...any)`
- `DebugCtx(ctx, args ...any)`
- `InfoCtxf(ctx, format, args...)`
- `WarnCtxf(ctx, format, args...)`
- `ErrorCtxf(ctx, format, args...)` (supports `%w`)
- `DebugCtxf(ctx, format, args...)`

### Configuration Options

- `WithAppName(name string)`
- `WithColor(enabled bool)`
- `WithFormat(format string)`
- `WithJSON(enabled bool)`
- `WithWriter(w io.Writer)`
- `WithRotatingFile(filename string, maxSizeMB, maxBackups, maxAgeDays int, compress bool)`
- `WithSpanAttributes(enabled bool)`

## Example of Advanced Configuration

```go
log := wslogger.NewLogger(
    wslogger.WithAppName("ServiceX"),
    wslogger.WithColor(false),
    wslogger.WithJSON(true),
    wslogger.WithRotatingFile("service.log", 10, 5, 30, true),
)
```

## Versioning and license

Our version numbers follow the [semantic versioning specification](http://semver.org/). You can see the available versions by checking the [tags on this repository](https://github.com/thiagozs/go-wslogger/tags). For more details about our license model, please take a look at the [LICENSE.md](LICENSE.md) file.

2025, thiagozs
