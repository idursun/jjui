package graph

import (
	"bytes"
	"strings"
)

type viewRange struct {
	start         int
	end           int
	firstRowIndex int
	lastRowIndex  int
}

func (v *viewRange) reset() {
	v.start = 0
	v.end = 0
	v.firstRowIndex = -1
	v.lastRowIndex = -1
}

type Renderer struct {
	buffer           bytes.Buffer
	viewRange        *viewRange
	skippedLineCount int
	lineCount        int
	Width            int
	Height           int
}

func NewRenderer(width int, height int) *Renderer {
	return &Renderer{
		buffer:    bytes.Buffer{},
		viewRange: &viewRange{start: 0, end: height, firstRowIndex: -1, lastRowIndex: -1},
		Width:     width,
		Height:    height,
	}
}

func (r *Renderer) SetSize(width int, height int) {
	r.Width = width
	r.Height = height
	if r.viewRange.end < r.viewRange.start+r.Height {
		r.viewRange.end = r.viewRange.start + r.Height
	}
}

func (r *Renderer) FirstRowIndex() int {
	return r.viewRange.firstRowIndex
}

func (r *Renderer) LastRowIndex() int {
	return r.viewRange.lastRowIndex
}

func (r *Renderer) ResetViewRange() {
	r.viewRange.reset()
	r.skippedLineCount = 0
	r.lineCount = 0
}

func (r *Renderer) SkipLines(amount int) {
	r.skippedLineCount = r.skippedLineCount + amount
}

func (r *Renderer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	r.lineCount += bytes.Count(p, []byte("\n"))
	return r.buffer.Write(p)
}

func (r *Renderer) LineCount() int {
	return r.lineCount + r.skippedLineCount
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
	r.lineCount = 0
	r.skippedLineCount = 0
}

func (r *Renderer) Render(iterator RowIterator) string {
	r.Reset()
	viewHeight := r.viewRange.end - r.viewRange.start
	if viewHeight != r.Height {
		r.viewRange.end = r.viewRange.start + r.Height
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
			selectedLineStart = r.LineCount()
		} else {
			rowLineCount := iterator.RowHeight()
			if rowLineCount+r.LineCount() < r.viewRange.start {
				r.SkipLines(rowLineCount)
				continue
			}
		}
		iterator.Render(r)
		if firstRenderedRowIndex == -1 {
			firstRenderedRowIndex = i
		}

		if iterator.IsHighlighted() {
			selectedLineEnd = r.LineCount()
		}
		if selectedLineEnd > 0 && r.LineCount() > r.viewRange.end {
			lastRenderedRowIndex = i
			break
		}
	}
	if lastRenderedRowIndex == -1 {
		lastRenderedRowIndex = iterator.Len() - 1
	}

	r.viewRange.firstRowIndex = firstRenderedRowIndex
	r.viewRange.lastRowIndex = lastRenderedRowIndex
	if selectedLineStart <= r.viewRange.start {
		r.viewRange.start = selectedLineStart
		r.viewRange.end = selectedLineStart + r.Height
	} else if selectedLineEnd > r.viewRange.end {
		r.viewRange.end = selectedLineEnd
		r.viewRange.start = selectedLineEnd - r.Height
	}

	return r.String(r.viewRange.start, r.viewRange.end)
}
