package archive

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"simple-downloader/internal/util"
)

// Extract extracts an archive based on its detected type
func Extract(path string, archiveType Type, opts ExtractOptions) error {
	switch archiveType {
	case Zip:
		return extractZip(path, opts)
	case Tar:
		return extractTarFromFile(path, opts)
	case Gzip:
		return extractGzipTar(path, opts)
	case Bzip2:
		return extractBzip2Tar(path, opts)
	case Xz:
		return extractXzTar(path, opts)
	case Zstd:
		return extractZstdTar(path, opts)
	default:
		return fmt.Errorf("unsupported archive type: %s", archiveType)
	}
}

// extractTarFromFile extracts a plain tar archive from a file
func extractTarFromFile(path string, opts ExtractOptions) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open tar file: %w", err)
	}
	defer f.Close()

	return extractTar(f, opts)
}

// extractTar extracts a tar archive from a reader with zip slip protection
func extractTar(r io.Reader, opts ExtractOptions) error {
	destDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	tr := tar.NewReader(r)
	type pendingLink struct {
		destPath   string
		linkTarget string
	}
	var pendingLinks []pendingLink
	var extracted int64

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		// Apply strip-components
		name := util.StripPathComponents(header.Name, opts.StripComponents)
		if name == "" {
			continue // Skip entries that are entirely stripped
		}

		// Zip slip protection
		destPath := filepath.Join(destDir, name)
		if !util.IsPathSafe(destPath, destDir) {
			return fmt.Errorf("tar slip detected: %s", name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			if header.Size < 0 {
				return fmt.Errorf("invalid file size for %s", name)
			}
			if opts.MaxBytes > 0 && extracted+header.Size > opts.MaxBytes {
				return fmt.Errorf("extraction exceeded maximum size limit of %s", util.HumanReadableBytes(opts.MaxBytes))
			}

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			written, err := io.Copy(outFile, tr)
			if err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
			extracted += written
			if opts.MaxBytes > 0 && extracted > opts.MaxBytes {
				os.Remove(destPath)
				return fmt.Errorf("extraction exceeded maximum size limit of %s", util.HumanReadableBytes(opts.MaxBytes))
			}

			// Preserve executable bit if set in archive
			if header.Mode&0111 != 0 {
				if err := os.Chmod(destPath, 0755); err != nil {
					return fmt.Errorf("failed to set executable permission: %w", err)
				}
			}

		case tar.TypeSymlink:
			// Apply strip-components to relative symlink targets
			linkname := header.Linkname
			if !filepath.IsAbs(linkname) {
				linkname = util.StripPathComponents(linkname, opts.StripComponents)
				if linkname == "" {
					continue // Skip symlinks with invalid targets after stripping
				}
			}

			// Validate symlink target doesn't escape
			targetPath := filepath.Join(filepath.Dir(destPath), linkname)
			if !util.IsPathSafe(targetPath, destDir) {
				return fmt.Errorf("symlink escape detected: %s -> %s", name, linkname)
			}

			// Remove existing symlink if present
			os.Remove(destPath)

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for symlink: %w", err)
			}

			if err := os.Symlink(linkname, destPath); err != nil {
				return fmt.Errorf("failed to create symlink: %w", err)
			}

		case tar.TypeLink:
			// Apply strip-components to hard link targets
			linkname := util.StripPathComponents(header.Linkname, opts.StripComponents)
			if linkname == "" {
				continue // Skip hard links with invalid targets after stripping
			}

			// Hard links - validate target exists within destDir
			linkTarget := filepath.Join(destDir, linkname)
			if !util.IsPathSafe(linkTarget, destDir) {
				return fmt.Errorf("hard link escape detected: %s -> %s", name, linkname)
			}

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for hard link: %w", err)
			}

			if _, err := os.Stat(linkTarget); err == nil {
				if err := os.Link(linkTarget, destPath); err != nil {
					return fmt.Errorf("failed to create hard link: %w", err)
				}
			} else if errors.Is(err, os.ErrNotExist) {
				pendingLinks = append(pendingLinks, pendingLink{destPath: destPath, linkTarget: linkTarget})
			} else {
				return fmt.Errorf("failed to stat hard link target: %w", err)
			}
		}
	}

	// Process deferred hard links after all entries have been read
	for _, pl := range pendingLinks {
		if !util.IsPathSafe(pl.destPath, destDir) || !util.IsPathSafe(pl.linkTarget, destDir) {
			return fmt.Errorf("hard link escape detected (deferred): %s -> %s", pl.destPath, pl.linkTarget)
		}
		if err := os.MkdirAll(filepath.Dir(pl.destPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for hard link: %w", err)
		}
		if _, err := os.Stat(pl.linkTarget); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("hard link target not found: %s", pl.linkTarget)
			}
			return fmt.Errorf("failed to stat hard link target: %w", err)
		}
		if err := os.Link(pl.linkTarget, pl.destPath); err != nil {
			return fmt.Errorf("failed to create hard link: %w", err)
		}
	}

	return nil
}
