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

func TestCircularSearch_MatchesAcrossSegments(t *testing.T) {
	items := []screen.Searchable{newItem("alpha"), newItem("u", "nnyyxrq")}
	assert.Equal(t, 1, CircularSearch(items, "unny", 0, 0, false))
}

func TestMatchRanges(t *testing.T) {
	assert.Nil(t, MatchRanges("abc", ""))
	assert.Nil(t, MatchRanges("abc", "x"))
	assert.Equal(t, [][2]int{{0, 2}, {3, 5}}, MatchRanges("AbxaB", "ab"))
	assert.Equal(t, [][2]int{{0, 2}, {2, 4}}, MatchRanges("aaaa", "aa"))
}

func TestSplitByRanges(t *testing.T) {
	type piece struct {
		text    string
		matched bool
	}
	split := func(text string, offset int, ranges [][2]int) []piece {
		var pieces []piece
		SplitByRanges(text, offset, ranges, func(p string, matched bool) {
			pieces = append(pieces, piece{p, matched})
		})
		return pieces
	}

	// line "u" + "nnyyxrq", query "unny" -> range {0, 4}
	ranges := [][2]int{{0, 4}}
	assert.Equal(t, []piece{{"u", true}}, split("u", 0, ranges))
	assert.Equal(t, []piece{{"nny", true}, {"yxrq", false}}, split("nnyyxrq", 1, ranges))

	assert.Equal(t, []piece{{"abc", false}}, split("abc", 0, nil))
	assert.Equal(t, []piece{{"abc", false}}, split("abc", 10, ranges))
	assert.Equal(t, []piece{{"a", false}, {"b", true}, {"c", false}, {"d", true}},
		split("abcd", 0, [][2]int{{1, 2}, {3, 4}}))
	assert.Nil(t, split("", 0, ranges))
}
