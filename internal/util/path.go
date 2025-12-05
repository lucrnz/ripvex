package util

import (
	"path/filepath"
	"strings"
)

// StripPathComponents removes n leading path components from a path.
// Returns empty string if n >= number of components (file should be skipped).
func StripPathComponents(path string, n int) string {
	if n <= 0 {
		return path
	}
	parts := strings.Split(filepath.ToSlash(path), "/")
	if n >= len(parts) {
		return ""
	}
	return filepath.FromSlash(strings.Join(parts[n:], "/"))
}

// IsPathSafe checks if a path is safely within the destination directory (zip slip protection)
func IsPathSafe(path, destDir string) bool {
	// Clean and resolve the path
	cleanPath := filepath.Clean(path)
	cleanDest := filepath.Clean(destDir)

	// Check if the path starts with the destination directory
	if !strings.HasPrefix(cleanPath, cleanDest+string(filepath.Separator)) && cleanPath != cleanDest {
		return false
	}

	return true
}

