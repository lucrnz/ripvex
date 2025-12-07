# Rename --max-time to --download-max-time and Set Sensible Defaults

## Changes

1. Renamed `--max-time` flag to `--download-max-time`
2. Changed `--download-max-time` default from `0` (unlimited) to `1h`
3. Changed `--extract-timeout` default from `0` (unlimited) to `30m`

## Rationale

### Flag Rename

The original `--max-time` name was misleading because it only controlled the download phase, not the entire operation (which also includes extraction). The new name `--download-max-time` clearly indicates its scope. The `-m` shorthand was retained for curl-like ergonomics.

### Default Values

**`--download-max-time` = 1h**

With the default `--max-bytes` of 4GiB:
- At 10 Mbps: ~55 minutes (within limit)
- At 100 Mbps: ~5-6 minutes (well within limit)
- At 5 Mbps: ~110 minutes (would need explicit override)

The 1-hour default covers most CI/CD and modern network scenarios while preventing indefinite hangs. Users on slower connections downloading large files can override as needed.

**`--extract-timeout` = 30m**

Archive extraction is primarily I/O-bound, not CPU-bound. Even multi-gigabyte archives typically decompress in minutes on reasonable hardware. A 30-minute default is generous for nearly all practical scenarios while still providing protection against pathological cases (e.g., zip bombs, deeply nested structures).

## Files Modified

- `internal/cli/root.go`: Variable rename, flag definition, error message
- `README.md`: Downloader and Archive Extractor flag tables
- `AGENTS.md`: HTTP Client Configuration and CLI Flags sections

