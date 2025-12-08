# AGENTS.md

This file provides guidance to AI agents when working with code in this repository.

## Project Overview

ripvex is a lightweight Go CLI tool for downloading files from URLs with optional hash verification and archive extraction. It's designed for simplicity and easy embedding in Docker containers and CI/CD pipelines.

## Build and Development Commands

### Building
```bash
make build          # Build to build/ripvex with dev version
make clean          # Remove build artifacts
```

Minimum Go version: 1.25.5 (per go.mod)

Build with custom version info:
```bash
VERSION_PREFIX=v1.0 VERSION_DATE=20250101 make build
```

### Running
```bash
./build/ripvex --help
./build/ripvex -U https://example.com/file.tar.gz -x
```

### Testing
Currently no test files exist in the codebase.

### Formatting
The project uses a pre-commit hook to auto-format Go files with `gofmt`. To enable:
```bash
git config core.hooksPath .githooks
```

Manually format code:
```bash
gofmt -w .
```

## Architecture

### Project Structure

The codebase follows a standard Go CLI application structure:

- **cmd/ripvex/main.go**: Entry point that delegates to the CLI package
- **internal/cli/**: Cobra-based command line interface and orchestration logic
- **internal/downloader/**: HTTP download logic with progress reporting and hash verification
- **internal/archive/**: Archive detection (magic bytes) and extraction with security protections
- **internal/util/**: Shared utilities (size parsing, path safety, formatting)
- **internal/cleanup/**: Cleanup tracker for temporary files and graceful interrupt handling
- **internal/version/**: Version information injected at build time via ldflags

### Key Design Patterns

**1. Separation of Concerns**
- CLI layer (internal/cli/) handles argument parsing and orchestrates the workflow
- Downloader handles HTTP operations, progress reporting, and hash verification
- Archive package handles format detection and extraction independently
- Each package has a focused responsibility

**2. Magic Byte Detection**
Archive format detection (internal/archive/detect.go) uses file magic bytes rather than file extensions for reliable format identification. Reads first 262 bytes to identify:
- ZIP: PK\x03\x04
- GZIP: \x1f\x8b
- BZIP2: BZh
- XZ: \xFD7zXZ\x00
- ZSTD: \x28\xB5\x2F\xFD
- TAR: "ustar" at offset 257

**3. Security Protections**
- Zip slip protection: All extracted paths validated via util.IsPathSafe() before writing
- Size limits: Both download (--max-bytes) and extraction (--extract-max-bytes) have configurable limits
- Hash verification: Supports sha256 and sha512 with algorithm prefix (e.g., sha256:abc123...)
- Path traversal prevention for symlinks and hard links in archives

**4. Stdout vs Stderr**
Critical design principle: stdout contains ONLY file data when using --output -, while stderr contains all status messages (progress, hash verification, logs). This enables clean piping: `ripvex -U url -O - | other-tool`

**5. Hash Verification with Stdout**
When outputting to stdout with hash verification (--output - --hash sha256:xxx), the file is downloaded to a temp file, verified, then streamed to stdout only if hash matches. This prevents corrupted data from reaching the pipe.

**6. Extensible Hash Algorithm Support**
Hash algorithms defined in a registry pattern (supportedHashes map in internal/cli/root.go) making it easy to add blake3, sha3, etc. Each algorithm has:
- name: Display name (e.g., "SHA-256")
- digestLen: Expected hex character length
- newHash: Constructor function

**7. Strip Components**
The objective of this feature is to be exactly like GNU tar.

Like GNU tar's --strip-components, the --extract-strip-components flag removes N leading path components during extraction. Applied to file paths and hard link targets. **Symlink targets are NOT modified** because they are relative to the symlink's destination location, not the archive root structure.

**8. Cleanup Tracker**
- `cleanup.Tracker` registers files as soon as they are created; downloader and archive extraction unregister them after success. `main` defers `tracker.Cleanup()` to remove temporary files on interrupt or failure.

**9. Signal Handling and Cancellation**
- `cmd/ripvex/main.go` uses `signal.NotifyContext` to handle SIGINT/SIGTERM, propagating cancellation through the CLI to downloader/extraction loops that poll `ctx.Err()`. Exits with code 130 on SIGINT.

**10. Content-Disposition Awareness**
- Downloader resolves filenames from the HTTP `Content-Disposition` header when `--output` is not set, preferring RFC 5987 `filename*` and falling back to `filename` while preventing path traversal.

### HTTP Client Configuration
- Connection timeout: --connect-timeout (default 300s)
- Download timeout: --download-max-time (default 1h)
- Redirect handling: --max-redirs (default 30)
- Custom User-Agent: Built from version info (injected via ldflags)
- TLS security: Minimum TLS 1.2 by default; `--allow-insecure-tls` lowers to TLS 1.0/1.1 for legacy endpoints (use sparingly).
- Proxy support: Honors `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` via `http.ProxyFromEnvironment`.

### CLI Flags and Defaults
- Authorization: `--header`, `--auth` (-A), `--auth-bearer` (-B), `--auth-basic-user/pass`, and `--auth-basic` are mutually exclusive and set the Authorization header in different ways.
- Directory control: `--chdir` changes working directory; `--chdir-create` optionally creates it (requires `--chdir`).
- HTTP identity: `--user-agent` overrides the default versioned User-Agent string.
- TLS compatibility: `--allow-insecure-tls` enables TLS 1.0/1.1 for legacy servers.
- Size/time defaults: `--max-bytes` defaults to 4GiB; `--extract-max-bytes` defaults to 8GiB; `--connect-timeout` defaults to 300s; `--download-max-time` defaults to 1h; `--extract-timeout` defaults to 30m.

### Version Injection
Version info is injected at build time via ldflags in the Makefile:
- CommitHash: Git commit (or "unknown")
- VersionPrefix: e.g., "dev" or "v1.0"
- VersionDate: Build date (YYYYMMDD format)
- CurlVersion: Optional, mimics curl versioning style

### Dependencies
- github.com/spf13/cobra: CLI framework
- github.com/dustin/go-humanize: Human-readable byte sizes
- github.com/klauspost/compress: Zstd compression support
- github.com/ulikunitz/xz: XZ compression support
- Indirect: github.com/inconshreveable/mousetrap, github.com/spf13/pflag (via cobra)

## Coding Conventions

- Static binary compilation: CGO_ENABLED=0 for portability
- All user-facing messages to stderr (except piped data to stdout)
- Error messages follow Go conventions: lowercase, no punctuation at end
- Progress updates throttled to 500ms intervals to prevent output spam
- File permissions: 0755 for directories and executables, 0644 for regular files
- Preserve executable bit from tar archives when extracting

## Documentation Requirements

### Change Log (`.llm/docs/`)
After implementing a new feature or code change, create a documentation file:
- **Location**: `.llm/docs/`
- **Naming**: `YYYYMMDD_title.md` (e.g., `20251206_add-bearer-auth.md`)
- **Content**: Document what was changed and the technical reasoning
- **Skip**: Do not document if the only rationale is "user requested" - focus on technical decisions and context
- This directory serves as a knowledge base for the project's development history

### README Updates
When introducing new CLI flags:
- Update the appropriate flags table in README.md
- Add usage examples demonstrating the new functionality
