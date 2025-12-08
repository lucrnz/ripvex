## Summary
- Added `nightly-build.yml` workflow to run nightly and on demand.
- Workflow compares GHCR tags/digests for `latest-dev` against the current commit’s tag (short SHA) and triggers the main build-and-release pipeline when they diverge.

## Technical Notes
- Uses GitHub Packages API to list container versions for the repo and find:
  - The version carrying the `latest-dev` tag.
  - The version whose tag contains the current short SHA (pattern `dev-*-<shortsha>`).
- Compares their `metadata.container.digest`; if missing, mismatched, or no tag for the commit, it dispatches `build-and-release.yml` with `pre_release_build=true` and `create_release=true` on the current ref.
- Avoids executing the image (`--version`) and relies solely on registry metadata (tags/digests). Keeps permissions minimal: read contents/packages, write actions.

