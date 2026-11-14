package common

import (
	"strings"

	"github.com/idursun/jjui/internal/screen"
)

// CircularSearch performs a circular search through searchable items.
// Returns the index of the first matching item, or cursor if no match is found.
func CircularSearch(items []screen.Searchable, query string, startIndex, cursor int, backward bool) int {
	if query == "" {
		return cursor
	}

	n := len(items)
	for i := range n {
		var c int
		if !backward {
			c = (startIndex + i) % n
		} else {
			c = (startIndex - i + n) % n
		}
		if MatchesQuery(items[c], query) {
			return c
		}
	}
	return cursor
}

func MatchesQuery(item screen.Searchable, query string) bool {
	for _, line := range item.GetSearchableLines() {
		var text strings.Builder
		for _, segment := range line.GetSegments() {
			text.WriteString(segment.Text)
		}
		if strings.Contains(strings.ToLower(text.String()), query) {
			return true
		}
	}
	return false
}

// MatchRanges returns the byte ranges of non-overlapping, case-insensitive matches of query in text.
func MatchRanges(text, query string) [][2]int {
	if query == "" {
		return nil
	}
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)
	var ranges [][2]int
	for start := 0; ; {
		idx := strings.Index(lowerText[start:], lowerQuery)
		if idx == -1 {
			return ranges
		}
		idx += start
		start = idx + len(lowerQuery)
		ranges = append(ranges, [2]int{idx, start})
	}
}

// SplitByRanges splits a segment that starts at offset within its line into
// consecutive pieces, reporting whether each piece falls inside one of the line's ranges.
func SplitByRanges(text string, offset int, ranges [][2]int, emit func(piece string, matched bool)) {
	pos := 0
	for _, r := range ranges {
		start := max(r[0]-offset, pos)
		end := min(r[1]-offset, len(text))
		if start >= end {
			continue
		}
		if start > pos {
			emit(text[pos:start], false)
		}
		emit(text[start:end], true)
		pos = end
	}
	if pos < len(text) {
		emit(text[pos:], false)
	}
}
