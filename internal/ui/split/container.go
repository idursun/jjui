package split

import (
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// SplitContent is a renderable model that can be shown inside a SplitContainer.
type SplitContent interface {
	common.ImmediateModel
	OnShow()
}

// SplitContainer manages one optional split-content pane next to a primary view.
type SplitContainer struct {
	renderer *SplitRenderer
	state    *SplitState
	dragging bool
	contents map[string]SplitContent
	activeID string
}

func NewSplitContainer(state *SplitState) *SplitContainer {
	return &SplitContainer{
		renderer: NewRenderer(state, nil, nil),
		state:    state,
		contents: make(map[string]SplitContent),
	}
}

func (sc *SplitContainer) RegisterContent(id string, content SplitContent) {
	sc.contents[id] = content
}

func (sc *SplitContainer) ShowContent(id string) bool {
	if sc.activeID == id {
		return false
	}
	sc.activeID = id
	sc.contents[id].OnShow()
	return true
}

func (sc *SplitContainer) ToggleContent(id string) bool {
	if sc.activeID != "" && sc.activeID == id {
		sc.Close()
		return true
	}
	return sc.ShowContent(id)
}

func (sc *SplitContainer) Close() bool {
	if sc.activeID == "" {
		return false
	}
	sc.activeID = ""
	return true
}

func (sc *SplitContainer) Resize(delta float64) bool {
	old := sc.state.Percent
	if delta > 0 {
		sc.state.Expand(delta)
	} else if delta < 0 {
		sc.state.Shrink(-delta)
	}
	return sc.state.Percent != old
}

func (sc *SplitContainer) TogglePosition() {
	sc.state.TogglePosition()
}

func (sc *SplitContainer) SetAutoPosition(atBottom bool) {
	if sc.state.AutoPosition {
		sc.state.AtBottom = atBottom
	}
}

func (sc *SplitContainer) Update(msg tea.Msg) tea.Cmd {
	content := sc.activeContent()
	if content == nil {
		return nil
	}
	return content.Update(msg)
}

func (sc *SplitContainer) Scopes() []common.Scope {
	content := sc.activeContent()
	if content == nil {
		return nil
	}
	provider, ok := content.(common.ScopeProvider)
	if !ok {
		return nil
	}
	return provider.Scopes()
}

func (sc *SplitContainer) StartDrag(msg SplitDragMsg) {
	if msg.Renderer != sc.renderer {
		return
	}
	sc.dragging = true
	sc.renderer.DragTo(msg.X, msg.Y)
}

func (sc *SplitContainer) DragTo(x, y int) bool {
	if !sc.dragging {
		return false
	}
	sc.renderer.DragTo(x, y)
	return true
}

func (sc *SplitContainer) EndDrag() {
	sc.dragging = false
}

func (sc *SplitContainer) Render(dl *render.DisplayContext, box layout.Box, primary common.ImmediateModel) {
	content := sc.activeContent()
	if content == nil {
		primary.ViewRect(dl, box)
		return
	}
	sc.renderer.State = sc.state
	sc.renderer.ViewModels(dl, box, primary, content, sc.state.AtBottom)
}

func (sc *SplitContainer) activeContent() SplitContent {
	if sc.activeID == "" {
		return nil
	}
	return sc.contents[sc.activeID]
}
