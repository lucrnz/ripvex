package util

import (
	"fmt"
	"os"
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

// ResolvePathWithinBase walks an absolute path, resolving any existing symlink
// segments while ensuring the resulting location remains within destDir. It
// does not require the final path to already exist; non-existent components
// simply terminate traversal.
func ResolvePathWithinBase(path, destDir string) (string, error) {
	cleanDest := filepath.Clean(destDir)
	cleanPath := filepath.Clean(path)

	if !IsPathSafe(cleanPath, cleanDest) {
		return "", fmt.Errorf("path escapes destination: %s", cleanPath)
	}

	rel, err := filepath.Rel(cleanDest, cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}
	if rel == "." {
		return cleanDest, nil
	}

	parts := strings.Split(rel, string(filepath.Separator))
	resolved := cleanDest
	symlinkDepth := 0

	for i := 0; i < len(parts); i++ {
		part := parts[i]
		resolved = filepath.Join(resolved, part)

		info, err := os.Lstat(resolved)
		if err != nil {
			if os.IsNotExist(err) {
				if i+1 < len(parts) {
					resolved = filepath.Join(resolved, filepath.Join(parts[i+1:]...))
				}
				return resolved, nil
			}
			return "", fmt.Errorf("lstat %s: %w", resolved, err)
		}

		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}

		symlinkDepth++
		if symlinkDepth > 255 {
			return "", fmt.Errorf("too many symlinks while resolving %s", cleanPath)
		}

		linkTarget, err := os.Readlink(resolved)
		if err != nil {
			return "", fmt.Errorf("readlink %s: %w", resolved, err)
		}

		var targetPath string
		if filepath.IsAbs(linkTarget) {
			targetPath = filepath.Clean(linkTarget)
		} else {
			targetPath = filepath.Clean(filepath.Join(filepath.Dir(resolved), linkTarget))
		}

		if !IsPathSafe(targetPath, cleanDest) {
			return "", fmt.Errorf("symlink escape detected: %s -> %s", resolved, targetPath)
		}

		remaining := filepath.Join(parts[i+1:]...)
		if remaining != "" {
			combined := filepath.Join(targetPath, remaining)
			relRemaining, err := filepath.Rel(cleanDest, combined)
			if err != nil || strings.HasPrefix(relRemaining, "..") {
				return "", fmt.Errorf("symlink resolution escapes destination: %s", combined)
			}
			parts = strings.Split(relRemaining, string(filepath.Separator))
			resolved = cleanDest
			i = -1
			continue
		}

		return targetPath, nil
	}

	return resolved, nil
}
