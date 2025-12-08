## Structured logging and progress reporting

- Added slog-based logger with `--log-level`, `--log-format` (text/json), and milestone progress logging via `--log-progress-step`.
- Quiet mode now forces log level `error`; visual progress bar removed, only structured logs remain. JSON mode continues to emit structured events.
- Progress logs include both raw bytes (`downloaded_bytes`, `total_bytes`) and human-readable fields (`downloaded_hr`, `total_hr`) for better readability.
- Progress logging operates on two schedules:
  - Interval-based: logs every `--progress-interval` (default `500ms`) regardless of milestones
  - Milestone-based: logs at percentage intervals (`--log-progress-step`, default 5%) when size is known, or byte intervals (`--log-progress-step-unknown`, default `25MB`) when size is unknown
- Both interval and milestone logs can coexist; milestone logs provide coarse-grained checkpoints while interval logs provide regular updates.
- Downloader emits structured progress milestones; for unknown sizes it logs every `--log-progress-step-unknown` bytes (default `25MB`). Hash results and cleanup warnings are logged with levels.
- CLI and cleanup packages route messages through the shared logger, removing direct stderr writes.

