package common

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// truncatePathFromLeft truncates a file path from the left side when it's too long,
// preserving the most meaningful parts (end segments) and adding "..." prefix.
// maxWidth should account for the fact that this will be rendered in a monospace font.
func TruncatePathFromLeft(path string, maxWidth int) string {
	if len(path) <= maxWidth {
		return path
	}

	// Split the path into segments
	segments := strings.Split(filepath.Clean(path), string(filepath.Separator))

	// Always try to keep at least the last 2 segments if possible
	minSegments := 2
	if len(segments) < minSegments {
		minSegments = len(segments)
	}

	// Start with the most important segments (from the end)
	var result strings.Builder
	ellipsis := "..."

	// Account for the ellipsis in our width calculation
	remainingWidth := maxWidth - len(ellipsis)

	// Build path from the end, keeping as many segments as fit
	var keptSegments []string
	currentLength := 0

	// Work backwards through segments
	for i := len(segments) - 1; i >= 0; i-- {
		segment := segments[i]
		segmentWithSep := segment

		// Add separator length except for the last segment
		if len(keptSegments) > 0 {
			segmentWithSep = string(filepath.Separator) + segment
		}

		// Check if adding this segment would exceed our width
		if currentLength+len(segmentWithSep) > remainingWidth {
			// If we haven't kept the minimum segments yet, force include them
			if len(keptSegments) < minSegments && i < len(segments)-minSegments {
				break
			}
			// Otherwise stop here
			break
		}

		keptSegments = append([]string{segment}, keptSegments...)
		currentLength += len(segmentWithSep)
	}

	// If we kept all segments, just return the original path
	if len(keptSegments) == len(segments) {
		return path
	}

	// Build the truncated path with ellipsis
	result.WriteString(ellipsis)
	for i, segment := range keptSegments {
		if i > 0 || len(keptSegments) == 1 {
			result.WriteString(string(filepath.Separator))
		}
		result.WriteString(segment)
	}

	return result.String()
}

// FormatTimeAgo formats a time into a shorthand "time ago" string
// Examples: "now", "5m", "2h", "3d", "2w", "1mo", "1y"
func FormatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	seconds := int(duration.Seconds())
	minutes := int(duration.Minutes())
	hours := int(duration.Hours())
	days := hours / 24
	weeks := days / 7
	months := days / 30
	years := days / 365

	switch {
	case seconds < 60:
		return "now"
	case minutes < 60:
		return formatUnit(minutes, "m")
	case hours < 24:
		return formatUnit(hours, "h")
	case days < 7:
		return formatUnit(days, "d")
	case days < 30:
		return formatUnit(weeks, "w")
	case days < 365:
		return formatUnit(months, "mo")
	default:
		return formatUnit(years, "y")
	}
}

func formatUnit(value int, unit string) string {
	if value == 0 {
		return "now"
	}
	return fmt.Sprintf("%d%s", value, unit)
}
