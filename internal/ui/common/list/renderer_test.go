package list

import (
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/ui/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ IItemRenderer = (*testItemRenderer)(nil)

type testItemRenderer struct {
	index  int
	height int
}

func (t testItemRenderer) Render(w CursorWriter, width int) {
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
			renderer := NewRenderer(&tc.list, common.NewViewNode(20, tc.height))
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
	renderer := NewRenderer(&l, common.NewViewNode(20, 3))

	_ = renderer.RenderWithOptions(RenderOptions{FocusIndex: 1, EnsureFocusVisible: true})

	totalLines := renderer.TotalLineCount()
	absoluteLines := renderer.AbsoluteLineCount()
	assert.Equal(t, 3, totalLines)
	assert.Equal(t, 6, absoluteLines)

	maxStart := absoluteLines - renderer.Height
	assert.Greater(t, maxStart, 0)
}

type itemPos struct {
	localLine  int
	localCol   int
	screenLine int
	screenCol  int
}

type cursorRecorder struct {
	lines []string
	pos   []itemPos
}

func (c *cursorRecorder) Render(w CursorWriter, width int) {
	for _, line := range c.lines {
		_, _ = io.WriteString(w, line)
		l, col := w.LocalPos()
		sl, scol := w.ViewportPos()
		c.pos = append(c.pos, struct {
			localLine  int
			localCol   int
			screenLine int
			screenCol  int
		}{l, col, sl, scol})
		_, _ = io.WriteString(w, "\n")
	}
}

func (c *cursorRecorder) Height() int {
	return len(c.lines)
}

type cursorList struct {
	item *cursorRecorder
}

func (c cursorList) Len() int {
	return 1
}

func (c cursorList) GetItemRenderer(index int) IItemRenderer {
	return c.item
}

func TestListRenderer_ProvidesCursorPositions(t *testing.T) {
	item := &cursorRecorder{
		lines: []string{"first", "second", "third"},
	}
	list := cursorList{item: item}
	renderer := NewRenderer(list, common.NewViewNode(10, 2))
	renderer.Start = 1 // only render second and third lines

	content := renderer.RenderWithOptions(RenderOptions{FocusIndex: 0, EnsureFocusVisible: false})
	assert.NotEmpty(t, content)

	require.Len(t, item.pos, 3)
	assert.Equal(t, itemPos{localLine: 0, localCol: len("first"), screenLine: -1, screenCol: -1}, item.pos[0])
	assert.Equal(t, itemPos{localLine: 1, localCol: len("second"), screenLine: 0, screenCol: len("second")}, item.pos[1])
	assert.Equal(t, itemPos{localLine: 2, localCol: len("third"), screenLine: 1, screenCol: len("third")}, item.pos[2])
}
