package preview

import (
	"bufio"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type viewRange struct {
	start int
	end   int
}

type Model struct {
	*common.Sizeable
	tag                     int
	previewVisible          bool
	previewAtBottom         bool
	previewWindowPercentage float64
	viewRange               *viewRange
	help                    help.Model
	content                 string
	contentLineCount        int
	context                 *context.MainContext
	keyMap                  config.KeyMappings[key.Binding]
	borderStyle             lipgloss.Style
}

const DebounceTime = 50 * time.Millisecond

type previewMsg struct {
	msg tea.Msg
}

// Allow a message to be targetted to this component.
func PreviewCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return previewMsg{msg: msg}
	}
}

type refreshPreviewContentMsg struct {
	Tag int
}

func (m *Model) SetHeight(h int) {
	m.viewRange.end = min(m.viewRange.start+h-3, m.contentLineCount)
	m.Height = h
}

func (m *Model) Init() tea.Cmd {
	return nil
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

func (m *Model) TogglePosition() {
	m.previewAtBottom = !m.previewAtBottom
}

func (m *Model) AtBottom() bool {
	return m.previewAtBottom
}

func (m *Model) WindowPercentage() float64 {
	return m.previewWindowPercentage
}

func (m *Model) updatePreviewContent(content string) {
	m.content = content
	m.contentLineCount = lipgloss.Height(m.content)
	m.reset()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(previewMsg); ok {
		msg = k.msg
	}
	switch msg := msg.(type) {
	case common.SelectionChangedMsg, common.RefreshMsg:
		m.tag++
		tag := m.tag
		return m, tea.Tick(DebounceTime, func(t time.Time) tea.Msg {
			return refreshPreviewContentMsg{Tag: tag}
		})
	case refreshPreviewContentMsg:
		if m.tag == msg.Tag {
			//replacements := m.context.ScopeValues
			//switch m.context.CurrentScope() {
			//case common.ScopeRevisions:
			//	if _, ok := replacements[jj.FilePlaceholder]; ok {
			//		output, _ := m.context.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.FileCommand, replacements))
			//		m.updatePreviewContent(string(output))
			//	} else {
			//		output, _ := m.context.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.RevisionCommand, replacements))
			//		m.updatePreviewContent(string(output))
			//	}
			//case common.ScopeOplog:
			//	output, _ := m.context.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.OplogCommand, replacements))
			//	m.updatePreviewContent(string(output))
			//}
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Preview.ScrollDown):
			if m.viewRange.end < m.contentLineCount {
				m.viewRange.start++
				m.viewRange.end++
			}
		case key.Matches(msg, m.keyMap.Preview.ScrollUp):
			if m.viewRange.start > 0 {
				m.viewRange.start--
				m.viewRange.end--
			}
		case key.Matches(msg, m.keyMap.Preview.HalfPageDown):
			contentHeight := m.contentLineCount
			halfPageSize := m.Height / 2
			if halfPageSize+m.viewRange.end > contentHeight {
				halfPageSize = contentHeight - m.viewRange.end
			}

			m.viewRange.start += halfPageSize
			m.viewRange.end += halfPageSize
		case key.Matches(msg, m.keyMap.Preview.HalfPageUp):
			halfPageSize := min(m.Height/2, m.viewRange.start)
			m.viewRange.start -= halfPageSize
			m.viewRange.end -= halfPageSize
		}
	}
	return m, nil
}

func (m *Model) View() string {
	var w strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(m.content))
	current := 0
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.ReplaceAll(line, "\r", "")
		if current >= m.viewRange.start && current <= m.viewRange.end {
			if current > m.viewRange.start {
				w.WriteString("\n")
			}
			w.WriteString(lipgloss.NewStyle().MaxWidth(m.Width - 2).Render(line))
		}
		current++
		if current > m.viewRange.end {
			break
		}
	}
	view := lipgloss.Place(m.Width-2, m.Height-2, 0, 0, w.String())
	return m.borderStyle.Render(view)
}

func (m *Model) reset() {
	m.viewRange.start, m.viewRange.end = 0, m.Height
}

func (m *Model) Expand() {
	m.previewWindowPercentage += config.Current.Preview.WidthIncrementPercentage
	if m.previewWindowPercentage > 95 {
		m.previewWindowPercentage = 95
	}
}

func (m *Model) Shrink() {
	m.previewWindowPercentage -= config.Current.Preview.WidthIncrementPercentage
	if m.previewWindowPercentage < 10 {
		m.previewWindowPercentage = 10
	}
}

func New(context *context.MainContext) tea.Model {
	borderStyle := common.DefaultPalette.GetBorder("preview border", lipgloss.NormalBorder())
	borderStyle = borderStyle.Inherit(common.DefaultPalette.Get("preview text"))

	return &Model{
		Sizeable:                &common.Sizeable{Width: 0, Height: 0},
		viewRange:               &viewRange{start: 0, end: 0},
		context:                 context,
		keyMap:                  config.Current.GetKeyMap(),
		help:                    help.New(),
		borderStyle:             borderStyle,
		previewAtBottom:         config.Current.Preview.ShowAtBottom,
		previewVisible:          config.Current.Preview.ShowAtStart,
		previewWindowPercentage: config.Current.Preview.WidthPercentage,
	}
}
