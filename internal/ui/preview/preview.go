package preview

import (
	"bufio"
	"log"
	"strings"
	"sync/atomic"
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

var _ common.Model = (*Model)(nil)

type Model struct {
	*common.Sizeable
	tag                     atomic.Uint64
	previewVisible          bool
	previewAutoPosition     bool
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

const DebounceTime = 200 * time.Millisecond

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
	Tag uint64
}

type updatePreviewContentMsg struct {
	Tag     uint64
	Content string
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

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(previewMsg); ok {
		msg = k.msg
	}
	switch msg := msg.(type) {
	case common.SelectionChangedMsg, common.RefreshMsg:
		tag := m.tag.Add(1)
		return tea.Tick(DebounceTime, func(t time.Time) tea.Msg {
			if tag != m.tag.Load() {
				return nil
			}
			return refreshPreviewContentMsg{Tag: tag}
		})
	case refreshPreviewContentMsg:
		if m.tag.Load() == msg.Tag {
			tag := msg.Tag
			return func() tea.Msg {
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
				if tag != m.tag.Load() {
					return nil
				}
				return updatePreviewContentMsg{
					Tag:     tag,
					Content: string(output),
				}
			}
		}
	case updatePreviewContentMsg:
		if m.tag.Load() == msg.Tag {
			m.content = msg.Content
			m.contentLineCount = lipgloss.Height(m.content)
			m.reset()
		}
		return nil
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
	return nil
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

func New(context *context.MainContext) Model {
	borderStyle := common.DefaultPalette.GetBorder("preview border", lipgloss.NormalBorder())
	borderStyle = borderStyle.Inherit(common.DefaultPalette.Get("preview text"))

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

	return Model{
		Sizeable:                &common.Sizeable{Width: 0, Height: 0},
		viewRange:               &viewRange{start: 0, end: 0},
		context:                 context,
		keyMap:                  config.Current.GetKeyMap(),
		help:                    help.New(),
		borderStyle:             borderStyle,
		previewAutoPosition:     previewAutoPosition,
		previewAtBottom:         previewAtBottom,
		previewVisible:          config.Current.Preview.ShowAtStart,
		previewWindowPercentage: config.Current.Preview.WidthPercentage,
	}
}
