### Require hash for plain HTTP unless explicitly allowed

**Date:** 2025-12-08

#### What changed
- Added `--allow-unsafe-http` (default false) to bypass the new HTTP safety check.
- Plain HTTP downloads now require a hash unless `--allow-unsafe-http` is set.
- CLI validation fails fast when an HTTP URL is used without integrity protection.

#### Why
- HTTP lacks transport integrity; requiring a hash prevents silent tampering unless the user knowingly opts out.

#### Files touched
- `internal/cli/root.go` — flag registration and HTTP/hash gate.
- `README.md` — documented the default HTTP restriction and new flag.

