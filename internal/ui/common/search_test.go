package common

import (
	"testing"

	"github.com/idursun/jjui/internal/screen"
	"github.com/stretchr/testify/assert"
)

type testLine struct {
	segments []*screen.Segment
}

func (l *testLine) GetSegments() []*screen.Segment {
	return l.segments
}

type testItem struct {
	lines []screen.SearchableLine
}

func (t *testItem) GetSearchableLines() []screen.SearchableLine {
	return t.lines
}

func newItem(texts ...string) screen.Searchable {
	segments := make([]*screen.Segment, len(texts))
	for i, t := range texts {
		segments[i] = &screen.Segment{Text: t}
	}
	return &testItem{
		lines: []screen.SearchableLine{&testLine{segments: segments}},
	}
}

func TestCircularSearch(t *testing.T) {
	items := []screen.Searchable{
		newItem("alpha"),
		newItem("bravo"),
		newItem("charlie"),
		newItem("delta"),
		newItem("alpha two"),
	}

	tests := []struct {
		name       string
		query      string
		startIndex int
		cursor     int
		backward   bool
		want       int
	}{
		{
			name:       "empty query returns cursor",
			query:      "",
			startIndex: 0,
			cursor:     2,
			want:       2,
		},
		{
			name:       "forward match from start",
			query:      "bravo",
			startIndex: 0,
			cursor:     0,
			want:       1,
		},
		{
			name:       "forward wraps around",
			query:      "alpha",
			startIndex: 3,
			cursor:     0,
			want:       4,
		},
		{
			name:       "backward match",
			query:      "bravo",
			startIndex: 3,
			cursor:     0,
			backward:   true,
			want:       1,
		},
		{
			name:       "backward wraps around",
			query:      "delta",
			startIndex: 1,
			cursor:     0,
			backward:   true,
			want:       3,
		},
		{
			name:       "no match returns cursor",
			query:      "zulu",
			startIndex: 0,
			cursor:     2,
			want:       2,
		},
		{
			name:       "case insensitive match",
			query:      "charlie",
			startIndex: 0,
			cursor:     0,
			want:       2,
		},
		{
			name:       "match spanning multiple segments",
			query:      "hello world",
			startIndex: 0,
			cursor:     0,
			want:       0,
		},
	}

	// override items for the spanning-segments test case
	spanItems := []screen.Searchable{
		newItem("hello ", "world"),
		newItem("other"),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchItems := items
			if tt.name == "match spanning multiple segments" {
				searchItems = spanItems
			}
			got := CircularSearch(searchItems, tt.query, tt.startIndex, tt.cursor, tt.backward)
			assert.Equal(t, tt.want, got)
		})
	}
}
