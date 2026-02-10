package timeparse

import (
	"strconv"
	"strings"
)

// Parse converts a human-readable time string like "2h 30m", "1.5ч", "30м" into seconds.
func Parse(input string) int {
	s := strings.ToLower(strings.TrimSpace(input))
	s = strings.ReplaceAll(s, "ч", "h")
	s = strings.ReplaceAll(s, "м", "m")

	totalSeconds := 0

	if strings.Contains(s, "h") {
		parts := strings.SplitN(s, "h", 2)
		hours, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err == nil {
			totalSeconds += int(hours * 3600)
		}
		if len(parts) > 1 {
			s = parts[1]
		} else {
			s = ""
		}
	}

	if strings.Contains(s, "m") {
		parts := strings.SplitN(s, "m", 2)
		minutes, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err == nil {
			totalSeconds += int(minutes * 60)
		}
	}

	return totalSeconds
}
