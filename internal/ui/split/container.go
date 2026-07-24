package split

import (
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// SplitContent is content that can be displayed beside the primary view.
type SplitContent interface {
	common.ImmediateModel
	common.ScopeProvider
	OnShow() tea.Cmd
	OnHide()
	SetFocused(bool)
}

type overlayContent interface {
	RenderOverlay(dl *render.DisplayContext, box layout.Box)
}

// SplitContainer manages one optional content pane beside a primary view.
type SplitContainer struct {
	renderer        *SplitRenderer
	state           *SplitState
	contents        map[string]SplitContent
	activeContentID string
	contentFocused  bool
	dragging        bool

	// OnPrimaryFocus is called when the primary view gains focus, registered as
	// a callback to the model's Update() method.
	OnPrimaryFocus func(bool)
}

func NewSplitContainer(state *SplitState) *SplitContainer {
	return &SplitContainer{
		renderer: NewRenderer(state),
		state:    state,
		contents: make(map[string]SplitContent),
	}
}

// setContentFocused is the single place that mutates focus. It keeps the
// active content's focus, the contentFocused flag, and the primary view's
// focus in sync.
func (sc *SplitContainer) setContentFocused(focused bool) {
	if focused {
		if sc.activeContent() == nil {
			return
		}
	}
	sc.contentFocused = focused
	if content := sc.activeContent(); content != nil {
		content.SetFocused(focused)
	}
	if sc.OnPrimaryFocus != nil {
		sc.OnPrimaryFocus(!focused)
	}
}

func (sc *SplitContainer) RegisterContent(id string, content SplitContent) {
	sc.contents[id] = content
}

func (sc *SplitContainer) ActiveID() string {
	return sc.activeContentID
}

func (sc *SplitContainer) ContentFocused() bool {
	return sc.contentFocused
}

func (sc *SplitContainer) ShowContent(id string) (tea.Cmd, bool) {
	if sc.activeContentID == id {
		return nil, false
	}
	content, ok := sc.contents[id]
	if !ok {
		return nil, false
	}
	if active := sc.activeContent(); active != nil {
		active.SetFocused(false)
		active.OnHide()
	}
	sc.activeContentID = id
	sc.setContentFocused(false)
	return content.OnShow(), true
}

func (sc *SplitContainer) ToggleContent(id string) (tea.Cmd, bool) {
	if sc.activeContentID == id {
		return nil, sc.Close()
	}
	return sc.ShowContent(id)
}

func (sc *SplitContainer) Close() bool {
	content := sc.activeContent()
	if content == nil {
		return false
	}
	sc.setContentFocused(false)
	content.OnHide()
	sc.activeContentID = ""
	return true
}

func (sc *SplitContainer) FocusSplitContent() {
	sc.setContentFocused(true)
}

func (sc *SplitContainer) FocusPrimary() {
	sc.setContentFocused(false)
}

func (sc *SplitContainer) ToggleFocus() {
	sc.setContentFocused(!sc.contentFocused)
}

func (sc *SplitContainer) Resize(delta float64) {
	if delta > 0 {
		sc.state.Expand(delta)
	} else if delta < 0 {
		sc.state.Shrink(-delta)
	}
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

func (sc *SplitContainer) UpdateContent(id string, msg tea.Msg) (tea.Cmd, bool) {
	content, ok := sc.contents[id]
	if !ok {
		return nil, false
	}
	return content.Update(msg), true
}

func (sc *SplitContainer) Scopes(primary []common.Scope) []common.Scope {
	content := sc.activeContent()
	if content == nil {
		return primary
	}
	if sc.contentFocused {
		return content.Scopes()
	}
	return append(primary, content.Scopes()...)
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
	sc.renderer.Render(dl, box, primary, content, sc.state.AtBottom)
}

func (sc *SplitContainer) RenderOverlay(dl *render.DisplayContext, box layout.Box) {
	content, ok := sc.activeContent().(overlayContent)
	if !ok {
		return
	}
	content.RenderOverlay(dl, box)
}

func (sc *SplitContainer) activeContent() SplitContent {
	if sc.activeContentID == "" {
		return nil
	}
	return sc.contents[sc.activeContentID]
}
