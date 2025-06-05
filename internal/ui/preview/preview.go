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
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type viewRange struct {
	start int
	end   int
}

type Model struct {
	tag              int
	viewRange        *viewRange
	help             help.Model
	width            int
	height           int
	content          string
	contentLineCount int
	context          context.AppContext
	keyMap           config.KeyMappings[key.Binding]
}

const DebounceTime = 50 * time.Millisecond

var border = lipgloss.NewStyle().Border(lipgloss.NormalBorder())

type refreshPreviewContentMsg struct {
	Tag int
}

type updatePreviewContentMsg struct {
	Content string
}

func (m *Model) Width() int {
	return m.width
}

func (m *Model) Height() int {
	return m.height
}

func (m *Model) SetWidth(w int) {
	m.width = w
}

func (m *Model) SetHeight(h int) {
	m.viewRange.end = min(m.viewRange.start+h-3, m.contentLineCount)
	m.height = h
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updatePreviewContentMsg:
		m.content = msg.Content
		m.contentLineCount = strings.Count(m.content, "\n")
		m.reset()
	case common.SelectionChangedMsg, common.RefreshMsg:
		m.tag++
		tag := m.tag
		return m, tea.Tick(DebounceTime, func(t time.Time) tea.Msg {
			return refreshPreviewContentMsg{Tag: tag}
		})
	case refreshPreviewContentMsg:
		if m.tag == msg.Tag {
			switch msg := m.context.SelectedItem().(type) {
			case context.SelectedFile:
				return m, func() tea.Msg {
					output, _ := m.context.RunCommandImmediate(jj.Diff(msg.ChangeId, msg.File, config.Current.Preview.ExtraArgs...))
					return updatePreviewContentMsg{Content: string(output)}
				}
			case context.SelectedRevision:
				return m, func() tea.Msg {
					output, _ := m.context.RunCommandImmediate(jj.Show(msg.ChangeId, config.Current.Preview.ExtraArgs...))
					return updatePreviewContentMsg{Content: string(output)}
				}
			case context.SelectedOperation:
				return m, func() tea.Msg {
					output, _ := m.context.RunCommandImmediate(jj.OpShow(msg.OperationId))
					return updatePreviewContentMsg{Content: string(output)}
				}
			}
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
			halfPageSize := m.height / 2
			if halfPageSize+m.viewRange.end > contentHeight {
				halfPageSize = contentHeight - m.viewRange.end
			}

			m.viewRange.start += halfPageSize
			m.viewRange.end += halfPageSize
		case key.Matches(msg, m.keyMap.Preview.HalfPageUp):
			halfPageSize := min(m.height/2, m.viewRange.start)
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
		if current >= m.viewRange.start && current <= m.viewRange.end {
			if current > m.viewRange.start {
				w.WriteString("\n")
			}
			w.WriteString(lipgloss.NewStyle().MaxWidth(m.width - 2).Render(line))
		}
		current++
		if current > m.viewRange.end {
			break
		}
	}
	view := lipgloss.Place(m.width-2, m.height-2, 0, 0, w.String())
	return border.Render(view)
}

func (m *Model) reset() {
	m.viewRange.start, m.viewRange.end = 0, m.height
}

func New(context context.AppContext) Model {
	keyMap := context.KeyMap()
	return Model{
		viewRange: &viewRange{start: 0, end: 0},
		context:   context,
		keyMap:    keyMap,
		help:      help.New(),
	}
}
