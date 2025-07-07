package graph

import (
	"bytes"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
)

type viewRange struct {
	start        int
	end          int
	lastRowIndex int
}

func (v *viewRange) reset() {
	v.start = 0
	v.end = 0
	v.lastRowIndex = -1
}

type rowMetadata struct {
	index     int
	startLine int
}

type Renderer struct {
	buffer           bytes.Buffer
	viewRange        *viewRange
	rowMetadatas     []rowMetadata
	skippedLineCount int
	lineCount        int
	Width            int
	Height           int
}

func NewRenderer(width int, height int) *Renderer {
	return &Renderer{
		buffer:    bytes.Buffer{},
		viewRange: &viewRange{},
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

func (r *Renderer) FindRowIndexAtLine(line int) int {
	if line < 0 || line >= r.LineCount() {
		return -1
	}
	for i, metadata := range r.rowMetadatas {
		if metadata.startLine <= line && (i == len(r.rowMetadatas)-1 || r.rowMetadatas[i+1].startLine > line) {
			return metadata.index
		}
	}
	return -1
}

func (r *Renderer) Reset() {
	r.buffer.Reset()
	r.rowMetadatas = make([]rowMetadata, 0)
	r.lineCount = 0
	r.skippedLineCount = 0
}

func (r *Renderer) Render(iterator RowIterator) string {
	r.Reset()
	h := r.Height
	viewHeight := r.viewRange.end - r.viewRange.start
	if viewHeight != h {
		r.viewRange.end = r.viewRange.start + h
	}

	selectedLineStart := -1
	selectedLineEnd := -1
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
		currentLine := r.LineCount()
		r.rowMetadatas = append(r.rowMetadatas, rowMetadata{index: i, startLine: currentLine})
		iterator.Render(r)

		if iterator.IsHighlighted() {
			selectedLineEnd = r.LineCount()
		}
		if selectedLineEnd > 0 && r.LineCount() > r.Height && r.LineCount() > r.viewRange.end {
			lastRenderedRowIndex = i
			break
		}
	}
	if lastRenderedRowIndex == -1 {
		lastRenderedRowIndex = iterator.Len() - 1
	}

	r.viewRange.lastRowIndex = lastRenderedRowIndex
	if selectedLineStart <= r.viewRange.start {
		r.viewRange.start = selectedLineStart
		r.viewRange.end = selectedLineStart + r.Height
	} else if selectedLineEnd > r.viewRange.end {
		r.viewRange.end = selectedLineEnd
		r.viewRange.start = selectedLineEnd - r.Height
	}

	content := r.String(r.viewRange.start, r.viewRange.end)
	content = lipgloss.PlaceHorizontal(r.Width, lipgloss.Left, content)

	return common.DefaultPalette.Normal.MaxWidth(r.Width).Render(content)
}
