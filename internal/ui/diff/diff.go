package diff

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type viewMode interface {
	totalLines(width int) int
	scrollHorizontal(delta int, viewportWidth int)
	ViewRect(dl *render.DisplayContext, box layout.Box, scrollY int)
}

type defaultView struct {
	lines        []string
	maxLineWidth int
	scrollX      int
}

func newDefaultView(lines []string, maxLineWidth int) *defaultView {
	return &defaultView{
		lines:        lines,
		maxLineWidth: maxLineWidth,
	}
}

func (v *defaultView) totalLines(_ int) int {
	return len(v.lines)
}

func (v *defaultView) scrollHorizontal(delta int, viewportWidth int) {
	maxScroll := max(0, v.maxLineWidth-viewportWidth)
	v.scrollX = max(0, min(v.scrollX+delta, maxScroll))
}

func (v *defaultView) ViewRect(dl *render.DisplayContext, box layout.Box, scrollY int) {
	width := box.R.Dx()
	height := box.R.Dy()
	buf := uv.NewScreenBuffer(width, height)
	firstLine := max(0, scrollY)
	lineW := max(width, v.maxLineWidth)
	for i := 0; i < height; i++ {
		physLine := firstLine + i
		if physLine >= len(v.lines) {
			break
		}
		ss := uv.NewStyledString(v.lines[physLine])
		ss.Wrap = false
		ss.Draw(buf, uv.Rect(-v.scrollX, i, lineW, 1))
	}
	dl.AddDraw(box.R, buf.Render(), 0)
}

type wrappedView struct {
	lines           []string
	rowHeights      []int
	visualRowStart  []int
	totalVisualRows int
	cachedWidth     int
}

func newWrappedView(lines []string) *wrappedView {
	return &wrappedView{lines: lines}
}

func (v *wrappedView) recomputeIndex(width int) {
	if width <= 0 {
		return
	}
	v.rowHeights = make([]int, len(v.lines))
	v.visualRowStart = make([]int, len(v.lines))
	total := 0
	for i, line := range v.lines {
		visWidth := lipgloss.Width(line)
		h := max(1, (visWidth+width-1)/width)
		v.rowHeights[i] = h
		v.visualRowStart[i] = total
		total += h
	}
	v.totalVisualRows = total
	v.cachedWidth = width
}

func (v *wrappedView) ensureIndex(width int) {
	if width <= 0 {
		return
	}
	if width != v.cachedWidth || len(v.rowHeights) != len(v.lines) {
		v.recomputeIndex(width)
	}
}

func (v *wrappedView) totalLines(width int) int {
	v.ensureIndex(width)
	return v.totalVisualRows
}

func (v *wrappedView) firstLine(scrollY int, width int) (line int, skip int) {
	v.ensureIndex(width)
	n := len(v.visualRowStart)
	if n == 0 {
		return 0, 0
	}
	idx := 0
	for idx+1 < n && v.visualRowStart[idx+1] <= scrollY {
		idx++
	}
	return idx, scrollY - v.visualRowStart[idx]
}

func (v *wrappedView) scrollHorizontal(_ int, _ int) {}

func (v *wrappedView) ViewRect(dl *render.DisplayContext, box layout.Box, scrollY int) {
	width := box.R.Dx()
	height := box.R.Dy()
	v.ensureIndex(width)
	buf := uv.NewScreenBuffer(width, height)
	firstLine, skip := v.firstLine(scrollY, width)
	destY := 0
	for i := firstLine; i < len(v.lines) && destY < height; i++ {
		lh := 1
		if i < len(v.rowHeights) {
			lh = v.rowHeights[i]
		}
		visibleRows := min(lh-skip, height-destY)
		if visibleRows <= 0 {
			break
		}
		ss := uv.NewStyledString(v.lines[i])
		ss.Wrap = true
		y0 := destY - skip
		ss.Draw(buf, uv.Rect(0, y0, width, skip+visibleRows))
		destY += visibleRows
		skip = 0
	}
	dl.AddDraw(box.R, buf.Render(), 0)
}

type Model struct {
	lines        []string
	maxLineWidth int

	scrollY        int
	viewportWidth  int
	viewportHeight int

	mode viewMode
}

func (m *Model) Init() tea.Cmd {
	return nil
}

type ScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (s ScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	s.Delta = delta
	s.Horizontal = horizontal
	return s
}

func (m *Model) clampScroll(width, height int) {
	total := m.mode.totalLines(width)
	m.scrollY = max(0, min(m.scrollY, max(0, total-height)))
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.DiffScroll:
		switch msg.Kind {
		case intents.DiffScrollUp:
			m.scrollY -= 1
		case intents.DiffScrollDown:
			m.scrollY += 1
		case intents.DiffPageUp:
			m.scrollY -= m.viewportHeight
		case intents.DiffPageDown:
			m.scrollY += m.viewportHeight
		case intents.DiffHalfPageUp:
			m.scrollY -= m.viewportHeight / 2
		case intents.DiffHalfPageDown:
			m.scrollY += m.viewportHeight / 2
		}
		return nil

	case intents.DiffToggleWrap:
		switch m.mode.(type) {
		case *wrappedView:
			m.mode = newDefaultView(m.lines, m.maxLineWidth)
		default:
			m.mode = newWrappedView(m.lines)
		}
		return nil

	case intents.DiffScrollHorizontal:
		switch msg.Kind {
		case intents.DiffScrollLeft:
			m.mode.scrollHorizontal(-1, m.viewportWidth)
		case intents.DiffScrollRight:
			m.mode.scrollHorizontal(1, m.viewportWidth)
		}
		return nil

	case ScrollMsg:
		if !msg.Horizontal {
			m.scrollY += msg.Delta
		} else {
			m.mode.scrollHorizontal(msg.Delta, m.viewportWidth)
		}
		return nil
	}
	return nil
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	width := box.R.Dx()
	height := box.R.Dy()
	m.viewportWidth = width
	m.viewportHeight = height
	m.clampScroll(width, height)

	m.mode.ViewRect(dl, box, m.scrollY)
	dl.AddInteraction(box.R, ScrollMsg{}, render.InteractionScroll, 0)
}

func New(output string) *Model {
	content := strings.ReplaceAll(output, "\r", "")
	if content == "" {
		content = "(empty)"
	}
	lines := strings.Split(content, "\n")
	maxWidth := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > maxWidth {
			maxWidth = w
		}
	}

	model := &Model{
		lines:        lines,
		maxLineWidth: maxWidth,
		mode:         newDefaultView(lines, maxWidth),
	}
	return model
}
