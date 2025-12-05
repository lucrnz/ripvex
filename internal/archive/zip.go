package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"simple-downloader/internal/util"
)

const maxSymlinkTarget = 4 * 1024

// extractZip extracts a ZIP archive with zip slip protection
func extractZip(path string, opts ExtractOptions) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	destDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	var extracted int64

	for _, f := range r.File {
		if err := extractZipFile(f, destDir, opts, &extracted); err != nil {
			return err
		}
	}

	return nil
}

// extractZipFile extracts a single file from a ZIP archive
func extractZipFile(f *zip.File, destDir string, opts ExtractOptions, extracted *int64) error {
	// Apply strip-components
	name := util.StripPathComponents(f.Name, opts.StripComponents)
	if name == "" {
		return nil // Skip entries that are entirely stripped
	}

	// Zip slip protection
	destPath := filepath.Join(destDir, name)
	if !util.IsPathSafe(destPath, destDir) {
		return fmt.Errorf("zip slip detected: %s", name)
	}

	// Handle directories
	if f.FileInfo().IsDir() {
		return os.MkdirAll(destPath, 0755)
	}

	// Handle symlinks
	if f.FileInfo().Mode()&os.ModeSymlink != 0 {
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open symlink entry: %w", err)
		}
		defer rc.Close()

		lr := io.LimitReader(rc, maxSymlinkTarget+1)
		linkTarget, err := io.ReadAll(lr)
		if err != nil {
			return fmt.Errorf("failed to read symlink target: %w", err)
		}
		if len(linkTarget) > maxSymlinkTarget {
			return fmt.Errorf("symlink target too long (limit %d bytes)", maxSymlinkTarget)
		}

		// Apply strip-components to relative symlink targets
		linkname := string(linkTarget)
		if !filepath.IsAbs(linkname) {
			linkname = util.StripPathComponents(linkname, opts.StripComponents)
			if linkname == "" {
				return nil // Skip symlinks with invalid targets after stripping
			}
		}

		// Validate symlink target doesn't escape
		targetPath := filepath.Join(filepath.Dir(destPath), linkname)
		if !util.IsPathSafe(targetPath, destDir) {
			return fmt.Errorf("symlink escape detected: %s -> %s", name, linkname)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for symlink: %w", err)
		}

		if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing path for symlink: %w", err)
		}

		return os.Symlink(linkname, destPath)
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Enforce extraction size limit using uncompressed size
	fileSize := int64(f.UncompressedSize64)
	if opts.MaxBytes > 0 && *extracted+fileSize > opts.MaxBytes {
		return fmt.Errorf("extraction exceeded maximum size limit of %s", util.HumanReadableBytes(opts.MaxBytes))
	}

	// Extract file
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip entry: %w", err)
	}
	defer rc.Close()

	outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, rc)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	*extracted += written
	if opts.MaxBytes > 0 && *extracted > opts.MaxBytes {
		os.Remove(destPath)
		return fmt.Errorf("extraction exceeded maximum size limit of %s", util.HumanReadableBytes(opts.MaxBytes))
	}

	return nil
}
