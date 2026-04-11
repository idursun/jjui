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
	y := area.Max.Y
	maxWidth := area.Dx() - 4

	rest, _ := box.CutBottom(1)
	dl.AddDim(rest.R, render.ZOverlay)

	for _, item := range m.renderedItems(maxWidth, area.Dy()) {
		y -= item.h
		rect := layout.Rect(area.Max.X-item.w, y, item.w, item.h)
		dl.AddDraw(rect, item.content, render.ZOverlay)
		dl.AddInteraction(rect, selectHistoryItemMsg{index: item.index}, render.InteractionClick, render.ZOverlay)
	}
}

type renderedHistoryItem struct {
	index   int
	content string
	w       int
	h       int
}

func (m *CommandHistoryModel) renderedItems(maxWidth, maxHeight int) []renderedHistoryItem {
	if len(m.items) == 0 || maxHeight <= 0 {
		return nil
	}
	m.clampSelection()

	selected := m.renderedItem(m.selectedIndex, maxWidth)
	if selected.h >= maxHeight {
		return []renderedHistoryItem{selected}
	}

	used := selected.h
	before := make([]renderedHistoryItem, 0, m.selectedIndex)
	for i := m.selectedIndex - 1; i >= 0; i-- {
		item := m.renderedItem(i, maxWidth)
		if used+item.h > maxHeight {
			break
		}
		before = append(before, item)
		used += item.h
	}

	items := make([]renderedHistoryItem, 0, len(before)+1)
	for i := len(before) - 1; i >= 0; i-- {
		items = append(items, before[i])
	}
	items = append(items, selected)

	for i := m.selectedIndex + 1; i < len(m.items); i++ {
		item := m.renderedItem(i, maxWidth)
		if used+item.h > maxHeight {
			break
		}
		items = append(items, item)
		used += item.h
	}

	return items
}

func (m *CommandHistoryModel) renderedItem(index, maxWidth int) renderedHistoryItem {
	content := m.renderer.RenderHistoryEntry(m.items[index], maxWidth, index == m.selectedIndex)
	w, h := lipgloss.Size(content)
	return renderedHistoryItem{
		index:   index,
		content: content,
		w:       w,
		h:       h,
	}
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
