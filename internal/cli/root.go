package cli

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/lucrnz/ripvex/internal/archive"
	"github.com/lucrnz/ripvex/internal/downloader"
	"github.com/lucrnz/ripvex/internal/util"
	"github.com/lucrnz/ripvex/internal/version"
)

var (
	urlStr             string
	output             string
	quiet              bool
	expectedHash       string
	extractArchive     bool
	removeArchive      bool
	chdir              string
	chdirCreate        bool
	stripComponents    int
	connectTimeout     time.Duration
	maxTime            time.Duration
	maxRedirects       int
	userAgent          string
	maxBytesStr        string
	extractMaxBytesStr string
	allowInsecureTLS   bool
	headers            []string
	auth               string
	authBearer         string
	authBasicUser      string
	authBasicPass      string
	authBasic          string
)

var rootCmd = &cobra.Command{
	Use:   "ripvex",
	Short: "Your Swiss-Army Knife for downloading files",
	Long: `ripvex

Lightweight Go program for downloading files from URLs with optional hash integrity verification and archive extraction.

Copyright (c) 2025 Luciano Hillcoat.
This program is open-source and warranty-free, read more at: https://github.com/lucrnz/ripvex/blob/main/LICENSE
`,
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
	rootCmd.Flags().BoolVar(&chdirCreate, "chdir-create", false, "Create directory if it doesn't exist (requires --chdir)")
	rootCmd.Flags().IntVar(&stripComponents, "extract-strip-components", 0, "Strip N leading components from file names during extraction")
	rootCmd.Flags().DurationVar(&connectTimeout, "connect-timeout", 300*time.Second, "Maximum time for connection establishment")
	rootCmd.Flags().DurationVarP(&maxTime, "max-time", "m", 0, "Maximum total time for the entire operation (0 = unlimited)")
	rootCmd.Flags().IntVar(&maxRedirects, "max-redirs", 30, "Maximum number of redirects to follow")
	rootCmd.Flags().StringVar(&userAgent, "user-agent", version.UserAgent(), "User-Agent header to send with HTTP requests")
	rootCmd.Flags().StringVarP(&maxBytesStr, "max-bytes", "M", "4GiB", "Maximum bytes to download (e.g., \"4GiB\", \"512MB\")")
	rootCmd.Flags().StringVar(&extractMaxBytesStr, "extract-max-bytes", "8GiB", "Maximum total bytes to extract from archive (e.g., \"8GiB\")")
	rootCmd.Flags().BoolVar(&allowInsecureTLS, "allow-insecure-tls", false, "Allow insecure TLS versions (1.0/1.1) with known vulnerabilities")
	rootCmd.Flags().StringArrayVar(&headers, "header", []string{}, "Custom header in \"Key: Value\" format. Can be specified multiple times.")
	rootCmd.Flags().StringVarP(&auth, "auth", "A", "", "Set Authorization header to the provided value")
	rootCmd.Flags().StringVarP(&authBearer, "auth-bearer", "B", "", "Set Authorization header to \"Bearer {value}\"")
	rootCmd.Flags().StringVar(&authBasicUser, "auth-basic-user", "", "Username for HTTP Basic authentication (requires --auth-basic-pass)")
	rootCmd.Flags().StringVar(&authBasicPass, "auth-basic-pass", "", "Password for HTTP Basic authentication (requires --auth-basic-user)")
	rootCmd.Flags().StringVar(&authBasic, "auth-basic", "", "Custom base64 value for Basic auth (cannot be used with --auth-basic-user/pass)")

	rootCmd.MarkFlagRequired("url")

	// Silence usage output for runtime errors, but show it for flag errors
	// SilenceErrors is true so we can control error output format in main()
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	// Show usage only when there's a flag parsing error
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		_ = cmd.Usage()
		return err
	})
}

// Execute runs the root command
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		// Show usage for required flag errors (not caught by SetFlagErrorFunc)
		if strings.Contains(err.Error(), "required flag") {
			_ = rootCmd.Usage()
		}
		return err
	}
	return nil
}

func run(cmd *cobra.Command, args []string) error {
	// Change directory first if specified
	if chdir != "" {
		if chdirCreate {
			if err := os.MkdirAll(chdir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %q: %w", chdir, err)
			}
		}
		if err := os.Chdir(chdir); err != nil {
			return fmt.Errorf("failed to change directory to %q: %w", chdir, err)
		}
	} else if chdirCreate {
		return fmt.Errorf("--chdir-create requires --chdir to be specified")
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q: only http and https are supported", parsedURL.Scheme)
	}
	urlStr = parsedURL.String()

	// Track whether --output was explicitly set
	outputExplicit := output != ""

	// Determine output filename (fallback if not explicitly set)
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

	// Validate max-redirs
	if maxRedirects < 0 {
		return fmt.Errorf("--max-redirs must be non-negative, got %d", maxRedirects)
	}

	// Validate strip-components
	if stripComponents < 0 {
		return fmt.Errorf("--extract-strip-components must be non-negative, got %d", stripComponents)
	}

	// Parse and validate authorization flags
	headersMap := make(map[string]string)

	// Parse --header flags (curl-style: "Key: Value")
	for _, headerStr := range headers {
		parts := strings.SplitN(headerStr, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid header format: expected \"Key: Value\", got %q", headerStr)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return fmt.Errorf("header key cannot be empty")
		}
		headersMap[key] = value
	}

	// Count auth methods to enforce mutual exclusion
	authMethods := 0
	if auth != "" {
		authMethods++
	}
	if authBearer != "" {
		authMethods++
	}
	if authBasicUser != "" || authBasicPass != "" {
		authMethods++
	}
	if authBasic != "" {
		authMethods++
	}

	if authMethods > 1 {
		return fmt.Errorf("only one authentication method can be specified at a time")
	}

	// Validate and set auth headers
	if auth != "" {
		headersMap["Authorization"] = auth
	} else if authBearer != "" {
		headersMap["Authorization"] = "Bearer " + authBearer
	} else if authBasicUser != "" || authBasicPass != "" {
		// Both user and pass must be set together
		if authBasicUser == "" || authBasicPass == "" {
			return fmt.Errorf("--auth-basic-user and --auth-basic-pass must both be specified together")
		}
		// Base64 encode username:password
		credentials := authBasicUser + ":" + authBasicPass
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		headersMap["Authorization"] = "Basic " + encoded
	} else if authBasic != "" {
		headersMap["Authorization"] = "Basic " + authBasic
	}

	// Perform download
	opts := downloader.Options{
		URL:              urlStr,
		Output:           output,
		OutputExplicit:   outputExplicit,
		Quiet:            quiet,
		HashAlgorithm:    hashAlgo,
		ExpectedHash:     hashDigest,
		ConnectTimeout:   connectTimeout,
		MaxTime:          maxTime,
		MaxRedirects:     maxRedirects,
		UserAgent:        userAgent,
		MaxBytes:         maxBytes,
		AllowInsecureTLS: allowInsecureTLS,
		Headers:          headersMap,
	}

	result, err := downloader.Download(opts)
	if err != nil {
		return err
	}

	// Use the final output filename from the download result (may have been updated by Content-Disposition)
	finalOutputFile := result.OutputFile
	if finalOutputFile == "" {
		// Fallback to original output if result doesn't have OutputFile set (shouldn't happen, but safety)
		finalOutputFile = output
	}

	// Extract archive if requested
	if extractArchive {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Detecting archive type...\n")
		}

		archiveType, err := archive.Detect(finalOutputFile)
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
		if err := archive.Extract(finalOutputFile, archiveType, opts); err != nil {
			return fmt.Errorf("error extracting archive: %w", err)
		}

		if !quiet {
			fmt.Fprintf(os.Stderr, "✅ Extraction complete\n")
		}

		if removeArchive {
			if err := os.Remove(finalOutputFile); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove archive file: %v\n", err)
			} else if !quiet {
				fmt.Fprintf(os.Stderr, "Removed archive file: %s\n", finalOutputFile)
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
