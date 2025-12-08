# Fix Symlink Target Corruption with Strip-Components

**Date:** 2025-12-08

## Summary

Fixed a bug where `--extract-strip-components` incorrectly modified relative symlink targets, causing symlinks to point to non-existent paths.

## The Bug

When extracting archives with `--extract-strip-components=N`, ripvex was applying the strip-components logic to symlink targets. This broke symlinks because:

- **Symlink targets** are relative to the symlink's filesystem location
- **Strip-components** operates on archive entry paths (the archive's structure)

These are fundamentally different coordinate systems.

### Example

The pyenv archive contains:
```
pyenv-2.6.15/bin/pyenv -> ../libexec/pyenv  (symlink)
pyenv-2.6.15/libexec/pyenv                  (actual file)
```

With `--extract-strip-components=1`:

| Tool | Symlink | Target | Resolves To |
|------|---------|--------|-------------|
| GNU tar (correct) | `bin/pyenv` | `../libexec/pyenv` | `libexec/pyenv` |
| ripvex (broken) | `bin/pyenv` | `libexec/pyenv` | `bin/libexec/pyenv` (doesn't exist) |

The buggy code stripped `../` from the target, transforming `../libexec/pyenv` into `libexec/pyenv`.

## Root Cause

Both `internal/archive/extract.go` (tar) and `internal/archive/zip.go` contained:

```go
if !filepath.IsAbs(linkname) {
    linkname = util.StripPathComponents(linkname, opts.StripComponents)
    // ...
}
```

This logic incorrectly assumed symlink targets follow the same structure as archive entry paths.

## The Fix

Removed the strip-components transformation from symlink targets in both tar and zip extraction. The security validation (`IsPathSafe`) continues to correctly validate resolved symlink targets against the destination directory.

### What Should Be Stripped

| Entry Type | Strip Components? | Reason |
|------------|-------------------|--------|
| File paths | Yes | Archive structure paths |
| Directory paths | Yes | Archive structure paths |
| Hard link targets | Yes | Reference archive entry paths |
| **Symlink targets** | **No** | Relative to symlink's location, not archive root |

## Files Changed

- `internal/archive/extract.go` - Removed symlink target stripping in tar extraction
- `internal/archive/zip.go` - Removed symlink target stripping in zip extraction
- `AGENTS.md` - Clarified strip-components behavior for symlinks

