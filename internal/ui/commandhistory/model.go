package commandhistory

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/flash"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ common.StackedModel = (*Model)(nil)

const historyWindowSize = 10
const commandMarkWidth = 3

type selectHistoryItemMsg struct {
	index int
}

type Model struct {
	context       *context.MainContext
	source        flash.CommandHistorySource
	items         []flash.CommandHistoryEntry
	selectedIndex int
	windowStart   int
	successStyle  lipgloss.Style
	errorStyle    lipgloss.Style
	textStyle     lipgloss.Style
	matchedStyle  lipgloss.Style
}

func New(context *context.MainContext, source flash.CommandHistorySource) *Model {
	m := &Model{
		context:      context,
		source:       source,
		successStyle: common.DefaultPalette.Get("flash success"),
		errorStyle:   common.DefaultPalette.Get("flash error"),
		textStyle:    common.DefaultPalette.Get("flash text"),
		matchedStyle: common.DefaultPalette.Get("flash matched"),
	}
	if source != nil {
		m.items = source.CommandHistorySnapshot()
	}
	if len(m.items) > 0 {
		m.selectedIndex = len(m.items) - 1
		m.windowStart = max(0, len(m.items)-historyWindowSize)
	}
	m.clampViewport()
	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) StackedActionOwner() string {
	return actions.OwnerCommandHistory
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		switch intent := msg.(type) {
		case intents.CommandHistoryNavigate:
			if len(m.items) == 0 {
				return nil
			}
			// History renders oldest->newest from bottom to top, so move selection
			// opposite to delta to keep j moving visually down and k up.
			m.selectedIndex = min(len(m.items)-1, max(0, m.selectedIndex-intent.Delta))
			m.clampViewport()
			return nil
		case intents.CommandHistoryDeleteSelected:
			m.deleteSelected()
			return nil
		case intents.CommandHistoryClose:
			return common.Close
		}
	case selectHistoryItemMsg:
		if msg.index < 0 || msg.index >= len(m.items) {
			return nil
		}
		m.selectedIndex = msg.index
		m.clampViewport()
		return nil
	case common.CloseViewMsg:
		return common.Close
	}
	return nil
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	area := box.R
	y := area.Max.Y - 1
	maxWidth := area.Dx() - 4

	rest, _ := box.CutBottom(1)
	dl.AddDim(rest.R, render.ZOverlay)

	for _, item := range m.window() {
		content := m.renderEntry(item.entry, maxWidth, item.selected)
		w, h := lipgloss.Size(content)
		y -= h
		rect := cellbuf.Rect(area.Max.X-w, y, w, h)
		dl.AddDraw(rect, content, render.ZOverlay)
		dl.AddInteraction(rect, selectHistoryItemMsg{index: item.index}, render.InteractionClick, render.ZOverlay)
	}
}

type historyItem struct {
	entry    flash.CommandHistoryEntry
	index    int
	selected bool
}

func (m *Model) window() []historyItem {
	if len(m.items) == 0 {
		return nil
	}
	m.clampViewport()
	start := m.windowStart
	end := min(len(m.items), start+historyWindowSize)
	items := make([]historyItem, 0, end-start)
	for i := start; i < end; i++ {
		items = append(items, historyItem{
			entry:    m.items[i],
			index:    i,
			selected: i == m.selectedIndex,
		})
	}
	return items
}

func (m *Model) clampViewport() {
	if len(m.items) == 0 {
		m.selectedIndex = 0
		m.windowStart = 0
		return
	}
	m.selectedIndex = min(len(m.items)-1, max(0, m.selectedIndex))
	maxStart := max(0, len(m.items)-historyWindowSize)
	m.windowStart = min(maxStart, max(0, m.windowStart))
	if m.selectedIndex < m.windowStart {
		m.windowStart = m.selectedIndex
	}
	if m.selectedIndex >= m.windowStart+historyWindowSize {
		m.windowStart = m.selectedIndex - historyWindowSize + 1
	}
}

func (m *Model) deleteSelected() {
	m.clampViewport()
	if len(m.items) == 0 {
		return
	}

	selected := m.selectedIndex
	removed := m.items[selected]
	m.items = append(m.items[:selected], m.items[selected+1:]...)
	if m.source != nil {
		m.source.DeleteCommandHistoryByID(removed.ID)
	}

	if len(m.items) == 0 {
		m.selectedIndex = 0
		m.windowStart = 0
		return
	}

	m.selectedIndex = min(selected, len(m.items)-1)
	m.clampViewport()
}

func (m *Model) renderEntry(entry flash.CommandHistoryEntry, maxWidth int, selected bool) string {
	style := lipgloss.NewStyle()
	if selected {
		style = m.successStyle
		if entry.Err != nil {
			style = m.errorStyle
		}
	}

	parts := []string{m.renderCommandLine(entry.Command, entry.Err)}
	if selected {
		if entry.Err != nil {
			parts = append(parts, m.errorStyle.Render(entry.Err.Error()))
		} else if entry.Text != "" {
			parts = append(parts, style.Render(entry.Text))
		}
	}

	content := strings.Join(parts, "\n")
	if lipgloss.Width(content) > maxWidth {
		content = lipgloss.NewStyle().Width(maxWidth).Render(content)
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		PaddingLeft(1).
		PaddingRight(1).
		BorderForeground(style.GetForeground()).
		Render(content)
}

func (m *Model) renderCommandLine(command string, commandErr error) string {
	mark := m.successStyle.Width(commandMarkWidth).Render("✓ ")
	if commandErr != nil {
		mark = m.errorStyle.Width(commandMarkWidth).Render("✗ ")
	}
	return mark + flash.ColorizeCommand(command, m.textStyle, m.matchedStyle)
}
