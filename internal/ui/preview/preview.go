package preview

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

const (
	scrollAmount = 3
	handleSize   = 3
)

var _ common.Model = (*Model)(nil)

var globalFileYOffsets = make(map[string]int)

type Model struct {
	*common.ViewNode
	*common.MouseAware
	*common.DragAware
	view                    viewport.Model
	previewVisible          bool
	previewAutoPosition     bool
	previewAtBottom         bool
	previewWindowPercentage float64
	content                 string
	contentLineCount        int
	contentWidth            int
	context                 *context.MainContext
	keyMap                  config.KeyMappings[key.Binding]
	searchQuery             string
	searchMatches           []int          // line numbers of matches
	currentSearchIndex      int            // current match index
	fileYOffsets            map[string]int // YOffset position per file key
}

const (
	debounceId       = "preview-refresh"
	debounceDuration = 50 * time.Millisecond
)

type previewMsg struct {
	msg tea.Msg
}

// Allow a message to be targetted to this component.
func PreviewCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return previewMsg{msg: msg}
	}
}

type updatePreviewContentMsg struct {
	Content string
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) SetFrame(frame cellbuf.Rectangle) {
	m.ViewNode.SetFrame(frame)
	if m.AtBottom() {
		m.view.Width = frame.Dx()
		m.view.Height = frame.Dy() - 1
	} else {
		m.view.Width = frame.Dx() - 1
		m.view.Height = frame.Dy()
	}
}

func (m *Model) Visible() bool {
	return m.previewVisible
}

func (m *Model) SetVisible(visible bool) {
	m.previewVisible = visible
	if m.previewVisible {
		m.reset()
	}
}

func (m *Model) ToggleVisible() {
	m.previewVisible = !m.previewVisible
	if m.previewVisible {
		m.reset()
	}
}

func (m *Model) SetPosition(autoPos bool, atBottom bool) {
	m.previewAutoPosition = autoPos
	m.previewAtBottom = atBottom
}

func (m *Model) AutoPosition() bool {
	return m.previewAutoPosition
}

func (m *Model) AtBottom() bool {
	return m.previewAtBottom
}

func (m *Model) WindowPercentage() float64 {
	return m.previewWindowPercentage
}

func (m *Model) Scroll(delta int) tea.Cmd {
	if delta > 0 {
		m.view.ScrollDown(delta)
	} else if delta < 0 {
		m.view.ScrollUp(-delta)
	}
	m.saveCurrentYOffset()
	return nil
}

func (m *Model) ScrollHorizontal(delta int) tea.Cmd {
	if delta > 0 {
		m.view.ScrollRight(delta)
	} else if delta < 0 {
		m.view.ScrollLeft(-delta)
	}
	return nil
}

func (m *Model) DragStart(x, y int) bool {
	if !m.previewVisible {
		return false
	}

	if m.Parent.Width == 0 || m.Parent.Height == 0 {
		return false
	}

	if m.AtBottom() {
		if y-m.Frame.Min.Y > handleSize {
			return false
		}
	} else {
		if x-m.Frame.Min.X > handleSize {
			return false
		}
	}

	m.BeginDrag(m.Frame.Min.X, y)
	return true
}

func (m *Model) DragMove(x, y int) tea.Cmd {
	if !m.IsDragging() {
		return nil
	}

	var percentage float64
	if m.AtBottom() {
		percentage = float64((m.Parent.Height-y)*100) / float64(m.Parent.Height)
	} else {
		percentage = float64((m.Parent.Width-x)*100) / float64(m.Parent.Width)
	}

	m.SetWindowPercentage(percentage)
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(previewMsg); ok {
		msg = k.msg
	}
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.Scroll(-scrollAmount)
		case tea.MouseButtonWheelDown:
			m.Scroll(scrollAmount)
		case tea.MouseButtonWheelLeft:
			m.ScrollHorizontal(-scrollAmount)
		case tea.MouseButtonWheelRight:
			m.ScrollHorizontal(scrollAmount)
		}
	case common.SelectionChangedMsg, common.RefreshMsg:
		return m.refreshPreview()
	case updatePreviewContentMsg:
		m.SetContent(msg.Content)
		return nil
	case common.PreviewSearchMsg:
		m.SetSearchQuery(string(msg))
		return nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Preview.ScrollDown):
			m.Scroll(1)
		case key.Matches(msg, m.keyMap.Preview.ScrollUp):
			m.Scroll(-1)
		case key.Matches(msg, m.keyMap.Preview.HalfPageDown):
			m.view.HalfPageDown()
			m.saveCurrentYOffset()
		case key.Matches(msg, m.keyMap.Preview.HalfPageUp):
			m.view.HalfPageUp()
			m.saveCurrentYOffset()
		case key.Matches(msg, m.keyMap.Details.NextMatch):
			m.NextSearchMatch()
		case key.Matches(msg, m.keyMap.Details.PreviousMatch):
			m.PreviousSearchMatch()
		case key.Matches(msg, m.keyMap.Details.GoToStart):
			m.GoToBeginning()
		case key.Matches(msg, m.keyMap.Details.GoToEnd):
			m.GoToEnd()
		}
	}
	return nil
}

func (m *Model) SetContent(content string) {
	m.content = strings.ReplaceAll(content, "\r", "")
	m.updateViewportContent()
	m.restoreYOffset()
}

func (m *Model) updateViewportContent() {
	if m.searchQuery == "" {
		m.view.SetContent(m.content)
		return
	}

	// Highlight matches with underline-only ANSI codes
	re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(m.searchQuery))
	highlighted := re.ReplaceAllStringFunc(m.content, func(match string) string {
		return "\x1b[4m" + match + "\x1b[24m"
	})
	m.view.SetContent(highlighted)
}

func (m *Model) View() string {
	border := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), m.AtBottom(), false, false, !m.AtBottom())
	return border.Render(m.view.View())
}

func (m *Model) reset() {
	m.saveCurrentYOffset()
	m.view.SetYOffset(0)
	m.view.SetXOffset(0)
}

// saveCurrentYOffset saves the current YOffset for the current file
func (m *Model) saveCurrentYOffset() {
	fileKey := m.getFileKey()
	if fileKey != "" {
		m.fileYOffsets[fileKey] = m.view.YOffset
	}
}

// restoreYOffset restores the saved YOffset for the current file
func (m *Model) restoreYOffset() {
	fileKey := m.getFileKey()
	if fileKey == "" {
		return
	}

	offset, found := m.fileYOffsets[fileKey]
	if found {
		m.view.SetYOffset(offset)
	} else {
		m.view.SetYOffset(0)
	}
}

// getFileKey generates a unique key for the current file
func (m *Model) getFileKey() string {
	switch item := m.context.SelectedItem.(type) {
	case context.SelectedFile:
		return item.File
	default:
		return ""
	}
}

func (m *Model) refreshPreview() tea.Cmd {
	return common.Debounce(debounceId, debounceDuration, func() tea.Msg {
		var args []string
		switch msg := m.context.SelectedItem.(type) {
		case context.SelectedFile:
			args = jj.TemplatedArgs(config.Current.Preview.FileCommand, map[string]string{
				jj.RevsetPlaceholder:   m.context.CurrentRevset,
				jj.ChangeIdPlaceholder: msg.ChangeId,
				jj.CommitIdPlaceholder: msg.CommitId,
				jj.FilePlaceholder:     msg.File,
			})
		case context.SelectedRevision:
			args = jj.TemplatedArgs(config.Current.Preview.RevisionCommand, map[string]string{
				jj.RevsetPlaceholder:   m.context.CurrentRevset,
				jj.ChangeIdPlaceholder: msg.ChangeId,
				jj.CommitIdPlaceholder: msg.CommitId,
			})
		case context.SelectedOperation:
			args = jj.TemplatedArgs(config.Current.Preview.OplogCommand, map[string]string{
				jj.RevsetPlaceholder:      m.context.CurrentRevset,
				jj.OperationIdPlaceholder: msg.OperationId,
			})
		}

		output, _ := m.context.RunCommandImmediate(args)
		return updatePreviewContentMsg{
			Content: string(output),
		}
	})
}

func (m *Model) SetWindowPercentage(percentage float64) {
	m.previewWindowPercentage = percentage
	if m.previewWindowPercentage < 10 {
		m.previewWindowPercentage = 10
	} else if m.previewWindowPercentage > 95 {
		m.previewWindowPercentage = 95
	}
}

func (m *Model) Expand() {
	m.SetWindowPercentage(m.previewWindowPercentage + config.Current.Preview.WidthIncrementPercentage)
}

func (m *Model) Shrink() {
	m.SetWindowPercentage(m.previewWindowPercentage - config.Current.Preview.WidthIncrementPercentage)
}

func (m *Model) StartSearch() {
	m.searchQuery = ""
	m.searchMatches = nil
	m.currentSearchIndex = -1
}

func (m *Model) SetSearchQuery(query string) {
	m.searchQuery = query
	m.searchMatches = nil
	m.currentSearchIndex = -1
	if query != "" {
		m.findMatches()
		if len(m.searchMatches) > 0 {
			m.currentSearchIndex = 0
			m.scrollToMatch(0)
		}
	}
	m.updateViewportContent()
}

func (m *Model) NextSearchMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.currentSearchIndex = (m.currentSearchIndex + 1) % len(m.searchMatches)
	m.scrollToMatch(m.currentSearchIndex)
}

func (m *Model) PreviousSearchMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.currentSearchIndex--
	if m.currentSearchIndex < 0 {
		m.currentSearchIndex = len(m.searchMatches) - 1
	}
	m.scrollToMatch(m.currentSearchIndex)
}

func (m *Model) GoToBeginning() {
	m.view.GotoTop()
	m.saveCurrentYOffset()
}

func (m *Model) GoToEnd() {
	m.view.GotoBottom()
	m.saveCurrentYOffset()
}

func (m *Model) findMatches() {
	m.searchMatches = nil
	if m.searchQuery == "" {
		return
	}
	lines := strings.Split(m.content, "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(m.searchQuery)) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}
}

func (m *Model) scrollToMatch(index int) {
	if index < 0 || index >= len(m.searchMatches) {
		return
	}
	lineNum := m.searchMatches[index]
	// Scroll to make the match visible, with some context
	targetY := lineNum - 2 // show 2 lines before
	if targetY < 0 {
		targetY = 0
	}
	m.view.SetYOffset(targetY)
	m.saveCurrentYOffset()
}

func New(context *context.MainContext) *Model {
	previewAutoPosition := false
	previewAtBottom := false
	previewPositionCfg, err := config.GetPreviewPosition(config.Current)
	if err != nil {
		log.Fatal(err)
	}

	if previewPositionCfg == config.PreviewPositionAuto {
		previewAutoPosition = true
	} else if previewPositionCfg == config.PreviewPositionBottom {
		previewAtBottom = true
	}

	return &Model{
		ViewNode:                &common.ViewNode{Width: 0, Height: 0},
		MouseAware:              common.NewMouseAware(),
		DragAware:               common.NewDragAware(),
		context:                 context,
		keyMap:                  config.Current.GetKeyMap(),
		previewAutoPosition:     previewAutoPosition,
		previewAtBottom:         previewAtBottom,
		previewVisible:          config.Current.Preview.ShowAtStart,
		previewWindowPercentage: config.Current.Preview.WidthPercentage,
		fileYOffsets:            globalFileYOffsets,
	}
}
