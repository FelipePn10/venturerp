// Package datetime provides tolerant parsing of date/time strings received
// from HTTP request payloads. Front-ends and integrations send dates in
// several shapes ("2006-01-02", RFC3339, "2006-01-02 15:04:05", ...); a
// strict single-layout time.Parse silently yields the zero value (0001-01-01)
// when the layout does not match, which is how emission/delivery dates were
// being persisted as 0001-01-01.
package datetime

import (
	"strings"
	"time"
)

// acceptedLayouts is tried in order. Date-only comes first because that is the
// documented contract; the remaining layouts tolerate datetime payloads.
var acceptedLayouts = []string{
	"2006-01-02",
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"02/01/2006",
}

// ParseDate parses s using the accepted layouts. It returns the parsed time
// truncated to date precision and true on success; on empty input or when no
// layout matches it returns the zero time and false.
func ParseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range acceptedLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// ParseDateOrDefault parses s and falls back to def when s is empty or invalid.
func ParseDateOrDefault(s string, def time.Time) time.Time {
	if t, ok := ParseDate(s); ok {
		return t
	}
	return def
}

// ParseDatePtr parses an optional date string. A nil or unparseable pointer
// yields nil so callers can leave nullable DATE columns untouched.
func ParseDatePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	if t, ok := ParseDate(*s); ok {
		return &t
	}
	return nil
}
