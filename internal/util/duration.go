package util

import (
	"time"

	str2duration "github.com/xhit/go-str2duration/v2"
)

// ParseDuration parses a human-readable duration string into time.Duration.
// Supports standard Go duration units (h, m, s, ms, us, ns) plus days (d) and weeks (w).
// Examples: "1h", "1h30m", "2d", "1w2d3h", "300s"
func ParseDuration(s string) (time.Duration, error) {
	return str2duration.ParseDuration(s)
}
