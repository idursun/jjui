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

// SplitContainer manages the split between the primary view and split content.
// It is content-agnostic: callers register content by id via RegisterContent
// and drive visibility, placement, and sizing through the exported methods.
type SplitContainer struct {
	// renderer owns the concrete split layout for the currently rendered primary
	// view and split content.
	renderer *SplitRenderer
	dragging bool
	contents map[string]SplitContent
	states   map[string]*SplitState
	activeID string
}

func NewSplitContainer() *SplitContainer {
	return &SplitContainer{
		renderer: NewRenderer(nil, nil, nil),
		contents: make(map[string]SplitContent),
		states:   make(map[string]*SplitState),
	}
}

func (sc *SplitContainer) RegisterContent(id string, content SplitContent, state *SplitState) {
	if sc == nil || id == "" || content == nil {
		return
	}
	if state == nil {
		state = NewSplitState(50)
	}
	if sc.contents == nil {
		sc.contents = make(map[string]SplitContent)
	}
	if sc.states == nil {
		sc.states = make(map[string]*SplitState)
	}
	sc.contents[id] = content
	sc.states[id] = state
}

func (sc *SplitContainer) ActiveContent() SplitContent {
	if sc == nil || sc.activeID == "" {
		return nil
	}
	return sc.contents[sc.activeID]
}

func (sc *SplitContainer) activeState() *SplitState {
	if sc == nil || sc.activeID == "" || sc.contents[sc.activeID] == nil {
		return nil
	}
	return sc.states[sc.activeID]
}

func (sc *SplitContainer) SetActiveAutoPosition(atBottom bool) {
	state := sc.activeState()
	if state == nil || !state.AutoPosition {
		return
	}
	state.AtBottom = atBottom
}

func (sc *SplitContainer) ToggleActivePosition() bool {
	state := sc.activeState()
	if state == nil {
		return false
	}
	state.TogglePosition()
	return true
}

func (sc *SplitContainer) ResizeActive(delta float64) bool {
	state := sc.activeState()
	if state == nil {
		return false
	}
	old := state.Percent
	if delta > 0 {
		state.Expand(delta)
	} else if delta < 0 {
		state.Shrink(-delta)
	}
	return state.Percent != old
}

func (sc *SplitContainer) ActivePercent() (float64, bool) {
	state := sc.activeState()
	if state == nil {
		return 0, false
	}
	return state.Percent, true
}

// ActiveID reports the id of the content currently displayed, or "" if none.
func (sc *SplitContainer) ActiveID() string {
	if sc == nil {
		return ""
	}
	return sc.activeID
}

// IsVisible reports whether the content with the given id is currently active.
func (sc *SplitContainer) IsVisible(id string) bool {
	return sc != nil && sc.activeID == id
}

// SetContent selects which registered content is displayed. Selecting the
// content that is already active closes it (toggle off); selecting a different
// content swaps to it and fires its OnShow hook. Unknown ids are ignored.
func (sc *SplitContainer) SetContent(id string) {
	if sc == nil {
		return
	}
	if id == sc.activeID {
		sc.activeID = ""
		return
	}
	if id == "" || sc.contents[id] == nil {
		return
	}
	sc.activeID = id
	sc.contents[id].OnShow()
}

// SetVisible ensures the content with the given id is shown (visible=true)
// When showing, it swaps in the content; when hiding, it only closes the
// content if it is currently active. Already-satisfied requests are no-ops.
func (sc *SplitContainer) SetVisible(id string, visible bool) {
	if sc == nil {
		return
	}
	if visible == (sc.activeID == id) {
		return
	}
	if visible {
		sc.SetContent(id)
		return
	}
	if sc.activeID == id {
		sc.SetContent(id)
	}
}

func (sc *SplitContainer) Update(msg tea.Msg) tea.Cmd {
	content := sc.ActiveContent()
	if content == nil {
		return nil
	}
	return content.Update(msg)
}

func (sc *SplitContainer) Scopes() []common.Scope {
	content := sc.ActiveContent()
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
	if sc == nil || sc.renderer == nil || msg.Renderer != sc.renderer {
		return
	}
	sc.dragging = true
	sc.renderer.DragTo(msg.X, msg.Y)
}

func (sc *SplitContainer) DragTo(x, y int) bool {
	if sc == nil || !sc.dragging || sc.renderer == nil {
		return false
	}
	sc.renderer.DragTo(x, y)
	return true
}

func (sc *SplitContainer) EndDrag() {
	if sc == nil {
		return
	}
	sc.dragging = false
}

func (sc *SplitContainer) Render(dl *render.DisplayContext, box layout.Box, primary common.ImmediateModel) {
	content := sc.ActiveContent()
	state := sc.activeState()
	if content == nil || state == nil {
		primary.ViewRect(dl, box)
		return
	}
	sc.renderer.State = state
	sc.renderer.ViewModels(dl, box, primary, content, state.AtBottom)
}
