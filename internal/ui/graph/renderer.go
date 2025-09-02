package graph

import (
	"bytes"
	"strings"

	"github.com/idursun/jjui/internal/ui/common"
)

type Renderer struct {
	*common.ViewRange
	buffer           bytes.Buffer
	skippedLineCount int
	lineCount        int
}

func NewRenderer(width int, height int) *Renderer {
	return &Renderer{
		ViewRange: &common.ViewRange{Sizeable: common.NewSizeable(width, height), Start: 0, End: height, FirstRowIndex: -1, LastRowIndex: -1},
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
	viewHeight := r.ViewRange.End - r.ViewRange.Start
	if viewHeight != r.Height {
		r.ViewRange.End = r.ViewRange.Start + r.Height
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
			if rowLineCount+r.totalLineCount() < r.ViewRange.Start {
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
		if selectedLineEnd > 0 && r.totalLineCount() > r.ViewRange.End {
			lastRenderedRowIndex = i
			break
		}
	}
	if lastRenderedRowIndex == -1 {
		lastRenderedRowIndex = iterator.Len() - 1
	}

	r.ViewRange.FirstRowIndex = firstRenderedRowIndex
	r.ViewRange.LastRowIndex = lastRenderedRowIndex
	if selectedLineStart <= r.ViewRange.Start {
		r.ViewRange.Start = selectedLineStart
		r.ViewRange.End = selectedLineStart + r.Height
	} else if selectedLineEnd > r.ViewRange.End {
		r.ViewRange.End = selectedLineEnd
		r.ViewRange.Start = selectedLineEnd - r.Height
	}

	return r.String(r.ViewRange.Start, r.ViewRange.End)
}

func (r *Renderer) skipLines(amount int) {
	r.skippedLineCount = r.skippedLineCount + amount
}

func (r *Renderer) totalLineCount() int {
	return r.lineCount + r.skippedLineCount
}
