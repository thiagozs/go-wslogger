# Changelog

All notable changes to this project will be documented in this file.

## [v0.1.0] - 2025-08-18

### Added

- Documented `WrapGoroutine()` helper in README with usage example and limitations.

### Changed

- Reorganized and cleaned up `logger.go` implementation; removed unused helper functions
  and tightened imports.

### Fixed

- Tests updated and validated; `go test ./...` passes.

### Notes

- The `WrapGoroutine()` helper captures the goroutine creation callsite so logs
  created inside goroutines can surface a reliable `goroutine_caller` field. See
  the README for usage and limitations (anonymous closures may still show names
  like `func1` depending on the runtime).

PR: #4 — cleanup: tidy logger implementation and docs
