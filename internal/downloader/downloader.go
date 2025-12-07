package downloader

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lucrnz/ripvex/internal/cleanup"
	"github.com/lucrnz/ripvex/internal/util"
)

// Options configures the download behavior
type Options struct {
	URL              string
	Output           string // Output file path, or "-" for stdout
	OutputExplicit   bool   // Whether --output was explicitly set by user
	Quiet            bool
	HashAlgorithm    string            // Hash algorithm name (e.g., "sha256", "sha512")
	ExpectedHash     string            // Hex string to verify against (digest only, without algorithm prefix)
	ConnectTimeout   time.Duration     // Maximum time for connection establishment
	MaxTime          time.Duration     // Maximum total time for the entire operation (0 = unlimited)
	MaxRedirects     int               // Maximum number of redirects to follow
	UserAgent        string            // User-Agent header to send with HTTP requests
	MaxBytes         int64             // Maximum allowed download size in bytes (0 = unlimited)
	AllowInsecureTLS bool              // Allow TLS 1.0/1.1 (insecure)
	Headers          map[string]string // Custom HTTP headers to send
}

// Result contains the outcome of a download
type Result struct {
	BytesDownloaded int64
	HashMatched     bool
	OutputFile      string // Final output filename used (for archive extraction)
}

// Download fetches a URL and writes it to the specified output
func Download(ctx context.Context, tracker *cleanup.Tracker, opts Options) (*Result, error) {
	// Check for cancellation before starting
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12, // Secure default
	}
	if opts.AllowInsecureTLS {
		tlsConfig.MinVersion = tls.VersionTLS10
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: opts.ConnectTimeout,
		}).DialContext,
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{
		Transport: transport,
	}

	if opts.MaxTime > 0 {
		client.Timeout = opts.MaxTime
	}

	// Configure redirect handling
	if opts.MaxRedirects >= 0 {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) > opts.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", opts.MaxRedirects)
			}
			return nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", opts.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	}

	// Apply custom headers
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}

	// Extract filename from Content-Disposition header if output was not explicitly set
	finalOutput := opts.Output
	if !opts.OutputExplicit && opts.Output != "-" {
		contentDisposition := resp.Header.Get("Content-Disposition")
		if contentDisposition != "" {
			cdFilename := extractFilenameFromContentDisposition(contentDisposition)
			if cdFilename != "" {
				finalOutput = cdFilename
			}
		}
	}

	// Enforce maximum download size by limiting the reader.
	var bodyReader io.Reader = resp.Body
	if opts.MaxBytes > 0 {
		bodyReader = io.LimitReader(resp.Body, opts.MaxBytes+1)
	}

	// Special handling: stdout + hash requires buffering to verify before output
	if finalOutput == "-" && opts.ExpectedHash != "" {
		tempFile, err := os.CreateTemp("", "ripvex-*")
		if err != nil {
			return nil, fmt.Errorf("error creating temp file: %w", err)
		}
		tempPath := tempFile.Name()
		if tracker != nil {
			tracker.Register(tempPath)
		}
		defer func() {
			os.Remove(tempPath) // Always remove temp file
			if tracker != nil {
				tracker.Unregister(tempPath)
			}
		}()

		result, err := downloadWithProgress(ctx, tempFile, bodyReader, resp.ContentLength, finalOutput, opts.Quiet, opts.HashAlgorithm, opts.ExpectedHash, opts.MaxBytes)
		if err := tempFile.Close(); err != nil {
			return nil, fmt.Errorf("error closing temp file: %w", err)
		}
		if err != nil {
			result.OutputFile = finalOutput
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
		result.OutputFile = finalOutput
		return result, nil
	}

	// Standard flow: file output or stdout without hash (stream directly)
	var writer io.Writer
	if finalOutput == "-" {
		writer = os.Stdout
		result, err := downloadWithProgress(ctx, writer, bodyReader, resp.ContentLength, finalOutput, opts.Quiet, opts.HashAlgorithm, opts.ExpectedHash, opts.MaxBytes)
		if result != nil {
			result.OutputFile = finalOutput
		}
		return result, err
	}

	file, err := os.Create(finalOutput)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}
	// Register file for cleanup immediately after creation
	if tracker != nil {
		tracker.Register(finalOutput)
	}
	result, err := downloadWithProgress(ctx, file, bodyReader, resp.ContentLength, finalOutput, opts.Quiet, opts.HashAlgorithm, opts.ExpectedHash, opts.MaxBytes)
	if result != nil {
		result.OutputFile = finalOutput
	}
	if closeErr := file.Close(); closeErr != nil && err == nil {
		return result, fmt.Errorf("error closing output file: %w", closeErr)
	}
	return result, err
}

// extractFilenameFromContentDisposition extracts the filename from Content-Disposition header
// Returns empty string if header is missing or invalid
func extractFilenameFromContentDisposition(header string) string {
	if header == "" {
		return ""
	}

	// Parse the Content-Disposition header
	_, params, err := mime.ParseMediaType(header)
	if err != nil {
		return ""
	}

	// Try filename* first (RFC 5987, preferred)
	if filenameStar, ok := params["filename*"]; ok {
		// Handle RFC 5987 encoding: charset'lang'value or charset''value
		// Format: charset'lang'encoded-value where value is percent-encoded
		parts := strings.SplitN(filenameStar, "'", 3)
		if len(parts) >= 2 {
			// Get the last part (encoded value), handling both 2-part and 3-part formats
			encodedValue := parts[len(parts)-1]
			// Decode percent-encoded value (RFC 3986 encoding)
			decoded, err := url.QueryUnescape(encodedValue)
			if err == nil && decoded != "" {
				// Validate filename doesn't contain path traversal
				base := filepath.Base(decoded)
				if base != "" && base != "." && base != ".." && !strings.Contains(base, string(filepath.Separator)) {
					return base
				}
			}
		}
	}

	// Fall back to filename parameter
	if filename, ok := params["filename"]; ok && filename != "" {
		// Remove quotes if present
		filename = strings.Trim(filename, `"`)
		// Validate filename doesn't contain path traversal
		base := filepath.Base(filename)
		if base != "" && base != "." && base != ".." && !strings.Contains(base, string(filepath.Separator)) {
			return base
		}
	}

	return ""
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
func downloadWithProgress(ctx context.Context, writer io.Writer, reader io.Reader, total int64, outName string, quiet bool, hashAlgorithm string, expectedHash string, maxBytes int64) (*Result, error) {
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

	// Check cancellation periodically (every 10 iterations to avoid overhead)
	iterCount := 0
	for {
		// Check for cancellation every 10 iterations
		if iterCount%10 == 0 && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		iterCount++

		n, err := reader.Read(buf)

		// Process bytes FIRST (even if err == io.EOF)
		// Per io.Reader contract, Read() may return n > 0 AND io.EOF simultaneously
		if n > 0 {
			if hasher != nil {
				hasher.Write(buf[:n])
			}
			n2, writeErr := writer.Write(buf[:n])
			if writeErr != nil || n2 != n {
				return nil, fmt.Errorf("error writing: %w", writeErr)
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

		// THEN check for errors
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading: %w", err)
		}
	}

	// Content-Length validation (skip if hash verification is enabled, as it provides stronger integrity)
	if total > 0 && downloaded != total && expectedHash == "" {
		// Delete incomplete file if writing to a file (not stdout)
		if outName != "-" {
			if err := os.Remove(outName); err != nil && !os.IsNotExist(err) {
				if !quiet {
					fmt.Fprintf(os.Stderr, "\nWarning: failed to remove incomplete file %s: %v\n", outName, err)
				}
			}
		}
		return nil, fmt.Errorf("incomplete download: received %s, expected %s (Content-Length)", util.HumanReadableBytes(downloaded), util.HumanReadableBytes(total))
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
