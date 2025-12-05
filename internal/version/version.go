package version

import "fmt"

var (
	CommitHash  = "unknown" // Set via -ldflags
	CurlVersion = "8.17.0"  // Default fallback, can be set via -ldflags
)

// UserAgent returns the default user agent string with embedded commit hash
func UserAgent() string {
	return "curl/" + CurlVersion + " simple-downloader/" + CommitHash
}

// Print returns the version information string with copyright notice
func Print() string {
	return fmt.Sprintf(`forever-dev-%s`, CommitHash)
}
