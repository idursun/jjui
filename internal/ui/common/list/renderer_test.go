package list

import (
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ IItemRenderer = (*testItemRenderer)(nil)

type testItemRenderer struct {
	index  int
	height int
}

func (t testItemRenderer) Render(w io.Writer, width int) {
	line := strings.Repeat(strconv.Itoa(t.index), width)
	for i := 0; i < t.height; i++ {
		io.WriteString(w, line+"\n")
	}
}

func (t testItemRenderer) Height() int {
	return t.height
}

var _ IList = (*testList)(nil)

type testList struct {
	itemHeights []int
}

func (t testList) Len() int {
	return len(t.itemHeights)
}

func (t testList) GetItemRenderer(index int) IItemRenderer {
	return &testItemRenderer{height: t.itemHeights[index], index: index}
}

func TestListRenderer_RowRanges(t *testing.T) {
	tests := []struct {
		name           string
		height         int
		list           testList
		viewRangeStart int
		opts           RenderOptions
		expected       []RowRange
	}{
		{
			name:   "renders all until they fit",
			height: 3,
			list:   testList{itemHeights: []int{2, 3, 1}},
			opts:   RenderOptions{FocusIndex: 0},
			expected: []RowRange{
				{Row: 0, StartLine: 0, EndLine: 2},
				{Row: 1, StartLine: 2, EndLine: 3},
			},
		},
		{
			name:   "ensures focused item is visible",
			height: 3,
			list:   testList{itemHeights: []int{2, 3, 1}},
			opts:   RenderOptions{FocusIndex: 1, EnsureFocusVisible: true},
			expected: []RowRange{
				{Row: 1, StartLine: 2, EndLine: 5},
			},
		},
		{
			name:           "no ensure focus visible",
			height:         3,
			list:           testList{itemHeights: []int{2, 3, 1}},
			opts:           RenderOptions{FocusIndex: 2, EnsureFocusVisible: false},
			viewRangeStart: 2,
			expected: []RowRange{
				{Row: 1, StartLine: 2, EndLine: 5},
			},
		},
		{
			name:           "ensures focused respect view range",
			height:         3,
			list:           testList{itemHeights: []int{2, 3, 1}},
			viewRangeStart: 2,
			opts:           RenderOptions{FocusIndex: 1, EnsureFocusVisible: true},
			expected: []RowRange{
				{Row: 1, StartLine: 2, EndLine: 5},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			renderer := NewRenderer(&tc.list, 20, tc.height)
			renderer.Start = tc.viewRangeStart

			v := renderer.RenderWithOptions(tc.opts)
			assert.NotEmpty(t, v)

			ranges := renderer.RowRanges()
			require.Equal(t, tc.expected, ranges)
			assert.Equal(t, tc.height, renderer.TotalLineCount())
		})
	}
}

func TestListRenderer_AbsoluteLineCount_AllowsScrollAfterFocusedRender(t *testing.T) {
	l := testList{
		itemHeights: []int{2, 3, 1},
	}
	renderer := NewRenderer(&l, 20, 3)

	_ = renderer.RenderWithOptions(RenderOptions{FocusIndex: 1, EnsureFocusVisible: true})

	totalLines := renderer.TotalLineCount()
	absoluteLines := renderer.AbsoluteLineCount()
	assert.Equal(t, 3, totalLines)
	assert.Equal(t, 6, absoluteLines)

	maxStart := absoluteLines - renderer.Height
	assert.Greater(t, maxStart, 0)
}
