package archive

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

// isTarContent peeks at the first 262 bytes to check for tar magic bytes.
// Returns (isTar, reader) where reader is a new reader that includes the peeked bytes.
func isTarContent(r io.Reader) (bool, io.Reader) {
	peekBuf := make([]byte, 262)
	n, err := io.ReadFull(r, peekBuf)
	
	// Handle the case where we read fewer than 262 bytes
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			// We read some bytes but not enough to check tar magic
			peekBuf = peekBuf[:n]
			return false, io.MultiReader(bytes.NewReader(peekBuf), r)
		}
		// Some other error - assume not tar
		return false, io.MultiReader(bytes.NewReader(peekBuf[:n]), r)
	}

	// We read exactly 262 bytes, check for tar magic: "ustar" at offset 257
	ustar := string(peekBuf[257:262])
	if ustar == "ustar" {
		return true, io.MultiReader(bytes.NewReader(peekBuf), r)
	}

	// Not a tar archive
	return false, io.MultiReader(bytes.NewReader(peekBuf), r)
}

// extractGzipTar extracts a .tar.gz archive
func extractGzipTar(path string, opts ExtractOptions) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	isTar, reader := isTarContent(gzr)
	if !isTar {
		return fmt.Errorf("gzip file does not contain a tar archive")
	}

	return extractTar(reader, opts)
}

// extractBzip2Tar extracts a .tar.bz2 archive
func extractBzip2Tar(path string, opts ExtractOptions) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	bzr := bzip2.NewReader(f)
	isTar, reader := isTarContent(bzr)
	if !isTar {
		return fmt.Errorf("bzip2 file does not contain a tar archive")
	}

	return extractTar(reader, opts)
}

// extractXzTar extracts a .tar.xz archive
func extractXzTar(path string, opts ExtractOptions) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	xzr, err := xz.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create xz reader: %w", err)
	}

	isTar, reader := isTarContent(xzr)
	if !isTar {
		return fmt.Errorf("xz file does not contain a tar archive")
	}

	return extractTar(reader, opts)
}

// extractZstdTar extracts a .tar.zstd archive
func extractZstdTar(path string, opts ExtractOptions) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	zstdr, err := zstd.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer zstdr.Close()

	isTar, reader := isTarContent(zstdr)
	if !isTar {
		return fmt.Errorf("zstd file does not contain a tar archive")
	}

	return extractTar(reader, opts)
}


