package graph

import (
	"bytes"
	"strings"

	"github.com/idursun/jjui/internal/ui/common"
)

type ViewRange struct {
	*common.Sizeable
	start         int
	end           int
	firstRowIndex int
	lastRowIndex  int
}

func (v *ViewRange) FirstRowIndex() int {
	return v.firstRowIndex
}

func (v *ViewRange) LastRowIndex() int {
	return v.lastRowIndex
}

func (v *ViewRange) Reset() {
	v.start = 0
	v.end = 0
	v.firstRowIndex = -1
	v.lastRowIndex = -1
}

type Renderer struct {
	*ViewRange
	buffer           bytes.Buffer
	skippedLineCount int
	lineCount        int
}

func NewRenderer(width int, height int) *Renderer {
	return &Renderer{
		ViewRange: &ViewRange{Sizeable: common.NewSizeable(width, height), start: 0, end: height, firstRowIndex: -1, lastRowIndex: -1},
		buffer:    bytes.Buffer{},
	}
}

func (r *Renderer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	r.lineCount += bytes.Count(p, []byte("\n"))
	return r.buffer.Write(p)
}

func (r *Renderer) String(start, end int) string {
	start = start - r.skippedLineCount
	end = end - r.skippedLineCount
	lines := strings.Split(r.buffer.String(), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
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

func (r *Renderer) Reset() {
	r.buffer.Reset()
	r.ViewRange.Reset()
	r.lineCount = 0
	r.skippedLineCount = 0
}

func (r *Renderer) Render(iterator RowIterator) string {
	r.Reset()
	viewHeight := r.ViewRange.end - r.ViewRange.start
	if viewHeight != r.Height {
		r.ViewRange.end = r.ViewRange.start + r.Height
	}

	selectedLineStart := -1
	selectedLineEnd := -1
	firstRenderedRowIndex := -1
	lastRenderedRowIndex := -1
	i := -1
	for {
		i++
		ok := iterator.Next()
		if !ok {
			break
		}
		if iterator.IsHighlighted() {
			selectedLineStart = r.totalLineCount()
		} else {
			rowLineCount := iterator.RowHeight()
			if rowLineCount+r.totalLineCount() < r.ViewRange.start {
				r.skipLines(rowLineCount)
				continue
			}
		}
		iterator.Render(r)
		if firstRenderedRowIndex == -1 {
			firstRenderedRowIndex = i
		}

		if iterator.IsHighlighted() {
			selectedLineEnd = r.totalLineCount()
		}
		if selectedLineEnd > 0 && r.totalLineCount() > r.ViewRange.end {
			lastRenderedRowIndex = i
			break
		}
	}
	if lastRenderedRowIndex == -1 {
		lastRenderedRowIndex = iterator.Len() - 1
	}

	r.ViewRange.firstRowIndex = firstRenderedRowIndex
	r.ViewRange.lastRowIndex = lastRenderedRowIndex
	if selectedLineStart <= r.ViewRange.start {
		r.ViewRange.start = selectedLineStart
		r.ViewRange.end = selectedLineStart + r.Height
	} else if selectedLineEnd > r.ViewRange.end {
		r.ViewRange.end = selectedLineEnd
		r.ViewRange.start = selectedLineEnd - r.Height
	}

	return r.String(r.ViewRange.start, r.ViewRange.end)
}

func (r *Renderer) skipLines(amount int) {
	r.skippedLineCount = r.skippedLineCount + amount
}

func (r *Renderer) totalLineCount() int {
	return r.lineCount + r.skippedLineCount
}
