### Prevent archive extraction escape via symlink ancestors

**Date:** 2025-12-08

#### What changed
- Added `util.ResolvePathWithinBase`, which walks each path segment, resolves existing symlinks (Lstat+Readlink), and rejects paths that would escape the extraction root. It tolerates non-existent tail components so future writes are still validated.
- Tar and zip extraction now:
  - Resolve the destination root with `EvalSymlinks`.
  - Validate every entry path with `ResolvePathWithinBase` before creation.
  - Validate symlink targets and hard-link targets with the same helper.
  - Defered hard links revalidate both dest and target.

#### Why
Archives could plant a symlink inside the destination (or rely on a preexisting symlink) and later entries would follow it, writing outside the extraction root. The previous check only validated the final joined path string, not symlink ancestors.

#### Files touched
- `internal/util/path.go` — new resolver with symlink walking and escape detection.
- `internal/archive/extract.go` — tar extraction path/symlink/hard-link validation updated.
- `internal/archive/zip.go` — zip extraction path/symlink validation updated.

