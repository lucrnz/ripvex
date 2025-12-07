## Summary
- Added support for human-readable duration strings on `--max-time`, `--connect-timeout`, and `--extract-timeout` flags
- Integrated `github.com/xhit/go-str2duration/v2` library to extend Go's standard duration parsing with days (`d`) and weeks (`w`) support
- Created `internal/util/duration.go` utility function wrapping the library for consistent parsing across the codebase

## Rationale
- Users can now specify durations in more intuitive formats like `"1h30m"`, `"2d"`, `"1w"` instead of being limited to Go's standard duration format
- The library extends `time.ParseDuration` with days and weeks, which are common units for timeout configurations
- Using a well-maintained library (`go-str2duration/v2`) avoids reinventing duration parsing logic and provides consistent behavior
- The implementation maintains backward compatibility - all existing duration formats (e.g., `"300s"`, `"5m"`) continue to work
- Changed from `DurationVar` to `StringVar` flags to allow parsing of human-readable strings, with validation happening in the `run()` function before use

## Technical Details
- Flags changed from `DurationVar` to `StringVar` to accept string input
- Duration parsing happens in `run()` function after flag parsing, with clear error messages for invalid formats
- Default values remain the same (`"300s"` for connect-timeout, `"0"` for max-time and extract-timeout)
- The library supports all standard Go duration units plus days and weeks: `h`, `m`, `s`, `ms`, `us`, `ns`, `d`, `w`

