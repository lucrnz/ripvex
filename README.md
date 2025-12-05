# simple-downloader
A lightweight Go program for downloading files from URLs with optional hash integrity verification and archive extraction.
Designed for simplicity and easy embedding in Docker containers or CI/CD pipelines.

It supports progress reporting, human-readable byte sizes, and quiet mode for non-interactive environments.

## Disclaimer

This project is a work in progress, even though the developer makes the best effort to keep it production-ready, it does not come with a warranty on its reliability, stability and/or security.

## Features

- **Download with Progress**: Real-time progress bar showing percentage and human-readable bytes (e.g., "1.2 MB / 5.0 GB"), updated every 500ms to prevent output spam.
- **Hash Verification**: Optional hash check against the downloaded file using SHA-256 or SHA-512—exits with code 1 on mismatch for easy CI integration. Hash values must be prefixed with the algorithm (e.g., `sha256:xxxxx...` or `sha512:xxxxx...`). When outputting to stdout (`--output -`) with hash verification, the file is stored in a temporary location, verified, and only written to stdout if the hash matches.
- **Archive Extraction**: Extract downloaded archives automatically. Supports zip, tar, tar.gz, tar.bz2, tar.xz, and tar.zstd formats.
- **Magic Byte Detection**: Archive format detection uses file magic bytes, not extensions, for reliable format identification.
- **Zip Slip Protection**: Production-ready security against path traversal attacks in archives.
- **Redirect Handling**: Automatically follows HTTP redirects.
- **Quiet Mode**: Suppress all non-error output for scripts or logs.
- **Flexible Output**: Write to file (default: URL basename) or stdout (`--output -`).
- **Clean Piping**: All status messages (progress, hash verification, final messages) are written to stderr, keeping stdout clean for data piping.
- **Working Directory**: Change to a specific directory before any operation with `--chdir`.

## Usage
```sh
simple-downloader [flags]
```

Run `simple-downloader --help` for full options.

### Flags

#### Global

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--chdir` | `-C` | Change working directory before any operation. Panics if directory doesn't exist. | None |
| `--quiet` | `-q` | Suppress progress and final messages (ideal for CI/CD). Errors still printed to stderr. | `false` |

#### Downloader

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--url` | `-U` | **Required**: The URL to download (e.g., `https://example.com/file.zip`). | None |
| `--output` | `-O` | Output file path. Use `-` for stdout. Defaults to the URL's basename (or `download` if none). | URL basename |
| `--hash` | `-H` | Expected hash with algorithm prefix (e.g., `sha256:xxxxx...` or `sha512:xxxxx...`). Supported algorithms: `sha256` (64 hex chars), `sha512` (128 hex chars). Case-insensitive. Verifies file integrity; exits 1 on mismatch. In quiet mode, no success message. When used with `--output -`, the file is buffered in memory and only written to stdout after successful verification. Legacy format (hash without prefix) is deprecated and will emit a warning, defaulting to SHA-256. | None |
| `--connect-timeout` | | Maximum time for connection establishment. | `300s` |
| `--max-time` | `-m` | Maximum total time for the entire operation (0 = unlimited). | `0` |
| `--max-bytes` | `-M` | Maximum bytes to download (supports `k/K/KB/KiB`, `m/M/MB/MiB`, `g/G/GB/GiB`). | `4GiB` |

#### Archive Extractor

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--extract-archive` | `-x` | Extract the downloaded archive. Format auto-detected via magic bytes. | `false` |
| `--remove-archive` | | Delete archive file after successful extraction. | `true` |
| `--extract-strip-components` | | Strip N leading components from file names during extraction. | `0` |
| `--extract-max-bytes` | | Maximum total bytes to extract from the archive. Supports the same units as `--max-bytes`. | `8GiB` |

### Supported Archive Formats

- ZIP
- TAR
- GZIP (tar.gz) 
- BZIP2 (tar.bz2)
- XZ (tar.xz)
- ZSTD (tar.zstd)

### Examples

Download and extract a tarball:
```sh
simple-downloader -U https://example.com/archive.tar.gz -x
```

Download to a specific directory and extract:
```sh
simple-downloader -U https://example.com/release.zip -C /opt/app -x
```

Download with hash verification and quiet mode:
```sh
simple-downloader -U https://example.com/file.tar.xz -H sha256:abc123... -x -q
```

Download with SHA-512 hash verification:
```sh
simple-downloader -U https://example.com/file.tar.xz -H sha512:def456... -x
```

Download with an explicit limit (recommended for CI/CD):
```sh
simple-downloader -U https://example.com/file.bin -M 2GiB
```

Keep the archive after extraction:
```sh
simple-downloader -U https://example.com/data.tar.gz -x --remove-archive=false
```

Download to stdout with hash verification (buffered):
```sh
simple-downloader -U https://example.com/file.bin -O - -H sha256:abc123... | process-file
```

## Output Behavior

### Stdout vs Stderr
- **stdout**: Contains only the downloaded file data (when using `--output -`)
- **stderr**: Contains all status messages (progress, hash verification results, final messages, archive extraction logs)

This design ensures clean piping: `simple-downloader -U url -O - | other-tool` will only pass file data to the next command.

### Hash Algorithm Prefix
Hash values must be prefixed with the algorithm name followed by a colon:
- `sha256:` for SHA-256 (64 hex characters)
- `sha512:` for SHA-512 (128 hex characters)

Examples:
- `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
- `sha512:cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e`

**Deprecation**: Providing a hash without a prefix (legacy format) will emit a deprecation warning and default to SHA-256. This behavior may be removed in a future version.

## License
MIT License. See [LICENSE](https://github.com/lucrnz/software-distillery/blob/main/LICENSE) for details.
