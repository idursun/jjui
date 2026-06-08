package common

import (
	"strings"

	"github.com/idursun/jjui/internal/screen"
)

// CircularSearch performs a circular search through searchable items.
// Returns (matchedIndex, true) when an item matches, or (cursor, false) otherwise.
func CircularSearch(items []screen.Searchable, query string, startIndex, cursor int, backward bool) (int, bool) {
	if query == "" {
		return cursor, false
	}

	n := len(items)
	for i := range n {
		var c int
		if !backward {
			c = (startIndex + i) % n
		} else {
			c = (startIndex - i + n) % n
		}
		if matchesQuery(items[c], query) {
			return c, true
		}
	}
	return cursor, false
}

func matchesQuery(item screen.Searchable, query string) bool {
	for _, line := range item.GetSearchableLines() {
		for _, segment := range line.GetSegments() {
			if segment.Text != "" && strings.Contains(strings.ToLower(segment.Text), query) {
				return true
			}
		}
	}
	return false
}
