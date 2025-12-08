## Configurable progress interval

- Added CLI flag `--progress-interval` with default `500ms`; accepts human-readable durations via `util.ParseDuration`.
- Downloader now carries `ProgressInterval` through options and passes it to `downloadWithProgress`.
- Progress throttling uses the configured interval; values <=0 are guarded by a fallback to 500ms to avoid spamming.

