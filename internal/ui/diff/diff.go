package diff

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/jj/source"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations/target_picker"
	"github.com/idursun/jjui/internal/ui/render"
)

type viewMode interface {
	totalLines(width int) int
	scrollHorizontal(delta int, viewportWidth int)
	ViewRect(dl *render.DisplayContext, box layout.Box, scrollY int)
}

const allFilesTargetLabel = "(all files)"

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
	surfaceStyle := common.DefaultPalette.Get("diff")
	buf := render.NewScreenBuffer(width, height)
	firstLine := max(0, scrollY)
	lineW := max(width, v.maxLineWidth)
	for i := range height {
		physLine := firstLine + i
		if physLine >= len(v.lines) {
			break
		}
		ss := uv.NewStyledString(v.lines[physLine])
		ss.Wrap = false
		ss.Draw(buf, uv.Rect(-v.scrollX, i, lineW, 1))
	}
	dl.AddFill(box.R, ' ', surfaceStyle, 0)
	dl.AddDraw(box.R, buf.Render(), 0, render.PreserveBackground())
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
		visWidth := render.StringWidth(line)
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
	surfaceStyle := common.DefaultPalette.Get("diff")
	v.ensureIndex(width)
	buf := render.NewScreenBuffer(width, height)
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
	dl.AddFill(box.R, ' ', surfaceStyle, 0)
	dl.AddDraw(box.R, buf.Render(), 0, render.PreserveBackground())
}

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	context      *appContext.MainContext
	originalArgs []string
	targetFiles  []string
	targetLoaded bool
	targetErr    error

	lines        []string
	maxLineWidth int

	scrollY        int
	viewportWidth  int
	viewportHeight int

	mode viewMode
}

type targetPickerPayload struct{}

type summaryLoadedMsg struct {
	args  []string
	files []string
	err   error
}

type fileLoadedMsg struct {
	content string
	err     error
}

func (m *Model) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeDiff,
			Leak:    common.LeakGlobal,
			Handler: m,
		},
	}
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch msg := intent.(type) {
	case intents.Cancel:
		return common.Close, true
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
		case intents.DiffMoveTop:
			m.scrollY = 0
		case intents.DiffMoveBottom:
			m.scrollY = max(0, m.mode.totalLines(m.viewportWidth)-m.viewportHeight)
		}
		return nil, true

	case intents.DiffToggleWrap:
		switch m.mode.(type) {
		case *wrappedView:
			m.mode = newDefaultView(m.lines, m.maxLineWidth)
		default:
			m.mode = newWrappedView(m.lines)
		}
		return nil, true

	case intents.DiffShow:
		m.SetContent(msg.Content)
		m.originalArgs = append([]string(nil), msg.Args...)
		m.targetFiles = nil
		m.targetLoaded = false
		m.targetErr = nil
		return m.Init(), true

	case intents.DiffOpenTargetPicker:
		return m.openTargetPicker(), true

	case intents.DiffScrollHorizontal:
		switch msg.Kind {
		case intents.DiffScrollLeft:
			m.mode.scrollHorizontal(-1, m.viewportWidth)
		case intents.DiffScrollRight:
			m.mode.scrollHorizontal(1, m.viewportWidth)
		}
		return nil, true
	}
	return nil, false
}

func (m *Model) Init() tea.Cmd {
	if len(m.originalArgs) == 0 || m.context == nil {
		return nil
	}
	originalArgs := append([]string(nil), m.originalArgs...)
	args := append(append([]string(nil), originalArgs...), "--summary")
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(args)
		if err != nil {
			return summaryLoadedMsg{args: originalArgs, err: err}
		}
		seen := map[string]bool{}
		var files []string
		for _, line := range strings.Split(string(output), "\n") {
			summary, ok := jj.ParseSummaryFile(line)
			if !ok || summary.FileName == "" || seen[summary.FileName] {
				continue
			}
			seen[summary.FileName] = true
			files = append(files, summary.FileName)
		}
		return summaryLoadedMsg{args: originalArgs, files: files}
	}
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

func (m *Model) SetContent(content string) {
	wrapped := false
	if m.mode != nil {
		_, wrapped = m.mode.(*wrappedView)
	}

	content = strings.ReplaceAll(content, "\r", "")
	if content == "" {
		content = "(empty)"
	}

	rawLines := strings.Split(content, "\n")
	lines := make([]string, len(rawLines))
	maxWidth := 0
	for i, line := range rawLines {
		line = render.ExpandTabs(line)
		lines[i] = line
		if w := render.StringWidth(line); w > maxWidth {
			maxWidth = w
		}
	}

	m.lines = lines
	m.maxLineWidth = maxWidth
	m.scrollY = 0

	if wrapped {
		m.mode = newWrappedView(lines)
		return
	}
	m.mode = newDefaultView(lines, maxWidth)
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.DiffScroll, intents.DiffToggleWrap, intents.DiffShow, intents.DiffOpenTargetPicker, intents.DiffScrollHorizontal:
		cmd, _ := m.HandleIntent(msg.(intents.Intent))
		return cmd

	case ScrollMsg:
		if !msg.Horizontal {
			m.scrollY += msg.Delta
		} else {
			m.mode.scrollHorizontal(msg.Delta, m.viewportWidth)
		}
		return nil
	case summaryLoadedMsg:
		if !slices.Equal(msg.args, m.originalArgs) {
			return nil
		}
		m.targetFiles = append([]string(nil), msg.files...)
		m.targetLoaded = msg.err == nil
		m.targetErr = msg.err
		return nil
	case fileLoadedMsg:
		if msg.err != nil {
			return intents.Invoke(intents.AddMessage{Text: msg.err.Error(), Err: msg.err})
		}
		m.SetContent(msg.content)
		return nil
	case target_picker.TargetSelectedMsg:
		if _, ok := msg.Payload.(targetPickerPayload); !ok {
			return nil
		}
		if msg.Target == allFilesTargetLabel {
			msg.Target = ""
		}
		return m.loadFile(msg.Target)
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
	return NewWithContext(nil, output, nil)
}

func NewWithContext(ctx *appContext.MainContext, output string, args []string) *Model {
	model := &Model{}
	model.context = ctx
	model.originalArgs = append([]string(nil), args...)
	model.SetContent(output)
	return model
}

func (m *Model) openTargetPicker() tea.Cmd {
	if len(m.originalArgs) == 0 || m.context == nil {
		return intents.Invoke(intents.AddMessage{Text: "File picker is unavailable for this diff"})
	}
	if m.targetErr != nil {
		return intents.Invoke(intents.AddMessage{Text: m.targetErr.Error(), Err: m.targetErr})
	}
	if !m.targetLoaded {
		return intents.Invoke(intents.AddMessage{Text: "File picker is still loading"})
	}
	if len(m.targetFiles) == 0 {
		return intents.Invoke(intents.AddMessage{Text: "No files found in diff summary"})
	}
	files := append([]string{allFilesTargetLabel}, m.targetFiles...)
	return common.OpenTargetPickerWithPayload(targetPickerPayload{}, source.FileSource{Files: files})
}

func (m *Model) loadFile(file string) tea.Cmd {
	if len(m.originalArgs) == 0 || m.context == nil {
		return intents.Invoke(intents.AddMessage{Text: "File picker is unavailable for this diff"})
	}
	args := append([]string(nil), m.originalArgs...)
	if file != "" {
		args = append(args, jj.EscapeFileName(file))
	}
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(args)
		if err != nil {
			return fileLoadedMsg{err: err}
		}
		return fileLoadedMsg{content: string(output)}
	}
}
