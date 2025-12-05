package util

import (
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
)

var sizeSynonyms = map[string]string{
	"":    "",
	"b":   "B",
	"k":   "KiB",
	"kb":  "KiB",
	"kib": "KiB",
	"m":   "MiB",
	"mb":  "MiB",
	"mib": "MiB",
	"g":   "GiB",
	"gb":  "GiB",
	"gib": "GiB",
}

// ParseByteSize parses a human-readable byte size (supports k/K/kb/KiB/MB/GiB) into bytes.
// Returns an error for unknown units or invalid numeric parts.
func ParseByteSize(s string) (int64, error) {
	str := strings.TrimSpace(s)
	if str == "" {
		return 0, fmt.Errorf("size cannot be empty")
	}
	lower := strings.ToLower(str)

	// Split numeric prefix and unit suffix
	numEnd := 0
	for numEnd < len(lower) {
		c := lower[numEnd]
		if (c >= '0' && c <= '9') || c == '.' {
			numEnd++
			continue
		}
		break
	}

	if numEnd == 0 {
		return 0, fmt.Errorf("invalid size: %q", s)
	}

	numPart := strings.TrimSpace(lower[:numEnd])
	unitPart := strings.TrimSpace(lower[numEnd:])

	canonUnit, ok := sizeSynonyms[unitPart]
	if !ok {
		return 0, fmt.Errorf("invalid size unit %q", unitPart)
	}

	combined := numPart
	if canonUnit != "" {
		combined = combined + canonUnit
	}

	val, err := humanize.ParseBytes(combined)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", s, err)
	}

	if val > uint64(^uint64(0)>>1) {
		return 0, fmt.Errorf("size out of range")
	}

	return int64(val), nil
}
