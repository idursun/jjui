package split

import (
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type Panel interface {
	common.ImmediateModel
	Visible() bool
	SetVisible(bool)
}

type SplitPanel struct {
	State        *SplitState
	split        *Split
	BookmarkPane Panel
	PreviewPane  Panel
	focusedPane  bool // true when the secondary pane has focus
}

func NewSplitPanel(state *SplitState, bookmarkPane, previewPane Panel) *SplitPanel {
	return &SplitPanel{
		State:        state,
		split:        New(state, nil, nil),
		BookmarkPane: bookmarkPane,
		PreviewPane:  previewPane,
	}
}

func (sp *SplitPanel) ActiveSecondary() Panel {
	if sp.BookmarkPane != nil && sp.BookmarkPane.Visible() {
		return sp.BookmarkPane
	}
	if sp.PreviewPane != nil && sp.PreviewPane.Visible() {
		return sp.PreviewPane
	}
	return nil
}

func (sp *SplitPanel) SecondaryVisible() bool {
	return sp.ActiveSecondary() != nil
}

func (sp *SplitPanel) FocusedSecondary() bool {
	return sp.focusedPane && sp.ActiveSecondary() != nil
}

func (sp *SplitPanel) FocusPrimary() {
	sp.focusedPane = false
}

func (sp *SplitPanel) FocusSecondary() {
	sp.focusedPane = true
}

func (sp *SplitPanel) ToggleFocus() {
	if sp.ActiveSecondary() == nil {
		return
	}
	if _, ok := sp.ActiveSecondary().(common.Focusable); !ok {
		return
	}
	sp.focusedPane = !sp.focusedPane
}

func (sp *SplitPanel) OpenBookmark() {
	if sp.BookmarkPane == nil {
		return
	}
	if sp.PreviewPane != nil {
		sp.PreviewPane.SetVisible(false)
	}
	sp.BookmarkPane.SetVisible(true)
}

func (sp *SplitPanel) CloseBookmark() {
	if sp.BookmarkPane == nil {
		return
	}
	sp.BookmarkPane.SetVisible(false)
	sp.focusedPane = false
}

func (sp *SplitPanel) ShowPreview() {
	if sp.PreviewPane == nil {
		return
	}
	if sp.BookmarkPane != nil {
		sp.BookmarkPane.SetVisible(false)
	}
	sp.focusedPane = false
	sp.PreviewPane.SetVisible(true)
}

func (sp *SplitPanel) HidePreview() {
	if sp.PreviewPane == nil {
		return
	}
	sp.PreviewPane.SetVisible(false)
}

func (sp *SplitPanel) TogglePreview() {
	if sp.PreviewPane == nil {
		return
	}
	if sp.PreviewPane.Visible() {
		sp.HidePreview()
	} else {
		sp.ShowPreview()
	}
}

func (sp *SplitPanel) Render(dl *render.DisplayContext, box layout.Box, primary common.ImmediateModel) {
	secondary := sp.ActiveSecondary()
	if secondary == nil {
		primary.ViewRect(dl, box)
		return
	}
	sp.split.ViewModels(dl, box, primary, secondary, sp.State.AtBottom)
}
