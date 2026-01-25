package diff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ common.ImmediateModel = (*Model)(nil)

// Model represents the diff viewer
type Model struct {
	// Content
	revision      string
	focusFiles    []string
	rawContent    string // For raw content display (fallback mode)
	isRawMode     bool   // Whether we're showing raw content vs structured diff
	parsedDiff    *ParsedDiff
	context       *context.MainContext
	keymap        config.KeyMappings[key.Binding]

	// UI state
	startLine      int  // First visible line
	height         int  // Viewport height (cached from ViewRect for Update)
	fileListWidth  int  // Width of file list panel
	showFileList   bool // Whether file list is visible
	wordWrap       bool // Whether word wrap is enabled
	fileList       *FileList
	selectedFileIdx int // Index of currently viewed file

	// Computed line data
	lines        []viewLine // All rendered lines
	hunkStarts   []int      // Line indices where hunks start (for navigation)
	fileStarts   []int      // Line indices where files start (for navigation)
}

// viewLine represents a single line in the diff view
type viewLine struct {
	fileIdx    int
	hunkIdx    int
	lineIdx    int
	lineType   lineViewType
	oldLineNo  int
	newLineNo  int
	content    string
	segments   []Segment
	isHunkHead bool
	isFileHead bool
}

type lineViewType int

const (
	lineViewContext lineViewType = iota
	lineViewAdded
	lineViewRemoved
	lineViewHunkHeader
	lineViewFileHeader
	lineViewEmpty
)

// ShortHelp returns the short help bindings
func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keymap.Up,
		m.keymap.Down,
		m.keymap.DiffViewer.HalfPageUp,
		m.keymap.DiffViewer.HalfPageDown,
		m.keymap.DiffViewer.NextHunk,
		m.keymap.DiffViewer.PrevHunk,
		m.keymap.DiffViewer.NextFile,
		m.keymap.DiffViewer.PrevFile,
		m.keymap.DiffViewer.ToggleFileList,
		m.keymap.DiffViewer.ToggleWordWrap,
		m.keymap.Cancel,
	}
}

// FullHelp returns the full help bindings
func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	if m.isRawMode {
		// Raw content mode - no additional loading needed
		return nil
	}
	// Load structured diff
	return m.loadDiff()
}

func (m *Model) loadDiff() tea.Cmd {
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(jj.DiffGit(m.revision, m.focusFiles))
		if err != nil {
			return diffLoadedMsg{err: err}
		}
		return diffLoadedMsg{content: string(output)}
	}
}

type diffLoadedMsg struct {
	content string
	err     error
}

// Scroll scrolls the view by delta lines
func (m *Model) Scroll(delta int) tea.Cmd {
	m.startLine += delta
	m.clampStartLine()
	return nil
}

func (m *Model) clampStartLine() {
	maxStart := len(m.lines) - m.height
	if maxStart < 0 {
		maxStart = 0
	}
	if m.startLine > maxStart {
		m.startLine = maxStart
	}
	if m.startLine < 0 {
		m.startLine = 0
	}
}

// ScrollMsg is sent when scrolling via mouse
type ScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (s ScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	s.Delta = delta
	s.Horizontal = horizontal
	return s
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return common.Close
		case key.Matches(msg, m.keymap.Up):
			m.startLine--
			m.clampStartLine()
		case key.Matches(msg, m.keymap.Down):
			m.startLine++
			m.clampStartLine()
		case key.Matches(msg, m.keymap.ScrollUp, m.keymap.DiffViewer.HalfPageUp):
			m.startLine -= m.height / 2
			m.clampStartLine()
		case key.Matches(msg, m.keymap.ScrollDown, m.keymap.DiffViewer.HalfPageDown):
			m.startLine += m.height / 2
			m.clampStartLine()
		case key.Matches(msg, m.keymap.DiffViewer.NextHunk):
			m.navigateToNextHunk()
		case key.Matches(msg, m.keymap.DiffViewer.PrevHunk):
			m.navigateToPrevHunk()
		case key.Matches(msg, m.keymap.DiffViewer.NextFile):
			m.navigateToNextFile()
		case key.Matches(msg, m.keymap.DiffViewer.PrevFile):
			m.navigateToPrevFile()
		case key.Matches(msg, m.keymap.DiffViewer.ToggleFileList):
			m.showFileList = !m.showFileList
		case key.Matches(msg, m.keymap.DiffViewer.ToggleWordWrap):
			m.wordWrap = !m.wordWrap
		case key.Matches(msg, m.keymap.DiffViewer.GoToTop):
			m.startLine = 0
		case key.Matches(msg, m.keymap.DiffViewer.GoToBottom):
			m.startLine = len(m.lines) - m.height
			m.clampStartLine()
		}
	case ScrollMsg:
		if msg.Horizontal {
			return nil
		}
		return m.Scroll(msg.Delta)
	case diffLoadedMsg:
		if msg.err != nil {
			m.rawContent = fmt.Sprintf("Error loading diff: %v", msg.err)
			return nil
		}
		m.parsedDiff = Parse(msg.content)
		m.fileList = NewFileList(m.parsedDiff.Files)
		m.buildLines()
		// Focus on specific file if requested
		if len(m.focusFiles) > 0 {
			m.focusOnFiles(m.focusFiles)
		}
	case FileSelectedMsg:
		m.selectFile(msg.Index)
	}
	return nil
}

func (m *Model) navigateToNextHunk() {
	if len(m.hunkStarts) == 0 {
		return
	}
	// Find next hunk start after current position
	for _, pos := range m.hunkStarts {
		if pos > m.startLine {
			m.startLine = pos
			m.clampStartLine()
			return
		}
	}
}

func (m *Model) navigateToPrevHunk() {
	if len(m.hunkStarts) == 0 {
		return
	}
	// Find previous hunk start before current position
	for i := len(m.hunkStarts) - 1; i >= 0; i-- {
		if m.hunkStarts[i] < m.startLine {
			m.startLine = m.hunkStarts[i]
			m.clampStartLine()
			return
		}
	}
}

func (m *Model) navigateToNextFile() {
	if len(m.fileStarts) == 0 || m.fileList == nil {
		return
	}
	m.fileList.MoveDown()
	m.selectFile(m.fileList.SelectedIndex())
}

func (m *Model) navigateToPrevFile() {
	if len(m.fileStarts) == 0 || m.fileList == nil {
		return
	}
	m.fileList.MoveUp()
	m.selectFile(m.fileList.SelectedIndex())
}

func (m *Model) selectFile(idx int) {
	if idx < 0 || idx >= len(m.fileStarts) {
		return
	}
	m.selectedFileIdx = idx
	if m.fileList != nil {
		m.fileList.SetSelectedIndex(idx)
	}
	m.startLine = m.fileStarts[idx]
	m.clampStartLine()
}

func (m *Model) focusOnFiles(files []string) {
	for _, file := range files {
		if m.focusOnFile(file) {
			return
		}
	}
}

func (m *Model) focusOnFile(fileName string) bool {
	if m.parsedDiff == nil {
		return false
	}
	for i, file := range m.parsedDiff.Files {
		if file.Path() == fileName || file.OldPath == fileName || file.NewPath == fileName {
			m.selectFile(i)
			return true
		}
	}
	return false
}

func (m *Model) buildLines() {
	m.lines = nil
	m.hunkStarts = nil
	m.fileStarts = nil

	if m.parsedDiff == nil {
		return
	}

	for fileIdx, file := range m.parsedDiff.Files {
		// Record file start
		m.fileStarts = append(m.fileStarts, len(m.lines))

		// File header
		m.lines = append(m.lines, viewLine{
			fileIdx:    fileIdx,
			lineType:   lineViewFileHeader,
			content:    file.Path(),
			isFileHead: true,
		})

		if file.IsBinary {
			m.lines = append(m.lines, viewLine{
				fileIdx:  fileIdx,
				lineType: lineViewEmpty,
				content:  "(binary file)",
			})
			continue
		}

		for hunkIdx, hunk := range file.Hunks {
			// Record hunk start
			m.hunkStarts = append(m.hunkStarts, len(m.lines))

			// Hunk header
			headerContent := fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount)
			if hunk.Header != "" {
				headerContent += " " + hunk.Header
			}
			m.lines = append(m.lines, viewLine{
				fileIdx:    fileIdx,
				hunkIdx:    hunkIdx,
				lineType:   lineViewHunkHeader,
				content:    headerContent,
				isHunkHead: true,
			})

			// Diff lines
			for lineIdx, line := range hunk.Lines {
				var vt lineViewType
				switch line.Type {
				case LineAdded:
					vt = lineViewAdded
				case LineRemoved:
					vt = lineViewRemoved
				default:
					vt = lineViewContext
				}

				m.lines = append(m.lines, viewLine{
					fileIdx:   fileIdx,
					hunkIdx:   hunkIdx,
					lineIdx:   lineIdx,
					lineType:  vt,
					oldLineNo: line.OldLineNo,
					newLineNo: line.NewLineNo,
					content:   line.Content,
					segments:  line.Segments,
				})
			}
		}

		// Empty line between files
		if fileIdx < len(m.parsedDiff.Files)-1 {
			m.lines = append(m.lines, viewLine{
				fileIdx:  fileIdx,
				lineType: lineViewEmpty,
			})
		}
	}
}

// ViewRect renders the diff viewer
func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	m.height = box.R.Dy()
	width := box.R.Dx()

	if m.height <= 0 || width <= 0 {
		return
	}

	// Raw content mode - simple display
	if m.isRawMode {
		m.renderRawContent(dl, box)
		return
	}

	// Structured diff mode
	if m.showFileList && m.fileList != nil && width > 40 {
		// Split layout: file list | diff content
		fileListWidth := min(30, width/4)
		cols := box.H(layout.Fixed(fileListWidth), layout.Fixed(1), layout.Fill(1))
		if len(cols) >= 3 {
			m.fileList.ViewRect(dl, cols[0])
			// Divider
			m.renderDivider(dl, cols[1])
			m.renderDiffContent(dl, cols[2])
		}
	} else {
		m.renderDiffContent(dl, box)
	}

	// Add scroll interaction
	dl.AddInteraction(box.R, ScrollMsg{}, render.InteractionScroll, 0)
}

func (m *Model) renderRawContent(dl *render.DisplayContext, box layout.Box) {
	if len(m.lines) == 0 {
		return
	}

	y := box.R.Min.Y
	contentWidth := box.R.Dx()

	for i := m.startLine; i < len(m.lines) && y < box.R.Max.Y; i++ {
		line := m.lines[i].content
		if len(line) > contentWidth {
			line = line[:contentWidth]
		}
		dl.AddDraw(cellbuf.Rect(box.R.Min.X, y, contentWidth, 1), line, 0)
		y++
	}
}

func (m *Model) renderDivider(dl *render.DisplayContext, box layout.Box) {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	for y := box.R.Min.Y; y < box.R.Max.Y; y++ {
		dl.AddDraw(cellbuf.Rect(box.R.Min.X, y, 1, 1), style.Render("│"), 0)
	}
}

func (m *Model) renderDiffContent(dl *render.DisplayContext, box layout.Box) {
	if len(m.lines) == 0 {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		dl.AddDraw(cellbuf.Rect(box.R.Min.X, box.R.Min.Y, box.R.Dx(), 1), style.Render("(no changes)"), 0)
		return
	}

	// Styles
	gutterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	contextStyle := lipgloss.NewStyle()
	hunkHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	fileHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	addedHighlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Background(lipgloss.Color("22"))
	removedHighlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Background(lipgloss.Color("52"))

	// Line number gutter width (old | new | content)
	gutterWidth := 14 // "     │ XXXX │ " or "XXXX │ XXXX │ "
	contentWidth := box.R.Dx() - gutterWidth
	if contentWidth < 10 {
		gutterWidth = 0
		contentWidth = box.R.Dx()
	}

	y := box.R.Min.Y
	for i := m.startLine; i < len(m.lines) && y < box.R.Max.Y; i++ {
		line := m.lines[i]
		x := box.R.Min.X

		// Render gutter
		var gutter string
		if gutterWidth > 0 {
			switch line.lineType {
			case lineViewAdded:
				gutter = fmt.Sprintf("     │ %4d │ ", line.newLineNo)
			case lineViewRemoved:
				gutter = fmt.Sprintf("%4d │      │ ", line.oldLineNo)
			case lineViewContext:
				gutter = fmt.Sprintf("%4d │ %4d │ ", line.oldLineNo, line.newLineNo)
			default:
				gutter = "     │      │ "
			}
		}

		// Render content
		var contentStr string
		var style lipgloss.Style

		switch line.lineType {
		case lineViewFileHeader:
			style = fileHeaderStyle
			contentStr = line.content
		case lineViewHunkHeader:
			style = hunkHeaderStyle
			contentStr = line.content
		case lineViewAdded:
			style = addedStyle
			contentStr = m.renderLineWithSegments(line, addedStyle, addedHighlightStyle)
		case lineViewRemoved:
			style = removedStyle
			contentStr = m.renderLineWithSegments(line, removedStyle, removedHighlightStyle)
		case lineViewContext:
			style = contextStyle
			contentStr = line.content
		case lineViewEmpty:
			contentStr = line.content
		}

		// Apply style and handle word wrapping
		if len(line.segments) == 0 && line.lineType != lineViewAdded && line.lineType != lineViewRemoved {
			if m.wordWrap {
				contentStr = style.Width(contentWidth).Render(contentStr)
			} else {
				contentStr = style.Render(contentStr)
			}
		} else if m.wordWrap {
			// For segment lines, wrap after rendering
			contentStr = lipgloss.NewStyle().Width(contentWidth).Render(contentStr)
		}

		// Handle multi-line wrapped content
		if m.wordWrap && strings.Contains(contentStr, "\n") {
			wrappedLines := strings.Split(contentStr, "\n")
			for j, wrappedLine := range wrappedLines {
				if y >= box.R.Max.Y {
					break
				}
				x = box.R.Min.X

				// Render gutter only for first wrapped line
				if gutterWidth > 0 {
					if j == 0 {
						dl.AddDraw(cellbuf.Rect(x, y, gutterWidth, 1), gutterStyle.Render(gutter), 0)
					} else {
						// Empty gutter for continuation lines
						dl.AddDraw(cellbuf.Rect(x, y, gutterWidth, 1), gutterStyle.Render("     │      │ "), 0)
					}
					x += gutterWidth
				}

				dl.AddDraw(cellbuf.Rect(x, y, contentWidth, 1), wrappedLine, 0)
				y++
			}
		} else {
			// Single line - let cellbuf handle clipping
			if gutterWidth > 0 {
				dl.AddDraw(cellbuf.Rect(x, y, gutterWidth, 1), gutterStyle.Render(gutter), 0)
				x += gutterWidth
			}

			dl.AddDraw(cellbuf.Rect(x, y, contentWidth, 1), contentStr, 0)
			y++
		}
	}
}

func (m *Model) renderLineWithSegments(line viewLine, baseStyle, highlightStyle lipgloss.Style) string {
	if len(line.segments) == 0 {
		return baseStyle.Render(line.content)
	}

	var result strings.Builder
	for _, seg := range line.segments {
		if seg.Highlight {
			result.WriteString(highlightStyle.Render(seg.Text))
		} else {
			result.WriteString(baseStyle.Render(seg.Text))
		}
	}
	return result.String()
}

// New creates a new diff viewer model
func New(ctx *context.MainContext, revision string, focusFiles []string, rawContent string) *Model {
	// Determine mode: raw content mode if revision is empty (use rawContent for display)
	isRawMode := revision == ""

	m := &Model{
		revision:      revision,
		focusFiles:    focusFiles,
		rawContent:    rawContent,
		isRawMode:     isRawMode,
		context:       ctx,
		keymap:        config.Current.GetKeyMap(),
		showFileList:  true,
		fileListWidth: 30,
	}

	// If in raw mode, build simple line list for scrolling
	if isRawMode {
		content := strings.ReplaceAll(rawContent, "\r", "")
		if content == "" {
			m.lines = []viewLine{{lineType: lineViewEmpty, content: "(empty)"}}
		} else {
			lines := strings.Split(content, "\n")
			m.lines = make([]viewLine, len(lines))
			for i, c := range lines {
				m.lines[i] = viewLine{
					lineType: lineViewContext,
					content:  c,
				}
			}
		}
	}

	return m
}
