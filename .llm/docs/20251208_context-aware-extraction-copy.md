## Context-aware extraction copy

- Added `copyWithContext` in `internal/archive/copy.go` to stream entry data in chunks and check `ctx.Err()` periodically.
- Replaced `io.CopyN` with `copyWithContext` in `internal/archive/extract.go` and `internal/archive/zip.go`, enabling `--extract-timeout` and SIGINT to abort mid-file.
- Aligned extraction cancellation behavior with downloader logic while keeping size-limit checks and incomplete-entry validation unchanged.

