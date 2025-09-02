package list

import (
	"bytes"
	"io"
	"strings"

	"github.com/idursun/jjui/internal/ui/common"
)

type ItemRenderFunc func(w io.Writer, index int)

type ListRenderer[T any] struct {
	*common.ViewRange
	list             *List[T]
	renderItemFn     ItemRenderFunc
	getItemHeight    func(index int) int
	buffer           bytes.Buffer
	skippedLineCount int
	lineCount        int
}

func NewRenderer[T any](list *List[T], renderFn ItemRenderFunc, getItemHeight func(index int) int, size *common.Sizeable) *ListRenderer[T] {
	return &ListRenderer[T]{
		ViewRange:     &common.ViewRange{Sizeable: size, Start: 0, End: size.Height, FirstRowIndex: -1, LastRowIndex: -1},
		list:          list,
		renderItemFn:  renderFn,
		getItemHeight: getItemHeight,
		buffer:        bytes.Buffer{},
	}
}

func (r *ListRenderer[T]) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	r.lineCount += bytes.Count(p, []byte("\n"))
	return r.buffer.Write(p)
}

func (r *ListRenderer[T]) String(start, end int) string {
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

func (r *ListRenderer[T]) Reset() {
	r.buffer.Reset()
	r.ViewRange.Reset()
	r.lineCount = 0
	r.skippedLineCount = 0
}

func (r *ListRenderer[T]) Render() string {
	r.Reset()
	viewHeight := r.End - r.Start
	if viewHeight != r.Height {
		r.End = r.Start + r.Height
	}

	selectedLineStart := -1
	selectedLineEnd := -1
	firstRenderedRowIndex := -1
	lastRenderedRowIndex := -1
	for i := range r.list.Items {
		isHighlighted := i == r.list.Cursor
		if isHighlighted {
			selectedLineStart = r.totalLineCount()
		} else {
			rowLineCount := r.getItemHeight(i)
			if rowLineCount+r.totalLineCount() < r.Start {
				r.skipLines(rowLineCount)
				continue
			}
		}
		r.renderItemFn(r, i)
		if firstRenderedRowIndex == -1 {
			firstRenderedRowIndex = i
		}

		if isHighlighted {
			selectedLineEnd = r.totalLineCount()
		}
		if selectedLineEnd > 0 && r.totalLineCount() > r.End {
			lastRenderedRowIndex = i
			break
		}
	}

	if lastRenderedRowIndex == -1 {
		lastRenderedRowIndex = len(r.list.Items) - 1
	}

	r.FirstRowIndex = firstRenderedRowIndex
	r.LastRowIndex = lastRenderedRowIndex
	if selectedLineStart <= r.Start {
		r.Start = selectedLineStart
		r.End = selectedLineStart + r.Height
	} else if selectedLineEnd > r.End {
		r.End = selectedLineEnd
		r.Start = selectedLineEnd - r.Height
	}

	return r.String(r.Start, r.End)
}

func (r *ListRenderer[T]) skipLines(amount int) {
	r.skippedLineCount = r.skippedLineCount + amount
}

func (r *ListRenderer[T]) totalLineCount() int {
	return r.lineCount + r.skippedLineCount
}
