package preview

import (
	"bufio"
	"log"
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

type previewState struct {
	activeList context.ListId
	pos        int
}

func (p *previewState) Equals(other *previewState) bool {
	if p == nil || other == nil {
		return false
	}
	if p == other {
		return true
	}
	return p.activeList == other.activeList && p.pos == other.pos
}

type Model struct {
	*common.Sizeable
	tag              int
	viewRange        *viewRange
	help             help.Model
	content          string
	contentLineCount int
	context          *context.MainContext
	keyMap           config.KeyMappings[key.Binding]
	borderStyle      lipgloss.Style
	state            *previewState
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

func (m *Model) updatePreviewContent(content string) {
	m.content = content
	m.contentLineCount = lipgloss.Height(m.content)
	m.reset()
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	if k, ok := msg.(previewMsg); ok {
		msg = k.msg
	}
	switch msg := msg.(type) {
	case common.RefreshMsg:
		if !m.context.Preview.Visible {
			return m, nil
		}

		m.tag++
		tag := m.tag
		m.state = capture(m.context)
		log.Println("preview: scheduling refresh", m.state)
		return m, tea.Tick(DebounceTime, func(t time.Time) tea.Msg {
			return refreshPreviewContentMsg{Tag: tag}
		})
	case refreshPreviewContentMsg:
		if m.tag == msg.Tag {
			replacements := m.context.CreateReplacements()
			switch m.state.activeList {
			case context.ListRevisions:
				output, _ := m.context.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.RevisionCommand, replacements))
				m.updatePreviewContent(string(output))
			case context.ListFiles:
				output, _ := m.context.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.FileCommand, replacements))
				m.updatePreviewContent(string(output))
			case context.ListOplog:
				output, _ := m.context.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.OplogCommand, replacements))
				m.updatePreviewContent(string(output))
			}
			m.state = capture(m.context)
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

func (m *Model) IsDirty() bool {
	if !m.context.Preview.Visible {
		return false
	}
	newState := capture(m.context)
	result := !newState.Equals(m.state)
	return result
}

func capture(ctx *context.MainContext) *previewState {
	var pos int
	switch ctx.ActiveList {
	case context.ListRevisions:
		pos = ctx.Revisions.Revisions.Cursor
	case context.ListFiles:
		pos = ctx.Revisions.Files.Cursor
	case context.ListOplog:
		pos = ctx.OpLog.Cursor
	case context.ListEvolog:
		pos = ctx.Evolog.Cursor
	}
	return &previewState{
		activeList: ctx.ActiveList,
		pos:        pos,
	}
}

func New(ctx *context.MainContext) Model {
	borderStyle := common.DefaultPalette.GetBorder("preview border", lipgloss.NormalBorder())
	borderStyle = borderStyle.Inherit(common.DefaultPalette.Get("preview text"))

	return Model{
		Sizeable:    &common.Sizeable{Width: 0, Height: 0},
		state:       nil,
		viewRange:   &viewRange{start: 0, end: 0},
		context:     ctx,
		keyMap:      config.Current.GetKeyMap(),
		help:        help.New(),
		borderStyle: borderStyle,
	}
}
