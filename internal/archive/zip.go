package archive

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lucrnz/ripvex/internal/cleanup"
	"github.com/lucrnz/ripvex/internal/util"
)

const maxSymlinkTarget = 4 * 1024

// extractZip extracts a ZIP archive with zip slip protection
func extractZip(ctx context.Context, tracker *cleanup.Tracker, path string, opts ExtractOptions) error {
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
		// Check for cancellation before processing each entry
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := extractZipFile(ctx, tracker, f, destDir, opts, &extracted); err != nil {
			return err
		}
	}

	return nil
}

// extractZipFile extracts a single file from a ZIP archive
func extractZipFile(ctx context.Context, tracker *cleanup.Tracker, f *zip.File, destDir string, opts ExtractOptions, extracted *int64) error {
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

		if err := os.Symlink(linkname, destPath); err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
		// Register symlink for cleanup
		if tracker != nil {
			tracker.Register(destPath)
		}
		return nil
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
	// Register file for cleanup immediately after creation
	if tracker != nil {
		tracker.Register(destPath)
	}

	written, err := io.CopyN(outFile, rc, fileSize)
	if err == io.EOF {
		err = nil // CopyN returns EOF when source has fewer bytes than limit
	}
	if written != fileSize {
		outFile.Close()
		return fmt.Errorf("incomplete file %s: wrote %d of %d bytes", name, written, fileSize)
	}
	if closeErr := outFile.Close(); closeErr != nil {
		if err == nil {
			return fmt.Errorf("failed to close file: %w", closeErr)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	*extracted += written
	if opts.MaxBytes > 0 && *extracted > opts.MaxBytes {
		os.Remove(destPath)
		if tracker != nil {
			tracker.Unregister(destPath)
		}
		return fmt.Errorf("extraction exceeded maximum size limit of %s", util.HumanReadableBytes(opts.MaxBytes))
	}

	// Preserve executable bit if set in archive
	if f.FileInfo().Mode()&0111 != 0 {
		if err := os.Chmod(destPath, 0755); err != nil {
			return fmt.Errorf("failed to set executable permission: %w", err)
		}
	}

	return nil
}
