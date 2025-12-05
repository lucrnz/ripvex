package cli

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"simple-downloader/internal/archive"
	"simple-downloader/internal/downloader"
	"simple-downloader/internal/util"
	"simple-downloader/internal/version"
)

var (
	urlStr             string
	output             string
	quiet              bool
	expectedHash       string
	extractArchive     bool
	removeArchive      bool
	chdir              string
	stripComponents    int
	connectTimeout     time.Duration
	maxTime            time.Duration
	userAgent          string
	maxBytesStr        string
	extractMaxBytesStr string
)

var rootCmd = &cobra.Command{
	Use:   "simple-downloader",
	Short: "Your Swiss-Army Knife for downloading files",
	Long: fmt.Sprintf(`Simple Downloader

Downloads a file from URL, following redirects.
Optionally extracts archives (zip, tar, tar.gz, tar.bz2, tar.xz, tar.zstd).

Copyright (c) %d Luciano Hillcoat.
This program is open-source and warranty-free, read more at: https://github.com/lucrnz/software-distillery/blob/main/LICENSE
`, time.Now().Year()),
	RunE:    run,
	Version: version.Print(),
}

func init() {
	rootCmd.Flags().StringVarP(&urlStr, "url", "U", "", "The URL to download (required)")
	rootCmd.Flags().StringVarP(&output, "output", "O", "", "The name for the file to write it as")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Does not show any progress or output")
	rootCmd.Flags().StringVarP(&expectedHash, "hash", "H", "", "Expected hash with algorithm prefix (e.g., sha256:xxxxx... or sha512:xxxxx...). Supported algorithms: sha256, sha512")
	rootCmd.Flags().BoolVarP(&extractArchive, "extract-archive", "x", false, "Extract the downloaded archive")
	rootCmd.Flags().BoolVar(&removeArchive, "remove-archive", true, "Delete archive file after successful extraction")
	rootCmd.Flags().StringVarP(&chdir, "chdir", "C", "", "Change working directory before any operation (panics if directory doesn't exist)")
	rootCmd.Flags().IntVar(&stripComponents, "extract-strip-components", 0, "Strip N leading components from file names during extraction")
	rootCmd.Flags().DurationVar(&connectTimeout, "connect-timeout", 300*time.Second, "Maximum time for connection establishment")
	rootCmd.Flags().DurationVarP(&maxTime, "max-time", "m", 0, "Maximum total time for the entire operation (0 = unlimited)")
	rootCmd.Flags().StringVar(&userAgent, "user-agent", version.UserAgent(), "User-Agent header to send with HTTP requests")
	rootCmd.Flags().StringVarP(&maxBytesStr, "max-bytes", "M", "4GiB", "Maximum bytes to download (e.g., \"4GiB\", \"512MB\")")
	rootCmd.Flags().StringVar(&extractMaxBytesStr, "extract-max-bytes", "8GiB", "Maximum total bytes to extract from archive (e.g., \"8GiB\")")

	rootCmd.MarkFlagRequired("url")
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Change directory first if specified
	if chdir != "" {
		if err := os.Chdir(chdir); err != nil {
			return fmt.Errorf("failed to change directory to %q: %w", chdir, err)
		}
	}

	// Normalize URL
	if !strings.HasPrefix(urlStr, "http") {
		urlStr = "https://" + urlStr
	}

	// Determine output filename
	if output == "" {
		parsedURL := urlStr
		if idx := strings.LastIndex(parsedURL, "/"); idx != -1 {
			output = parsedURL[idx+1:]
		}
		if output == "" || output == "/" {
			output = "download"
		}
		// Strip query string if present
		if idx := strings.Index(output, "?"); idx != -1 {
			output = output[:idx]
		}
	}

	// Cannot extract when outputting to stdout
	if extractArchive && output == "-" {
		return fmt.Errorf("cannot extract archive when output is stdout (-)")
	}

	// Parse size limits
	maxBytes, err := util.ParseByteSize(maxBytesStr)
	if err != nil {
		return fmt.Errorf("invalid --max-bytes value: %w", err)
	}

	extractMaxBytes, err := util.ParseByteSize(extractMaxBytesStr)
	if err != nil {
		return fmt.Errorf("invalid --extract-max-bytes value: %w", err)
	}

	hashAlgo, hashDigest, err := parseExpectedHash(expectedHash)
	if err != nil {
		return err
	}

	// Perform download
	opts := downloader.Options{
		URL:            urlStr,
		Output:         output,
		Quiet:          quiet,
		HashAlgorithm:  hashAlgo,
		ExpectedHash:   hashDigest,
		ConnectTimeout: connectTimeout,
		MaxTime:        maxTime,
		UserAgent:      userAgent,
		MaxBytes:       maxBytes,
	}

	_, err = downloader.Download(opts)
	if err != nil {
		return err
	}

	// Extract archive if requested
	if extractArchive {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Detecting archive type...\n")
		}

		archiveType, err := archive.Detect(output)
		if err != nil {
			return fmt.Errorf("error detecting archive type: %w", err)
		}

		if archiveType == archive.Unknown {
			return fmt.Errorf("unknown or unsupported archive format")
		}

		if !quiet {
			fmt.Fprintf(os.Stderr, "Detected archive type: %s\n", archiveType)
			fmt.Fprintf(os.Stderr, "Extracting...\n")
		}

		opts := archive.ExtractOptions{
			StripComponents: stripComponents,
			MaxBytes:        extractMaxBytes,
		}
		if err := archive.Extract(output, archiveType, opts); err != nil {
			return fmt.Errorf("error extracting archive: %w", err)
		}

		if !quiet {
			fmt.Fprintf(os.Stderr, "✅ Extraction complete\n")
		}

		if removeArchive {
			if err := os.Remove(output); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove archive file: %v\n", err)
			} else if !quiet {
				fmt.Fprintf(os.Stderr, "Removed archive file: %s\n", output)
			}
		}
	}

	return nil
}

// hashConfig holds configuration for a hash algorithm
type hashConfig struct {
	name      string
	digestLen int
	newHash   func() hash.Hash
}

// supportedHashes is a registry of supported hash algorithms
// This design makes it easy to add blake3, sha3, etc. in the future
var supportedHashes = map[string]hashConfig{
	"sha256": {
		name:      "SHA-256",
		digestLen: 64, // 256 bits = 64 hex chars
		newHash:   sha256.New,
	},
	"sha512": {
		name:      "SHA-512",
		digestLen: 128, // 512 bits = 128 hex chars
		newHash:   sha512.New,
	},
}

// parseExpectedHash parses a hash string that may include an algorithm prefix.
// Returns (algorithm, digest, error).
// If no prefix is found, emits a deprecation warning and defaults to SHA-256.
func parseExpectedHash(hashStr string) (string, string, error) {
	if hashStr == "" {
		return "", "", nil
	}

	// Check if hash has a prefix (e.g., "sha256:xxxxx")
	parts := strings.SplitN(hashStr, ":", 2)
	if len(parts) == 2 {
		// Has prefix
		algo := strings.ToLower(parts[0])
		digest := strings.ToLower(parts[1])

		// Validate algorithm is supported
		config, ok := supportedHashes[algo]
		if !ok {
			supported := make([]string, 0, len(supportedHashes))
			for k := range supportedHashes {
				supported = append(supported, k)
			}
			return "", "", fmt.Errorf("unsupported hash algorithm %q. Supported algorithms: %s", algo, strings.Join(supported, ", "))
		}

		// Validate digest length
		if len(digest) != config.digestLen {
			return "", "", fmt.Errorf("invalid %s hash: expected %d hex characters, got %d", config.name, config.digestLen, len(digest))
		}

		// Validate hex characters
		for _, c := range digest {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return "", "", fmt.Errorf("invalid %s hash: contains non-hex character '%c'", config.name, c)
			}
		}

		return algo, digest, nil
	} else {
		return "", "", fmt.Errorf("hash must be prefixed with the algorithm name followed by a colon. example: sha256:{value}")
	}
}
