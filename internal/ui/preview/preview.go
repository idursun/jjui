package preview

import (
	"bufio"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type viewRange struct {
	start int
	end   int
}

type Model struct {
	*common.Sizeable
	tag                     atomic.Uint64
	previewVisible          bool
	previewAtBottom         bool
	previewWindowPercentage float64
	viewRange               *viewRange
	help                    help.Model
	content                 string
	contentLineCount        int
	context                 *context.MainContext
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
	item      string
	Tag       uint64
	variables map[string]string
	args      []string
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
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		case "preview.update":
			log.Printf("preview update action received tag: %d", m.tag.Load())
			currentTag := m.tag.Add(1)
			variables := m.context.GetVariables()
			args := msg.Action.GetArgs("jj")
			if len(args) == 0 {
				log.Println("no jj args for preview update action")
				return m, nil
			}
			args = jj.TemplatedArgs(args, variables)
			return m, tea.Tick(DebounceTime, func(t time.Time) tea.Msg {
				if currentTag == m.tag.Load() {
					log.Printf("Scheduling preview refresh for (tag %d)", currentTag)
					return refreshPreviewContentMsg{Tag: currentTag, args: args}
				}
				log.Printf("Not scheduling preview refresh for (tag %d)", currentTag)
				return nil
			})
		}
	case refreshPreviewContentMsg:
		if m.tag.Load() != msg.Tag {
			log.Printf("ignoring preview tag changed from %d to %d", msg.Tag, m.tag.Load())
			return m, nil
		}

		log.Printf("Refreshing preview for %d", msg.Tag)
		if len(msg.args) > 0 {
			output, _ := m.context.RunCommandImmediate(msg.args)
			m.updatePreviewContent(string(output))
			return m, nil
		}
	case tea.KeyMsg:
		switch {
		//case key.Matches(msg, m.keyMap.Preview.ScrollDown):
		//	if m.viewRange.end < m.contentLineCount {
		//		m.viewRange.start++
		//		m.viewRange.end++
		//	}
		//case key.Matches(msg, m.keyMap.Preview.ScrollUp):
		//	if m.viewRange.start > 0 {
		//		m.viewRange.start--
		//		m.viewRange.end--
		//	}
		//case key.Matches(msg, m.keyMap.Preview.HalfPageDown):
		//	contentHeight := m.contentLineCount
		//	halfPageSize := m.Height / 2
		//	if halfPageSize+m.viewRange.end > contentHeight {
		//		halfPageSize = contentHeight - m.viewRange.end
		//	}
		//
		//	m.viewRange.start += halfPageSize
		//	m.viewRange.end += halfPageSize
		//case key.Matches(msg, m.keyMap.Preview.HalfPageUp):
		//	halfPageSize := min(m.Height/2, m.viewRange.start)
		//	m.viewRange.start -= halfPageSize
		//	m.viewRange.end -= halfPageSize
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
		help:                    help.New(),
		borderStyle:             borderStyle,
		previewAtBottom:         config.Current.Preview.ShowAtBottom,
		previewVisible:          config.Current.Preview.ShowAtStart,
		previewWindowPercentage: config.Current.Preview.WidthPercentage,
	}
}
