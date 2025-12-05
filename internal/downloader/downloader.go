package downloader

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"simple-downloader/internal/util"
)

// Options configures the download behavior
type Options struct {
	URL            string
	Output         string // Output file path, or "-" for stdout
	Quiet          bool
	HashAlgorithm  string        // Hash algorithm name (e.g., "sha256", "sha512")
	ExpectedHash   string        // Hex string to verify against (digest only, without algorithm prefix)
	ConnectTimeout time.Duration // Maximum time for connection establishment
	MaxTime        time.Duration // Maximum total time for the entire operation (0 = unlimited)
	UserAgent      string        // User-Agent header to send with HTTP requests
	MaxBytes       int64         // Maximum allowed download size in bytes (0 = unlimited)
}

// Result contains the outcome of a download
type Result struct {
	BytesDownloaded int64
	HashMatched     bool
}

// Download fetches a URL and writes it to the specified output
func Download(opts Options) (*Result, error) {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: opts.ConnectTimeout,
		}).DialContext,
	}

	client := &http.Client{
		Transport: transport,
	}

	if opts.MaxTime > 0 {
		client.Timeout = opts.MaxTime
	}

	req, err := http.NewRequest("GET", opts.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}

	// Enforce maximum download size by limiting the reader.
	var bodyReader io.Reader = resp.Body
	if opts.MaxBytes > 0 {
		bodyReader = io.LimitReader(resp.Body, opts.MaxBytes+1)
	}

	// Special handling: stdout + hash requires buffering to verify before output
	if opts.Output == "-" && opts.ExpectedHash != "" {
		tempFile, err := os.CreateTemp("", "simple-downloader-*")
		if err != nil {
			return nil, fmt.Errorf("error creating temp file: %w", err)
		}
		tempPath := tempFile.Name()
		defer os.Remove(tempPath)

		result, err := downloadWithProgress(tempFile, bodyReader, resp.ContentLength, opts.Output, opts.Quiet, opts.HashAlgorithm, opts.ExpectedHash, opts.MaxBytes)
		tempFile.Close()
		if err != nil {
			return result, err
		}

		// Hash verification passed, stream temp file to stdout
		tempFile, err = os.Open(tempPath)
		if err != nil {
			return nil, fmt.Errorf("error reopening temp file: %w", err)
		}
		defer tempFile.Close()

		if _, err := io.Copy(os.Stdout, tempFile); err != nil {
			return nil, fmt.Errorf("error writing to stdout: %w", err)
		}
		return result, nil
	}

	// Standard flow: file output or stdout without hash (stream directly)
	var writer io.Writer
	if opts.Output == "-" {
		writer = os.Stdout
	} else {
		file, err := os.Create(opts.Output)
		if err != nil {
			return nil, fmt.Errorf("error creating file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	return downloadWithProgress(writer, bodyReader, resp.ContentLength, opts.Output, opts.Quiet, opts.HashAlgorithm, opts.ExpectedHash, opts.MaxBytes)
}

// newHashFromAlgorithm creates a hash.Hash instance for the given algorithm name
func newHashFromAlgorithm(algo string) (hash.Hash, string, error) {
	algo = strings.ToLower(algo)
	switch algo {
	case "sha256":
		return sha256.New(), "SHA-256", nil
	case "sha512":
		return sha512.New(), "SHA-512", nil
	default:
		return nil, "", fmt.Errorf("unsupported hash algorithm: %s", algo)
	}
}

// downloadWithProgress reads from reader in chunks and writes to writer, showing real-time progress
// throttled to update every 500ms, with optional hash verification
func downloadWithProgress(writer io.Writer, reader io.Reader, total int64, outName string, quiet bool, hashAlgorithm string, expectedHash string, maxBytes int64) (*Result, error) {
	updateInterval := 500 * time.Millisecond
	lastUpdate := time.Now()
	var downloaded int64
	buf := make([]byte, 4096)

	var hasher hash.Hash
	var hashName string
	var err error
	if expectedHash != "" {
		hasher, hashName, err = newHashFromAlgorithm(hashAlgorithm)
		if err != nil {
			return nil, err
		}
	}

	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				return nil, fmt.Errorf("error reading: %w", err)
			}
			break
		}
		if hasher != nil {
			hasher.Write(buf[:n])
		}
		n2, err := writer.Write(buf[:n])
		if err != nil || n2 != n {
			return nil, fmt.Errorf("error writing: %w", err)
		}
		downloaded += int64(n)
		if maxBytes > 0 && downloaded > maxBytes {
			if outName != "-" {
				if err := os.Remove(outName); err != nil && !os.IsNotExist(err) && !quiet {
					fmt.Fprintf(os.Stderr, "\nWarning: failed to remove oversized file %s: %v\n", outName, err)
				}
			}
			return nil, fmt.Errorf("download exceeded maximum size limit of %s", util.HumanReadableBytes(maxBytes))
		}
		if !quiet {
			if time.Since(lastUpdate) >= updateInterval {
				if total <= 0 {
					fmt.Fprintf(os.Stderr, "\rDownloaded: %s...", util.HumanReadableBytes(downloaded))
				} else {
					percent := float64(downloaded) / float64(total) * 100
					fmt.Fprintf(os.Stderr, "\rProgress: %.1f%% (%s/%s)", percent, util.HumanReadableBytes(downloaded), util.HumanReadableBytes(total))
				}
				lastUpdate = time.Now()
			}
		}
	}

	result := &Result{
		BytesDownloaded: downloaded,
		HashMatched:     true,
	}

	// Hash verification
	if expectedHash != "" {
		sum := hasher.Sum(nil)
		computed := hex.EncodeToString(sum)
		if computed != expectedHash {
			result.HashMatched = false
			// Delete corrupted file if writing to a file (not stdout)
			if outName != "-" {
				if err := os.Remove(outName); err != nil && !os.IsNotExist(err) {
					if !quiet {
						fmt.Fprintf(os.Stderr, "\nWarning: failed to remove corrupted file %s: %v\n", outName, err)
					}
				}
			}
		if !quiet {
			fmt.Fprintf(os.Stderr, "\n❌ error: invalid %s sum\n", hashName)
		}
		return result, fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, computed)
	}
	if !quiet {
		fmt.Fprintf(os.Stderr, "\n✅ %s sum hash matches\n", hashName)
	}
	}

	// Final message
	if !quiet {
		sizeStr := util.HumanReadableBytes(downloaded)
		if total != -1 {
			sizeStr = util.HumanReadableBytes(total)
		}
		if outName == "-" {
			fmt.Fprintf(os.Stderr, "\nDownloaded %s\n", sizeStr)
		} else {
			fmt.Fprintf(os.Stderr, "\nDownloaded %s to %s\n", sizeStr, outName)
		}
	}

	return result, nil
}
