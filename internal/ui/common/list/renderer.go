package list

import (
	"bytes"
	"strings"

	"github.com/idursun/jjui/internal/ui/common"
)

type ListRenderer struct {
	*common.ViewRange
	list             IList
	buffer           bytes.Buffer
	skippedLineCount int
	lineCount        int
	rowRanges        []RowRange
}

func NewRenderer(list IList, size *common.Sizeable) *ListRenderer {
	return &ListRenderer{
		ViewRange: &common.ViewRange{Sizeable: size, Start: 0, End: size.Height, FirstRowIndex: -1, LastRowIndex: -1},
		list:      list,
		buffer:    bytes.Buffer{},
	}
}

func (r *ListRenderer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	r.lineCount += bytes.Count(p, []byte("\n"))
	return r.buffer.Write(p)
}

func (r *ListRenderer) String(start, end int) string {
	start = start - r.skippedLineCount
	end = end - r.skippedLineCount
	lines := strings.Split(r.buffer.String(), "\n")
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	for end > len(lines) {
		lines = append(lines, "")
	}
	return strings.Join(lines[start:end], "\n")
}

func (r *ListRenderer) Reset() {
	r.buffer.Reset()
	r.lineCount = 0
	r.skippedLineCount = 0
}

type RenderOptions struct {
	FocusIndex         int
	EnsureFocusVisible bool
}

func (r *ListRenderer) Render(focusIndex int) string {
	return r.RenderWithOptions(RenderOptions{FocusIndex: focusIndex, EnsureFocusVisible: true})
}

func (r *ListRenderer) RenderWithOptions(opts RenderOptions) string {
	r.Reset()
	r.rowRanges = r.rowRanges[:0]
	if opts.FocusIndex < 0 {
		opts.FocusIndex = 0
	}
	viewHeight := r.End - r.Start
	if viewHeight != r.Height {
		r.End = r.Start + r.Height
	}
	if r.Start < 0 {
		r.Start = 0
	}

	selectedLineStart := -1
	selectedLineEnd := -1
	firstRenderedRowIndex := -1
	lastRenderedRowIndex := -1
	for i := range r.list.Len() {
		isFocused := i == opts.FocusIndex
		itemRenderer := r.list.GetItemRenderer(i)
		if isFocused {
			selectedLineStart = r.totalLineCount()
			if opts.EnsureFocusVisible && selectedLineStart < r.Start {
				r.Start = selectedLineStart
			}
		} else {
			rowLineCount := itemRenderer.Height()
			if rowLineCount+r.totalLineCount() < r.Start {
				r.skipLines(rowLineCount)
				continue
			}
		}
		rowStart := r.totalLineCount()
		itemRenderer.Render(r, r.ViewRange.Width)
		rowEnd := r.totalLineCount()
		if rowEnd > rowStart {
			r.rowRanges = append(r.rowRanges, RowRange{Row: i, StartLine: rowStart, EndLine: rowEnd})
		}
		if firstRenderedRowIndex == -1 {
			firstRenderedRowIndex = i
		}

		if isFocused {
			selectedLineEnd = r.totalLineCount()
		}
		if r.totalLineCount() > r.End {
			lastRenderedRowIndex = i
			break
		}
	}

	if lastRenderedRowIndex == -1 {
		lastRenderedRowIndex = r.list.Len() - 1
	}

	r.FirstRowIndex = firstRenderedRowIndex
	r.LastRowIndex = lastRenderedRowIndex
	if opts.EnsureFocusVisible && selectedLineStart >= 0 {
		if selectedLineStart <= r.Start {
			r.Start = selectedLineStart
			r.End = selectedLineStart + r.Height
		} else if selectedLineEnd > r.End {
			r.End = selectedLineEnd
			r.Start = selectedLineEnd - r.Height
		}
	}

	if maxStart := r.totalLineCount() - r.Height; r.Start > maxStart && maxStart >= 0 {
		r.Start = maxStart
		r.End = r.Start + r.Height
	}
	return r.String(r.Start, r.End)
}

func (r *ListRenderer) skipLines(amount int) {
	r.skippedLineCount = r.skippedLineCount + amount
}

func (r *ListRenderer) totalLineCount() int {
	return r.lineCount + r.skippedLineCount
}

func (r *ListRenderer) TotalLineCount() int {
	// Walk all rows to avoid depending on a prior render's buffer state.
	total := 0
	for i := range r.list.Len() {
		total += r.list.GetItemRenderer(i).Height()
	}
	return total
}

type RowRange struct {
	Row       int
	StartLine int
	EndLine   int
}

func (r *ListRenderer) RowRanges() []RowRange {
	return r.rowRanges
}
