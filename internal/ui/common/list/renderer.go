package list

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/ops"
)

type ListRenderer struct {
	*common.ViewRange
	list                IList
	displayList         *ops.DisplayList
	skippedLineCount    int // lines skipped before the rendered window
	lineCount           int // number of lines we actually rendered (post-skipping)
	rowRanges           []RowRange
	absoluteLines       int    // total lines including skipped content (for scrolling/clicks)
	interactionIDPrefix string // prefix for generating interaction IDs
}

func NewRenderer(list IList) *ListRenderer {
	return &ListRenderer{
		ViewRange:           &common.ViewRange{Start: 0, FirstRowIndex: -1, LastRowIndex: -1},
		list:                list,
		displayList:         ops.NewDisplayList(),
		interactionIDPrefix: "list-row",
	}
}

func (r *ListRenderer) SetInteractionIDPrefix(prefix string) {
	r.interactionIDPrefix = prefix
}

func (r *ListRenderer) Reset() {
	r.displayList.Clear()
	r.lineCount = 0
	r.skippedLineCount = 0
	r.absoluteLines = 0
}

type RenderOptions struct {
	Box                layout.Box
	FocusIndex         int
	EnsureFocusVisible bool
}

func (r *ListRenderer) Render(focusIndex int) *ops.DisplayList {
	return r.RenderWithOptions(RenderOptions{FocusIndex: focusIndex, EnsureFocusVisible: true})
}

func (r *ListRenderer) RenderWithOptions(opts RenderOptions) *ops.DisplayList {
	r.Reset()
	r.rowRanges = r.rowRanges[:0]
	r.absoluteLines = 0
	listLen := r.list.Len()
	if listLen == 0 || opts.Box.R.Dy() <= 0 {
		return r.displayList
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
		if focusEnd > start+opts.Box.R.Dy() {
			start = focusEnd - opts.Box.R.Dy()
		}
		if start < 0 {
			start = 0
		}
	}

	r.Start = start

	firstRenderedRowIndex := -1
	lastRenderedRowIndex := -1
	focusRendered := false

	height := opts.Box.R.Dy()
	width := opts.Box.R.Dx()
	for i := 0; i < listLen; i++ {
		itemRenderer := r.list.GetItemRenderer(i)
		rowHeight := itemRenderer.Height()

		rowStart := r.absoluteLines
		rowEnd := rowStart + rowHeight

		overlaps := rowEnd > r.Start && rowStart < r.Start+height
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
			overlapEnd := min(rowEnd, r.Start+height)
			renderLines := overlapEnd - overlapStart
			postSkip := rowEnd - overlapEnd

			if preSkip > 0 {
				r.skipLines(preSkip)
			}

			if renderLines > 0 {
				// Calculate the viewport rectangle for this item's visible portion
				// Use r.lineCount (how many lines we've already rendered) for positioning
				// cellbuf.Rect takes (x, y, width, height) NOT (minX, minY, maxX, maxY)!
				viewportRect := cellbuf.Rect(
					opts.Box.R.Min.X,
					opts.Box.R.Min.Y+r.lineCount,
					width,
					renderLines,
				)

				// Render the item to a temporary DisplayList with a full-height rect
				// (items expect to render at their full height)
				tempDL := ops.NewDisplayList()
				fullHeightRect := cellbuf.Rect(
					opts.Box.R.Min.X,
					0, // Temporary Y position
					opts.Box.R.Max.X,
					rowHeight, // Full height of the item
				)
				itemRenderer.Render(tempDL, fullHeightRect, width)

				// Clip the rendered content and position it in the viewport
				for _, drawOp := range tempDL.DrawOpsList() {
					clippedContent := r.clipContent(drawOp.Content, preSkip, renderLines)
					if clippedContent != "" {
						r.displayList.AddDraw(viewportRect, clippedContent, drawOp.Z)
					}
				}
				// Copy effects (adjust their rects to viewport coordinates if needed)
				for _, effect := range tempDL.EffectsList() {
					r.displayList.AddEffectOp(effect)
				}

				// Add interactive zone for this item
				interactionID := fmt.Sprintf("%s:%d", r.interactionIDPrefix, i)
				r.displayList.AddClickable(viewportRect, interactionID, 1)

				// Track line count for the rendered item
				r.lineCount += renderLines
				r.absoluteLines += renderLines

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

		if r.lineCount >= height && (!opts.EnsureFocusVisible || focusRendered) {
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

	return r.displayList
}

func (r *ListRenderer) skipLines(amount int) {
	r.skippedLineCount = r.skippedLineCount + amount
	r.absoluteLines += amount
}

func (r *ListRenderer) addAbsolute(amount int) {
	r.absoluteLines += amount
}

// clipContent skips the first skipLines and keeps only the next keepLines from the content
func (r *ListRenderer) clipContent(content string, skipLines, keepLines int) string {
	if skipLines == 0 && keepLines <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")

	// Skip lines from the beginning
	if skipLines > 0 {
		if skipLines >= len(lines) {
			return ""
		}
		lines = lines[skipLines:]
	}

	// Keep only the specified number of lines
	if keepLines > 0 && keepLines < len(lines) {
		lines = lines[:keepLines]
	}

	return strings.Join(lines, "\n")
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

