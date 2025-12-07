## Summary

- Added byte-count verification after `io.CopyN` in tar extraction to fail on truncated entries.
- Added the same verification for zip extraction to prevent silent partial files.

## Rationale

- `io.CopyN` returns `io.EOF` when fewer bytes are available; previous logic swallowed the EOF and never checked `written == expected`.
- Truncated or malformed archives could produce incomplete files without error, compromising integrity checks performed after extraction.

## Notes

- Error messages now include the entry name and the shortfall to aid debugging.

