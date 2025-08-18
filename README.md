
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

## Goroutines and callsite reporting

By design it's not possible, in general, to reliably determine the exact source
location (file:function:line) where a goroutine was created from inside the
goroutine itself. The reliable approach is to capture the callsite at the
moment the goroutine is spawned and attach it to subsequent log calls.

This package provides a small helper to do that conveniently:

- `g := log.WrapGoroutine()` — call this in the goroutine creator, then pass
    `g` (a `*GoroutineLogger`) into the goroutine. Inside the goroutine use
    `g.Info(...) / g.Infof(...)` etc. The helper captures the creation
    callsite and automatically appends the `goroutine_caller` extra to log lines.

Example:

```go
// creator
g := log.WrapGoroutine()
go func(lg *wslogger.GoroutineLogger) {
        defer wg.Done()
        lg.Info("started worker")
}(g)
```

Notes & limitations:

- `WrapGoroutine()` is best-effort: it tries to locate the `go` statement and
    format the callsite as `basename:function:line`. In some cases (anonymous
    closures, generated function names) the runtime will expose names like
    `func1`; using `WrapGoroutine()` gives the most reliable, consistent
    information because it captures the creator location at spawn time.
- If you prefer not to change call sites, you can also manually pass
    `"goroutine_caller", "file:func:line"` as an extra argument on log calls,
    but that requires the creator to construct the string.

