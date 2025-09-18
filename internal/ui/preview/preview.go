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
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
)

const DebounceTime = 200 * time.Millisecond

type viewRange struct {
	start int
	end   int
}

var _ tea.Model = (*Model)(nil)
var _ view.IViewModel = (*Model)(nil)

type previewMsg struct {
	msg tea.Msg
}

type refreshPreviewContentMsg struct {
	Tag uint64
}

type Model struct {
	context.CommandRunner
	*view.ViewNode
	context          *context.MainContext
	previewContext   *context.PreviewContext
	tag              atomic.Uint64
	viewRange        *viewRange
	help             help.Model
	content          string
	contentLineCount int
	keyMap           config.KeyMappings[key.Binding]
	borderStyle      lipgloss.Style
	previewed        models.IItem
}

func (m *Model) GetId() view.ViewId {
	return view.PreviewViewId
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	v.Id = m.GetId()
	v.NeedsRefresh = true
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keyMap.Up,
		m.keyMap.Down,
		m.keyMap.Preview.Expand,
		m.keyMap.Preview.Shrink,
		m.keyMap.Preview.HalfPageUp,
		m.keyMap.Preview.HalfPageDown,
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}
func (m *Model) Init() tea.Cmd {
	return tea.Tick(DebounceTime, func(t time.Time) tea.Msg {
		return refreshPreviewContentMsg{Tag: m.tag.Load()}
	})
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
	if !m.Visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case common.RefreshMsg:
		log.Printf("preview will be read at %d", m.tag.Load())
		if current := m.getCurrentItem(); current != nil && (m.previewed == nil || !current.Equals(m.previewed)) {
			tag := m.tag.Add(1)
			log.Printf("previewed updated by %d", tag)
			m.previewed = current
			log.Println("preview: scheduling refresh", tag)
			return m, tea.Tick(DebounceTime, func(t time.Time) tea.Msg {
				if tag == m.tag.Load() {
					return refreshPreviewContentMsg{Tag: tag}
				}
				return nil
			})
		}
	case refreshPreviewContentMsg:
		if m.tag.Load() != msg.Tag {
			return m, nil
		}

		log.Println("preview: updating ", msg.Tag)
		replacements := m.context.CreateReplacements()
		m.previewed = m.getCurrentItem()
		focusedView := m.ViewManager.GetFocusedView()
		if focusedView == nil {
			return m, nil
		}

		switch item := m.previewed.(type) {
		case *models.RevisionItem:
			replacements[jj.ChangeIdPlaceholder] = item.Commit.CommitId
			replacements[jj.CommitIdPlaceholder] = item.Commit.CommitId
			output, _ := m.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.RevisionCommand, replacements))
			m.updatePreviewContent(string(output))
		case *models.RevisionFileItem:
			output, _ := m.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.FileCommand, replacements))
			m.updatePreviewContent(string(output))
		case *models.OperationLogItem:
			output, _ := m.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.OplogCommand, replacements))
			m.updatePreviewContent(string(output))
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Cancel) && m.ViewManager.IsFocused(m.Id):
			m.ViewManager.StopEditing()
			m.ViewManager.RestorePreviousFocus()
			return m, nil
		case key.Matches(msg, m.keyMap.Preview.ScrollDown):
			if m.viewRange.end < m.contentLineCount {
				m.viewRange.start++
				m.viewRange.end++
			}
		case key.Matches(msg, m.keyMap.Down) && m.ViewManager.IsFocused(m.Id):
			if m.viewRange.end < m.contentLineCount {
				m.viewRange.start++
				m.viewRange.end++
			}
		case key.Matches(msg, m.keyMap.Preview.ScrollUp):
			if m.viewRange.start > 0 {
				m.viewRange.start--
				m.viewRange.end--
			}
		case key.Matches(msg, m.keyMap.Up) && m.ViewManager.IsFocused(m.Id):
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
		case key.Matches(msg, m.keyMap.Preview.Expand):
			m.previewContext.Expand()
			m.ViewManager.UpdateViewConstraint(m.Id, func(constraint *view.LayoutConstraint) {
				constraint.Percentage = int(m.previewContext.WindowPercentage)
			})
			m.ViewManager.Layout()
		case key.Matches(msg, m.keyMap.Preview.Shrink):
			m.previewContext.Shrink()
			m.ViewManager.UpdateViewConstraint(m.Id, func(constraint *view.LayoutConstraint) {
				constraint.Percentage = int(m.previewContext.WindowPercentage)
			})
			m.ViewManager.Layout()
		case key.Matches(msg, m.keyMap.Preview.ToggleBottom):
			if container := m.ViewManager.GetViewContainer(m.Id); container != nil {
				if container.Direction == view.Horizontal {
					container.Direction = view.Vertical
				} else {
					container.Direction = view.Horizontal
				}
			}
			m.ViewManager.Layout()
		}
	}
	return m, nil
}

func (m *Model) View() string {
	m.viewRange.end = min(m.viewRange.start+m.Height-2, m.contentLineCount)
	var w strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(m.content))
	current := 0
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.ReplaceAll(line, "\r", "")
		if current >= m.viewRange.start && current < m.viewRange.end {
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
	content := w.String()
	v := lipgloss.Place(m.Width-2, m.Height-2, 0, 0, content)
	borderStyle := m.borderStyle
	if m.ViewManager.IsFocused(m.Id) {
		borderStyle = lipgloss.NewStyle().Border(lipgloss.DoubleBorder())
	}
	return borderStyle.Render(v)
}

func (m *Model) reset() {
	m.viewRange.start, m.viewRange.end = 0, m.Height
}

func (m *Model) getCurrentItem() models.IItem {
	focusedViews := m.ViewManager.GetFocusedViews()
	if len(focusedViews) == 0 {
		return nil
	}
	var ret models.IItem
	for _, v := range focusedViews {
		if listProvider, ok := v.Model.(list.IListProvider); ok {
			if currentItem := listProvider.CurrentItem(); currentItem != nil {
				ret = currentItem
			}
		}
	}
	return ret
}

func New(ctx *context.MainContext, previewContext *context.PreviewContext) view.IViewModel {
	borderStyle := common.DefaultPalette.GetBorder("preview border", lipgloss.NormalBorder())
	borderStyle = borderStyle.Inherit(common.DefaultPalette.Get("preview text"))

	m := Model{
		CommandRunner:  ctx.CommandRunner,
		context:        ctx,
		previewContext: previewContext,
		viewRange:      &viewRange{start: 0, end: 0},
		keyMap:         config.Current.GetKeyMap(),
		help:           help.New(),
		borderStyle:    borderStyle,
	}
	return &m
}
