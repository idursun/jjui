package list

import (
	"bytes"
	"io"

	"github.com/idursun/jjui/internal/ui/common"
)

type ListRenderer struct {
	*common.ViewRange
	list             IList
	buffer           bytes.Buffer
	skippedLineCount int // lines skipped before the rendered window
	lineCount        int // number of lines we actually rendered (post-skipping)
	rowRanges        []RowRange
	absoluteLines    int // total lines including skipped content (for scrolling/clicks)
}

func NewRenderer(list IList, size *common.ViewNode) *ListRenderer {
	return &ListRenderer{
		ViewRange: &common.ViewRange{ViewNode: size, Start: 0, FirstRowIndex: -1, LastRowIndex: -1},
		list:      list,
		buffer:    bytes.Buffer{},
	}
}

func (r *ListRenderer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	lines := bytes.Count(p, []byte("\n"))
	r.lineCount += lines
	r.absoluteLines += lines
	return r.buffer.Write(p)
}

func (r *ListRenderer) String() string {
	return r.buffer.String()
}

func (r *ListRenderer) Reset() {
	r.buffer.Reset()
	r.lineCount = 0
	r.skippedLineCount = 0
	r.absoluteLines = 0
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
	r.absoluteLines = 0
	listLen := r.list.Len()
	if listLen == 0 || r.Height <= 0 {
		return ""
	}

	if opts.FocusIndex < 0 {
		opts.FocusIndex = 0
	}
	if opts.FocusIndex >= listLen {
		opts.FocusIndex = listLen - 1
	}

	start := r.Start
	if start < 0 {
		start = 0
	}
	if opts.EnsureFocusVisible {
		focusStart := 0
		focusHeight := 0
		for i := 0; i <= opts.FocusIndex && i < listLen; i++ {
			h := r.list.GetItemRenderer(i).Height()
			if i == opts.FocusIndex {
				focusHeight = h
				break
			}
			focusStart += h
		}
		focusEnd := focusStart + focusHeight
		if focusStart < start {
			start = focusStart
		}
		if focusEnd > start+r.Height {
			start = focusEnd - r.Height
		}
		if start < 0 {
			start = 0
		}
	}

	r.Start = start

	firstRenderedRowIndex := -1
	lastRenderedRowIndex := -1
	focusRendered := false

	for i := 0; i < listLen; i++ {
		itemRenderer := r.list.GetItemRenderer(i)
		rowHeight := itemRenderer.Height()

		rowStart := r.absoluteLines
		rowEnd := rowStart + rowHeight

		overlaps := rowEnd > r.Start && rowStart < r.Start+r.Height
		if !overlaps {
			if rowEnd <= r.Start {
				r.skipLines(rowHeight)
			} else {
				r.addAbsolute(rowHeight)
			}
		} else {
			preSkip := 0
			if rowStart < r.Start {
				preSkip = r.Start - rowStart
			}
			overlapStart := rowStart + preSkip
			overlapEnd := min(rowEnd, r.Start+r.Height)
			renderLines := overlapEnd - overlapStart
			postSkip := rowEnd - overlapEnd

			if preSkip > 0 {
				r.skipLines(preSkip)
			}

			if renderLines > 0 {
				writer := newCursorWriter(cursorWriterConfig{
					dst:            r,
					skipLines:      preSkip,
					remainingLines: renderLines,
					rowStart:       rowStart,
					viewportStart:  r.Start,
					viewportHeight: r.Height,
				})
				itemRenderer.Render(writer, r.ViewRange.Width)

				if firstRenderedRowIndex == -1 {
					firstRenderedRowIndex = i
				}
				lastRenderedRowIndex = i
				r.rowRanges = append(r.rowRanges, RowRange{
					Row:       i,
					StartLine: overlapStart,
					EndLine:   overlapEnd,
				})

				if opts.EnsureFocusVisible && i == opts.FocusIndex {
					focusRendered = true
				}
			}

			if postSkip > 0 {
				r.addAbsolute(postSkip)
			}
		}

		if r.lineCount >= r.Height && (!opts.EnsureFocusVisible || focusRendered) {
			for j := i + 1; j < listLen; j++ {
				r.addAbsolute(r.list.GetItemRenderer(j).Height())
			}
			break
		}
	}

	if lastRenderedRowIndex == -1 {
		lastRenderedRowIndex = listLen - 1
	}

	r.FirstRowIndex = firstRenderedRowIndex
	r.LastRowIndex = lastRenderedRowIndex

	visibleHeight := r.Height
	if r.lineCount < visibleHeight {
		visibleHeight = r.lineCount
	}

	return r.String()
}

func (r *ListRenderer) skipLines(amount int) {
	r.skippedLineCount = r.skippedLineCount + amount
	r.absoluteLines += amount
}

func (r *ListRenderer) addAbsolute(amount int) {
	r.absoluteLines += amount
}

func (r *ListRenderer) TotalLineCount() int {
	return r.lineCount
}

func (r *ListRenderer) AbsoluteLineCount() int {
	return r.absoluteLines
}

type RowRange struct {
	Row       int
	StartLine int
	EndLine   int
}

func (r *ListRenderer) RowRanges() []RowRange {
	return r.rowRanges
}

// skipWriter discards the first N lines before forwarding to the underlying writer.
type skipWriter struct {
	dst         io.Writer
	linesToSkip int
}

// limitWriter stops forwarding after a fixed number of lines to keep rendered output within the viewport.
type limitWriter struct {
	dst       io.Writer
	remaining int
}

func (l *limitWriter) Write(p []byte) (n int, err error) {
	if l.remaining <= 0 {
		return len(p), nil
	}

	start := 0
	lines := 0
	for i, b := range p {
		if b == '\n' {
			lines++
			if lines > l.remaining {
				if i > start {
					_, err = l.dst.Write(p[start:i])
				}
				l.remaining = 0
				return len(p), err
			}
		}
	}
	if len(p) > start {
		_, err = l.dst.Write(p[start:])
	}
	l.remaining -= lines
	return len(p), err
}

func (s *skipWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	if s.linesToSkip <= 0 {
		_, err = s.dst.Write(p)
		return
	}

	start := 0
	for i, b := range p {
		if s.linesToSkip == 0 {
			start = i
			break
		}
		if b == '\n' {
			s.linesToSkip--
		}
		start = i + 1
	}

	if s.linesToSkip == 0 && start < len(p) {
		_, err = s.dst.Write(p[start:])
	}
	return
}

type cursorWriterConfig struct {
	dst            io.Writer
	skipLines      int
	remainingLines int
	rowStart       int
	viewportStart  int
	viewportHeight int
}

// cursorWriter tracks local and viewport coordinates while forwarding writes to
// the underlying destination. It enforces skipping and line limits without
// ListRenderer knowing about item-specific layout.
type cursorWriter struct {
	cfg       cursorWriterConfig
	localLine int
	localCol  int
}

func newCursorWriter(cfg cursorWriterConfig) *cursorWriter {
	return &cursorWriter{cfg: cfg, localLine: 0, localCol: 0}
}

func (c *cursorWriter) LocalPos() (line, col int) {
	return c.localLine, c.localCol
}

func (c *cursorWriter) ViewportPos() (line, col int) {
	absoluteLine := c.cfg.rowStart + c.localLine
	screenLine := absoluteLine - c.cfg.viewportStart
	if screenLine < 0 || screenLine >= c.cfg.viewportHeight {
		return -1, -1
	}
	return screenLine, c.localCol
}

func (c *cursorWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	start := 0

	for i, b := range p {
		if b != '\n' {
			continue
		}

		// line with newline
		lineBytes := p[start : i+1]
		c.handleLine(lineBytes, true)
		start = i + 1
	}

	// trailing content without newline
	if start < len(p) {
		tail := p[start:]
		c.handleLine(tail, false)
	}

	return n, err
}

func (c *cursorWriter) handleLine(line []byte, hasNewline bool) {
	write := false
	if c.cfg.skipLines > 0 {
		c.cfg.skipLines--
	} else if c.cfg.remainingLines != 0 {
		write = true
	}

	if write {
		_, _ = c.cfg.dst.Write(line)
		if hasNewline && c.cfg.remainingLines > 0 {
			c.cfg.remainingLines--
		}
	}

	if hasNewline {
		// exclude newline character from column
		c.localCol += len(line) - 1
		c.localLine++
		c.localCol = 0
	} else {
		c.localCol += len(line)
	}
}
