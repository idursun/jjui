package diff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
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

// searchMatch represents a single search match in the diff view
type searchMatch struct {
	lineIdx  int // Index into m.lines
	startCol int // Starting byte position in content
	endCol   int // Ending byte position in content
}

// Model represents the diff viewer
type Model struct {
	// Content
	revision      string
	focusFiles    []string // Files to filter the diff command
	focusFile     string   // Single file to focus cursor on (independent of filtering)
	rawContent    string   // For raw content display (fallback mode)
	isRawMode     bool     // Whether we're showing raw content vs structured diff
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

	// Search state
	searchInput   textinput.Model
	searchQuery   string        // Current active search query
	searchMatches []searchMatch // All matches found
	currentMatch  int           // Index of current match in searchMatches
	isSearching   bool          // true when search input is focused

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
		m.keymap.DiffViewer.Search,
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
		// Handle search input mode
		if m.isSearching {
			switch msg.Type {
			case tea.KeyEnter:
				// Confirm search
				m.searchQuery = m.searchInput.Value()
				m.isSearching = false
				m.searchInput.Blur()
				m.performSearch()
				return nil
			case tea.KeyEsc:
				// Cancel search input (keep existing search if any)
				m.isSearching = false
				m.searchInput.Blur()
				m.searchInput.SetValue(m.searchQuery) // restore previous query
				return nil
			default:
				// Forward to text input
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return cmd
			}
		}

		// Normal mode key handling
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			// If search is active, clear it first
			if m.hasActiveSearch() {
				m.clearSearch()
				return nil
			}
			return common.Close
		case key.Matches(msg, m.keymap.DiffViewer.Search):
			// "/" key - start search input
			m.isSearching = true
			m.searchInput.Focus()
			return nil
		case key.Matches(msg, m.keymap.DiffViewer.NextHunk):
			// "n" key - next match when search active, otherwise next hunk
			if m.hasActiveSearch() {
				m.jumpToNextMatch()
				return nil
			}
			m.navigateToNextHunk()
		case key.Matches(msg, m.keymap.DiffViewer.PrevHunk), key.Matches(msg, m.keymap.DiffViewer.SearchPrev):
			// "N" key - previous match when search active, otherwise prev hunk
			if m.hasActiveSearch() {
				m.jumpToPrevMatch()
				return nil
			}
			m.navigateToPrevHunk()
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
		if m.focusFile != "" {
			m.focusOnFile(m.focusFile)
		} else if len(m.focusFiles) > 0 {
			m.focusOnFiles(m.focusFiles)
		}
	case FileSelectedMsg:
		m.selectFile(msg.Index)
	case TreeToggleMsg:
		if m.fileList != nil {
			m.fileList.ToggleExpand(msg.VisibleIndex)
		}
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
	if idx := m.fileList.SelectedIndex(); idx >= 0 {
		m.selectFile(idx)
	}
}

func (m *Model) navigateToPrevFile() {
	if len(m.fileStarts) == 0 || m.fileList == nil {
		return
	}
	m.fileList.MoveUp()
	if idx := m.fileList.SelectedIndex(); idx >= 0 {
		m.selectFile(idx)
	}
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

// performSearch finds all case-insensitive matches of the search query in m.lines
func (m *Model) performSearch() {
	m.searchMatches = nil
	m.currentMatch = 0

	if m.searchQuery == "" {
		return
	}

	queryLower := strings.ToLower(m.searchQuery)

	for lineIdx, line := range m.lines {
		contentLower := strings.ToLower(line.content)
		startPos := 0
		for {
			idx := strings.Index(contentLower[startPos:], queryLower)
			if idx == -1 {
				break
			}
			matchStart := startPos + idx
			matchEnd := matchStart + len(m.searchQuery)
			m.searchMatches = append(m.searchMatches, searchMatch{
				lineIdx:  lineIdx,
				startCol: matchStart,
				endCol:   matchEnd,
			})
			startPos = matchEnd
		}
	}

	// If matches found, scroll to the first one at or after current position
	if len(m.searchMatches) > 0 {
		for i, match := range m.searchMatches {
			if match.lineIdx >= m.startLine {
				m.currentMatch = i
				m.scrollToMatch(m.searchMatches[i])
				return
			}
		}
		// No match at or after current position, go to first match
		m.currentMatch = 0
		m.scrollToMatch(m.searchMatches[0])
	}
}

// jumpToNextMatch moves to the next match, wrapping around
func (m *Model) jumpToNextMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.currentMatch = (m.currentMatch + 1) % len(m.searchMatches)
	m.scrollToMatch(m.searchMatches[m.currentMatch])
}

// jumpToPrevMatch moves to the previous match, wrapping around
func (m *Model) jumpToPrevMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.currentMatch--
	if m.currentMatch < 0 {
		m.currentMatch = len(m.searchMatches) - 1
	}
	m.scrollToMatch(m.searchMatches[m.currentMatch])
}

// scrollToMatch scrolls the view to show the given match
func (m *Model) scrollToMatch(match searchMatch) {
	// Position the match line roughly in the middle of the viewport
	targetLine := match.lineIdx - m.height/3
	if targetLine < 0 {
		targetLine = 0
	}
	m.startLine = targetLine
	m.clampStartLine()
}

// clearSearch resets the search state
func (m *Model) clearSearch() {
	m.searchQuery = ""
	m.searchMatches = nil
	m.currentMatch = 0
	m.isSearching = false
	m.searchInput.SetValue("")
	m.searchInput.Blur()
}

// hasActiveSearch returns true if there's an active search with matches
func (m *Model) hasActiveSearch() bool {
	return m.searchQuery != ""
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
	width := box.R.Dx()

	if box.R.Dy() <= 0 || width <= 0 {
		return
	}

	// Reserve bottom line for search bar when search is active or searching
	contentBox := box
	if m.isSearching || m.hasActiveSearch() {
		rows := box.V(layout.Fill(1), layout.Fixed(1))
		if len(rows) >= 2 {
			contentBox = rows[0]
			m.renderSearchBar(dl, rows[1])
		}
	}

	m.height = contentBox.R.Dy()

	// Raw content mode - simple display
	if m.isRawMode {
		m.renderRawContent(dl, contentBox)
		return
	}

	// Structured diff mode
	if m.showFileList && m.fileList != nil && width > 40 {
		// Split layout: file list | diff content
		fileListWidth := min(30, width/4)
		cols := contentBox.H(layout.Fixed(fileListWidth), layout.Fixed(1), layout.Fill(1))
		if len(cols) >= 3 {
			m.fileList.ViewRect(dl, cols[0])
			// Divider
			m.renderDivider(dl, cols[1])
			m.renderDiffContent(dl, cols[2])
		}
	} else {
		m.renderDiffContent(dl, contentBox)
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

func (m *Model) renderSearchBar(dl *render.DisplayContext, box layout.Box) {
	width := box.R.Dx()
	y := box.R.Min.Y
	x := box.R.Min.X

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	if m.isSearching {
		// Show search input
		label := labelStyle.Render("/")
		dl.AddDraw(cellbuf.Rect(x, y, 1, 1), label, 0)
		x += 1

		// Render text input
		inputView := m.searchInput.View()
		inputWidth := width - 1
		if inputWidth > 0 {
			dl.AddDraw(cellbuf.Rect(x, y, inputWidth, 1), inputView, 0)
		}
	} else if m.hasActiveSearch() {
		// Show search status
		label := labelStyle.Render("/")
		dl.AddDraw(cellbuf.Rect(x, y, 1, 1), label, 0)
		x += 1

		query := m.searchQuery
		maxQueryLen := width - 15 // Leave room for status
		if len(query) > maxQueryLen && maxQueryLen > 3 {
			query = query[:maxQueryLen-3] + "..."
		}
		dl.AddDraw(cellbuf.Rect(x, y, len(query), 1), query, 0)
		x += len(query)

		// Show match count
		var status string
		if len(m.searchMatches) == 0 {
			status = " (no matches)"
		} else {
			status = fmt.Sprintf(" [%d/%d]", m.currentMatch+1, len(m.searchMatches))
		}
		dl.AddDraw(cellbuf.Rect(x, y, len(status), 1), statusStyle.Render(status), 0)
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
		hasSegments := len(line.segments) > 0

		switch line.lineType {
		case lineViewFileHeader:
			contentStr = m.applySearchHighlight(line.content, i, fileHeaderStyle)
		case lineViewHunkHeader:
			contentStr = m.applySearchHighlight(line.content, i, hunkHeaderStyle)
		case lineViewAdded:
			if hasSegments {
				contentStr = m.renderLineWithSegments(line, addedStyle, addedHighlightStyle)
			} else {
				contentStr = m.applySearchHighlight(line.content, i, addedStyle)
			}
		case lineViewRemoved:
			if hasSegments {
				contentStr = m.renderLineWithSegments(line, removedStyle, removedHighlightStyle)
			} else {
				contentStr = m.applySearchHighlight(line.content, i, removedStyle)
			}
		case lineViewContext:
			contentStr = m.applySearchHighlight(line.content, i, contextStyle)
		case lineViewEmpty:
			contentStr = line.content
		}

		// Apply word wrapping if enabled
		if m.wordWrap {
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

// getSearchMatchesForLine returns the search matches for a specific line index
func (m *Model) getSearchMatchesForLine(lineIdx int) []searchMatch {
	var matches []searchMatch
	for _, match := range m.searchMatches {
		if match.lineIdx == lineIdx {
			matches = append(matches, match)
		} else if match.lineIdx > lineIdx {
			break // searchMatches is sorted by lineIdx
		}
	}
	return matches
}

// isCurrentMatch returns true if the match at matchIdx is the current match
func (m *Model) isCurrentMatch(matchIdx int) bool {
	// Find the index of this match in searchMatches
	idx := 0
	for i, match := range m.searchMatches {
		if match.lineIdx < matchIdx {
			idx = i + 1
		} else {
			break
		}
	}
	return idx == m.currentMatch
}

// applySearchHighlight applies search highlighting to content
func (m *Model) applySearchHighlight(content string, lineIdx int, baseStyle lipgloss.Style) string {
	if !m.hasActiveSearch() {
		return baseStyle.Render(content)
	}

	matches := m.getSearchMatchesForLine(lineIdx)
	if len(matches) == 0 {
		return baseStyle.Render(content)
	}

	// Search highlight styles
	searchStyle := lipgloss.NewStyle().Background(lipgloss.Color("226")).Foreground(lipgloss.Color("0"))          // Yellow background
	currentSearchStyle := lipgloss.NewStyle().Background(lipgloss.Color("208")).Foreground(lipgloss.Color("0")).Bold(true) // Orange background, bold

	var result strings.Builder
	lastEnd := 0

	for _, match := range matches {
		// Render text before this match
		if match.startCol > lastEnd {
			result.WriteString(baseStyle.Render(content[lastEnd:match.startCol]))
		}

		// Determine if this is the current match
		isCurrentSearchMatch := false
		for i, sm := range m.searchMatches {
			if sm.lineIdx == match.lineIdx && sm.startCol == match.startCol {
				isCurrentSearchMatch = (i == m.currentMatch)
				break
			}
		}

		// Render the match with highlight
		matchText := content[match.startCol:match.endCol]
		if isCurrentSearchMatch {
			result.WriteString(currentSearchStyle.Render(matchText))
		} else {
			result.WriteString(searchStyle.Render(matchText))
		}

		lastEnd = match.endCol
	}

	// Render remaining text after last match
	if lastEnd < len(content) {
		result.WriteString(baseStyle.Render(content[lastEnd:]))
	}

	return result.String()
}

// New creates a new diff viewer model
func New(ctx *context.MainContext, revision string, focusFiles []string, focusFile string, rawContent string) *Model {
	// Determine mode: raw content mode if revision is empty (use rawContent for display)
	isRawMode := revision == ""

	// Initialize search input
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 256

	m := &Model{
		revision:      revision,
		focusFiles:    focusFiles,
		focusFile:     focusFile,
		rawContent:    rawContent,
		isRawMode:     isRawMode,
		context:       ctx,
		keymap:        config.Current.GetKeyMap(),
		showFileList:  true,
		fileListWidth: 30,
		searchInput:   ti,
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
