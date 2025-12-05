package version

import "fmt"

var (
	VersionPrefix = "dev"     // Set via -ldflags
	VersionDate   = "edge"    // Set via -ldflags - Value should be: YYYYMMDD
	CommitHash    = "unknown" // Set via -ldflags
	CurlVersion   = "8.17.0"  // Default fallback, can be set via -ldflags
)

// UserAgent returns the default user agent string with embedded commit hash
func UserAgent() string {
	return "curl/" + CurlVersion + " ripvex/" + Print()
}

// Print returns the version information
func Print() string {
	return fmt.Sprintf(`%s-%s-%s`, VersionPrefix, VersionDate, CommitHash)
}
