# Nightly Build: Fix Duplicate Releases via Commit SHA Labels

**Date:** 2025-12-12

## Problem

The nightly build workflow (`nightly-build.yml`) was repeatedly triggering builds and releases for the same commit, even though a release already existed for that commit hash. The workflow was supposed to compare the `latest-dev` container digest with the current commit's container digest to avoid duplicate releases.

**Root Cause:** Multi-architecture manifest digest instability. When `docker buildx imagetools create` creates a multi-arch manifest list, it generates a **new manifest digest** each time, even if the underlying platform-specific image digests are identical. This is because manifest lists include metadata (ordering, timestamps) that can differ between builds. As a result, the digest comparison would always fail, triggering unnecessary rebuilds.

## Solution

Replaced digest-based comparison with **commit SHA label comparison**. The solution stores the Git commit SHA as an OCI standard label (`org.opencontainers.image.revision`) on container images, then compares this label directly instead of relying on digest values.

## Technical Changes

### 1. Dockerfile
Added `ARG` and `LABEL` to store the commit SHA:
```dockerfile
FROM scratch
ARG COMMIT_HASH=unknown
LABEL org.opencontainers.image.revision="${COMMIT_HASH}"
```

**Rationale:** Uses the OCI standard label convention for source control revision. This makes the commit SHA intrinsic to the image metadata.

### 2. build-and-release.yml
Updated build step to pass the commit hash as a build argument:
```yaml
build-args: |
  VERSION_PREFIX=${{ needs.prepare.outputs.version_prefix }}
  VERSION_DATE=${{ needs.prepare.outputs.version_date }}
  COMMIT_HASH=${{ github.sha }}
```

**Rationale:** Injects the full Git commit SHA (not the 7-character short version) into the build so it can be stored in the image label.

### 3. nightly-build.yml
Replaced the entire GitHub Packages API-based digest comparison with label-based comparison:
```bash
# Pull latest-dev image
docker pull ghcr.io/${OWNER}/${REPO}:latest-dev

# Extract commit SHA from label
LATEST_COMMIT=$(docker inspect ghcr.io/${OWNER}/${REPO}:latest-dev | \
  jq -r '.[0].Config.Labels["org.opencontainers.image.revision"] // empty')

# Compare with current commit
if [ "$LATEST_COMMIT" = "$CURRENT_SHA" ]; then
  # Commits match, no build needed
fi
```

**Key improvements:**
- **Deterministic comparison:** Git commit SHAs are absolute identifiers
- **No API pagination issues:** Only inspects one image, no need to fetch 100+ versions
- **No digest instability:** Compares source commit, not derived manifest digests
- **Built-in debugging:** Adds debug output to `$GITHUB_STEP_SUMMARY` for visibility
- **Simpler logic:** Direct string comparison instead of complex jq queries

## Migration Behavior

The first nightly build after this change will trigger a build because existing `latest-dev` images don't have the `org.opencontainers.image.revision` label yet. This is expected and ensures the system converges to the correct state.

If the label is missing on old images, the workflow sets `reason=missing_revision_label` and triggers a build as a fallback.

## New Reason Codes

The comparison step now outputs these reason codes:

- `missing_latest_dev`: The latest-dev tag doesn't exist or isn't pullable
- `missing_revision_label`: The latest-dev image exists but has no revision label (old image)
- `commit_matches`: Commit SHAs match, no build needed ✅
- `commit_mismatch`: Commit SHAs differ, build triggered 🔄

## Why This Approach

**Alternatives considered:**
1. **Platform-specific digest comparison:** Compare digests of individual platforms (e.g., linux/amd64) instead of the manifest list. This would work but requires more complex manifest inspection.
2. **API pagination:** Fix the pagination to fetch all versions. Doesn't solve the digest instability problem.
3. **Exact tag matching:** Search for exact version tag (e.g., `dev-20251212-abc1234`). Fragile due to date/time dependencies.

**Why commit SHA labels won:**
- Directly addresses the root cause (digest instability)
- Uses industry-standard OCI label conventions
- Simplest implementation with fewest edge cases
- Eliminates all API-based pagination concerns
- Provides better debugging output

## Testing

After implementation:
1. Manually trigger nightly build workflow → Expected: Triggers build (first time, missing label)
2. Wait for build to complete, trigger again → Expected: Skips build (`reason=commit_matches`)
3. Make a commit, trigger workflow → Expected: Triggers build (`reason=commit_mismatch`)
4. Monitor scheduled runs for several days → Expected: Only builds on code changes
