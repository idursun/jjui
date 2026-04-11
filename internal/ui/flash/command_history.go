package flash

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ common.StackedModel = (*CommandHistoryModel)(nil)

type selectHistoryItemMsg struct {
	index int
}

type CommandHistoryModel struct {
	source        *Model
	items         []commandHistoryEntry
	selectedIndex int
	renderer      CardRenderer
}

func newCommandHistory(source *Model) *CommandHistoryModel {
	m := &CommandHistoryModel{
		source:   source,
		renderer: NewCardRenderer(),
	}
	if source != nil {
		m.items = source.commandHistorySnapshot()
	}
	if len(m.items) > 0 {
		m.selectedIndex = len(m.items) - 1
	}
	m.clampSelection()
	return m
}

func (m *CommandHistoryModel) Init() tea.Cmd {
	return nil
}

func (m *CommandHistoryModel) Scopes() []dispatch.Scope {
	return []dispatch.Scope{
		{
			Name:    actions.ScopeCommandHistory,
			Leak:    dispatch.LeakGlobal,
			Handler: m,
		},
	}
}

func (m *CommandHistoryModel) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.CommandHistoryNavigate:
		if len(m.items) == 0 {
			return nil, true
		}
		// History renders oldest->newest from bottom to top, so move selection
		// opposite to delta to keep j moving visually down and k up.
		m.selectedIndex = min(len(m.items)-1, max(0, m.selectedIndex-intent.Delta))
		m.clampSelection()
		return nil, true
	case intents.CommandHistoryDeleteSelected:
		m.deleteSelected()
		return nil, true
	case intents.CommandHistoryClose:
		return common.Close, true
	}
	return nil, false
}

func (m *CommandHistoryModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
	case selectHistoryItemMsg:
		if msg.index < 0 || msg.index >= len(m.items) {
			return nil
		}
		m.selectedIndex = msg.index
		m.clampSelection()
		return nil
	case common.CloseViewMsg:
		return common.Close
	}
	return nil
}

func (m *CommandHistoryModel) ViewRect(dl *render.DisplayContext, box layout.Box) {
	area := box.R
	y := area.Max.Y - 1
	maxWidth := area.Dx() - 4

	rest, _ := box.CutBottom(1)
	dl.AddDim(rest.R, render.ZOverlay)

	for _, item := range m.visibleWindow(maxWidth, area.Dy()) {
		content := m.renderer.RenderHistoryEntry(item.entry, maxWidth, item.selected)
		w, h := lipgloss.Size(content)
		y -= h
		rect := layout.Rect(area.Max.X-w, y, w, h)
		dl.AddDraw(rect, content, render.ZOverlay)
		dl.AddInteraction(rect, selectHistoryItemMsg{index: item.index}, render.InteractionClick, render.ZOverlay)
	}
}

type historyItem struct {
	entry    commandHistoryEntry
	index    int
	selected bool
}

func (m *CommandHistoryModel) visibleWindow(maxWidth, maxHeight int) []historyItem {
	if len(m.items) == 0 || maxHeight <= 0 {
		return nil
	}
	m.clampSelection()

	heights := make([]int, len(m.items))
	for i, entry := range m.items {
		_, heights[i] = lipgloss.Size(m.renderer.RenderHistoryEntry(entry, maxWidth, i == m.selectedIndex))
	}

	start := m.selectedIndex
	end := m.selectedIndex + 1
	used := heights[m.selectedIndex]
	if used >= maxHeight {
		return []historyItem{{
			entry:    m.items[m.selectedIndex],
			index:    m.selectedIndex,
			selected: true,
		}}
	}

	for i := m.selectedIndex - 1; i >= 0; i-- {
		if used+heights[i] > maxHeight {
			break
		}
		start = i
		used += heights[i]
	}

	for i := m.selectedIndex + 1; i < len(m.items); i++ {
		if used+heights[i] > maxHeight {
			break
		}
		end = i + 1
		used += heights[i]
	}

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

func (m *CommandHistoryModel) clampSelection() {
	if len(m.items) == 0 {
		m.selectedIndex = 0
		return
	}
	m.selectedIndex = min(len(m.items)-1, max(0, m.selectedIndex))
}

func (m *CommandHistoryModel) deleteSelected() {
	m.clampSelection()
	if len(m.items) == 0 {
		return
	}

	selected := m.selectedIndex
	removed := m.items[selected]
	m.items = append(m.items[:selected], m.items[selected+1:]...)
	if m.source != nil {
		m.source.deleteCommandHistoryByID(removed.ID)
	}

	if len(m.items) == 0 {
		m.selectedIndex = 0
		return
	}

	m.selectedIndex = min(selected, len(m.items)-1)
	m.clampSelection()
}
