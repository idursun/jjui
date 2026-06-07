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

// initSplitContainer wires the preview pane into the reusable SplitContainer
// using the preview-specific config (placement, width, show-at-start).
func (m *Model) initSplitContainer() {
	m.splitContainer = split.NewSplitContainer()

	state := split.NewSplitState(config.Current.Preview.WidthPercentage)
	previewPositionCfg, err := config.GetPreviewPosition(config.Current)
	if err != nil {
		log.Fatal(err)
	}
	state.SetPlacement(splitPlacementFromPreviewConfig(previewPositionCfg))
	m.registerSplitContent(previewContentID, preview.New(m.context), state, config.Current.Preview.ShowAtStart)
}

// registerSplitContent registers a piece of content with the split container
// and optionally shows it immediately. It is the generic seam through which
// app-specific content (preview and future panes) is wired in.
func (m *Model) registerSplitContent(id string, content split.SplitContent, state *split.SplitState, showAtStart bool) {
	if m.splitContainer == nil {
		return
	}
	m.splitContainer.RegisterContent(id, content, state)
	if showAtStart {
		m.splitContainer.SetVisible(id, true)
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
		m.splitContainer.SetVisible(previewContentID, bool(msg))
		return m.selectionChanged(), true
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
		m.splitContainer.SetContent(previewContentID)
		return m.selectionChanged(), true
	case intents.PreviewToggleBottom:
		if !m.splitContainer.IsVisible(previewContentID) {
			m.splitContainer.SetContent(previewContentID)
			m.splitContainer.ToggleActivePosition()
			return m.selectionChanged(), true
		}
		m.splitContainer.ToggleActivePosition()
		return nil, true
	case intents.PreviewExpand:
		if m.splitContainer.IsVisible(previewContentID) {
			m.splitContainer.ResizeActive(config.Current.Preview.WidthIncrementPercentage)
		}
		return nil, true
	case intents.PreviewShrink:
		if m.splitContainer.IsVisible(previewContentID) {
			m.splitContainer.ResizeActive(-config.Current.Preview.WidthIncrementPercentage)
		}
		return nil, true
	case intents.PreviewShow:
		if !m.splitContainer.IsVisible(previewContentID) {
			m.splitContainer.SetContent(previewContentID)
		}
		content := m.splitContainer.ActiveContent()
		if content == nil {
			return nil, true
		}
		return content.Update(msg), true
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
	m.splitContainer.SetActiveAutoPosition(m.height >= m.width/2)
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
