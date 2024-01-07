# go-wslogger - wrapper slog

`wslogger` is a versatile logging package for Go, providing an easy-to-use interface for structured logging with support for different output formats and log levels. It is built on top of the `slog` package and offers additional features like buffered logging and group-based logging. 

* ***(working in progress)***

## Features

- Support for multiple log formats: JSON, text, etc.
- Log level control: Info, Debug, Error, Warn.
- Buffered logging for improved performance.
- Group-based logging to categorize log messages.
- Flexible configuration with various output options: Stdout, File.

## Installation

To install `wslogger`, use `go get`:

```bash
go get github.com/thiagozs/go-wslogger
```

## Usage

Here is a simple example of how to use `wslogger`:

```go
package main

import (
    "github.com/yourusername/wslogger"
)

func main() {
    opts := wslogger.Options{}
    logger, err := wslogger.NewWsLogger(opts...)
    if err != nil {
        panic(err)
    }

    logger.Info("Application started")
    logger.WithGroup("database").Debug("Database connection established")
}
```

## Configuration

You can customize `WsLogger` by passing various options:

```go
logger, err := wslogger.NewWsLogger(
    wslogger.WithKind(wslogger.File),
    wslogger.WithFileName("app.log"),
)
```

## Versioning and license

Our version numbers follow the [semantic versioning specification](http://semver.org/). You can see the available versions by checking the [tags on this repository](https://github.com/thiagozs/go-wslogger/tags). For more details about our license model, please take a look at the [LICENSE.md](LICENSE.md) file.

2024, thiagozs
