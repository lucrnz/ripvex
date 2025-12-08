## Tag latest images by release type

- Added conditional manifest tagging in `.github/workflows/build-and-release.yml` so pre-release builds push `latest-dev` and production builds push `latest`.
- Keeps versioned tag intact while providing stable tags for automation and users who want the most recent dev vs prod image.

