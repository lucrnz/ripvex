## Summary
- Expanded `AGENTS.md` to cover the cleanup tracker package, signal-aware cancellation, Content-Disposition filename resolution, and mutual exclusivity of auth flags.
- Documented TLS minimum/version override, proxy env support, CLI defaults for size/time limits, and additional flags (`--chdir-create`, `--user-agent`, `--allow-insecure-tls`).
- Added Go version requirement and indirect dependency notes to keep guidance aligned with `go.mod`.

## Rationale
- The codebase already enforces cleanup via `cleanup.Tracker`, context cancellation via `signal.NotifyContext`, and filename detection from `Content-Disposition`; documenting these helps future changes stay consistent.
- Security and networking behaviors (TLS floor, proxy envs) and default limits are critical for users and CI consumers; surfacing them reduces surprises.
- Auth flag mutual exclusion and new flags improve usability; explicitly noting them in agent guidance prevents conflicting edits or regressions.

