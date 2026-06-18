package ui

import (
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/split"
)

const previewContentID = "preview"

// initSplitContainer wires split layout state and the preview split content.
func (m *Model) initSplitContainer() {
	state := split.NewSplitState(config.Current.Preview.WidthPercentage)
	previewPositionCfg, err := config.GetPreviewPosition(config.Current)
	if err != nil {
		log.Fatal(err)
	}
	state.SetPlacement(splitPlacementFromPreviewConfig(previewPositionCfg))

	m.splitContainer = split.NewSplitContainer(state)
	m.splitContainer.RegisterContent(previewContentID, preview.New(m.context))
	if config.Current.Preview.ShowAtStart {
		m.splitContainer.ShowContent(previewContentID)
	}
}

func splitPlacementFromPreviewConfig(position config.PreviewPosition) split.Placement {
	switch position {
	case config.PreviewPositionBottom:
		return split.PlacementBottom
	case config.PreviewPositionRight:
		return split.PlacementRight
	default:
		return split.PlacementAuto
	}
}

// selectionChanged emits a SelectionChangedMsg for the currently selected
// item. It is fired whenever the preview becomes visible so the freshly shown
// pane loads content for the current selection instead of staying empty until
// the next cursor move.
func (m *Model) selectionChanged() tea.Cmd {
	if m.context == nil {
		return nil
	}
	return common.SelectionChanged(m.context.SelectedItem)
}

func (m *Model) handleSplitMsg(msg tea.Msg) (tea.Cmd, bool) {
	if m.splitContainer == nil {
		return nil, false
	}
	switch msg := msg.(type) {
	case common.ShowPreview:
		if m.splitContainer.ShowContent(previewContentID) {
			return m.selectionChanged(), true
		}
		return nil, true
	case split.SplitDragMsg:
		m.splitContainer.StartDrag(msg)
		return nil, false
	}
	return nil, false
}

func (m *Model) handleSplitIntent(intent intents.Intent) (tea.Cmd, bool) {
	if m.splitContainer == nil {
		return nil, false
	}
	switch msg := intent.(type) {
	case intents.PreviewToggle:
		m.splitContainer.ToggleContent(previewContentID)
		return m.selectionChanged(), true
	case intents.PreviewToggleBottom:
		shown := m.splitContainer.ShowContent(previewContentID)
		m.splitContainer.TogglePosition()
		if shown {
			return m.selectionChanged(), true
		}
		return nil, true
	case intents.PreviewExpand:
		m.splitContainer.Resize(config.Current.Preview.WidthIncrementPercentage)
		return nil, true
	case intents.PreviewShrink:
		m.splitContainer.Resize(-config.Current.Preview.WidthIncrementPercentage)
		return nil, true
	case intents.PreviewShow:
		m.splitContainer.ShowContent(previewContentID)
		return m.splitContainer.Update(msg), true
	}
	return nil, false
}

func (m *Model) updateSplit(msg tea.Msg) tea.Cmd {
	if m.splitContainer == nil {
		return nil
	}
	return m.splitContainer.Update(msg)
}

func (m *Model) splitScopes() []common.Scope {
	if m.splitContainer == nil {
		return nil
	}
	return m.splitContainer.Scopes()
}

func (m *Model) renderSplit(primary common.ImmediateModel, box layout.Box) {
	if m.splitContainer == nil {
		primary.ViewRect(m.displayContext, box)
		return
	}
	m.splitContainer.Render(m.displayContext, box, primary)
}

func (m *Model) updateSplitAutoPosition() {
	if m.splitContainer == nil {
		return
	}
	m.splitContainer.SetAutoPosition(m.height >= m.width/2)
}

func (m *Model) handleSplitMouseMsg(msg tea.Msg) (tea.Cmd, bool) {
	if m.splitContainer == nil {
		return nil, false
	}
	switch msg := msg.(type) {
	case tea.MouseReleaseMsg:
		m.splitContainer.EndDrag()
		return nil, false
	case tea.MouseMotionMsg:
		mouse := msg.Mouse()
		if m.splitContainer.DragTo(mouse.X, mouse.Y) {
			return nil, true
		}
	}
	return nil, false
}
