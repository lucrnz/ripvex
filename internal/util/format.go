package util

import "fmt"

// HumanReadableBytes formats bytes to a human-readable string (B, KB, MB, GB)
func HumanReadableBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	var exp int = 0
	val := float64(bytes)
	for val >= unit && exp < 3 {
		val /= unit
		exp++
	}
	units := []string{"B", "KB", "MB", "GB"}
	return fmt.Sprintf("%.1f %s", val, units[exp])
}

