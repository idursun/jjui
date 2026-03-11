package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/render"

	"github.com/idursun/jjui/internal/ui/bookmarkpane"
)

type secondaryPaneKind int

const (
	secondaryPaneNone secondaryPaneKind = iota
	secondaryPanePreview
	secondaryPaneBookmark
)

type secondaryPaneController struct {
	preview         *preview.Model
	bookmark        *bookmarkpane.Model
	revisions       interface{ SetFocused(bool) }
	previewSplit    *split
	bookmarkSplit   *split
	active          secondaryPaneKind
	restoreOnClose  secondaryPaneKind
	bookmarkFocused bool
}

func newSecondaryPaneController(previewModel *preview.Model, bookmarkModel *bookmarkpane.Model, revisionsModel interface{ SetFocused(bool) }) *secondaryPaneController {
	return &secondaryPaneController{
		preview:   previewModel,
		bookmark:  bookmarkModel,
		revisions: revisionsModel,
		previewSplit: newSplit(
			newSplitState(config.Current.Preview.WidthPercentage),
			nil,
			previewModel,
		),
		bookmarkSplit: newSplit(
			newSplitState(45),
			nil,
			bookmarkModel,
		),
	}
}

func (c *secondaryPaneController) syncPreviewOrientation() {
	if c == nil || c.previewSplit == nil || c.preview == nil {
		return
	}
	c.previewSplit.Vertical = c.preview.AtBottom()
}

func (c *secondaryPaneController) render(primary common.ImmediateModel, dl *render.DisplayContext, box layout.Box) {
	if c == nil {
		if primary != nil {
			primary.ViewRect(dl, box)
		}
		return
	}
	switch c.active {
	case secondaryPaneBookmark:
		if c.bookmarkSplit == nil {
			return
		}
		c.bookmarkSplit.Primary = primary
		c.bookmarkSplit.Secondary = c.bookmark
		c.bookmarkSplit.Vertical = false
		c.bookmarkSplit.SeparatorVisible = true
		c.bookmarkSplit.Render(dl, box)
	case secondaryPanePreview:
		if c.previewSplit == nil {
			return
		}
		c.previewSplit.Primary = primary
		c.previewSplit.Secondary = c.preview
		c.previewSplit.Render(dl, box)
	default:
		if primary != nil {
			primary.ViewRect(dl, box)
		}
	}
}

func (c *secondaryPaneController) currentSplit() *split {
	if c == nil {
		return nil
	}
	switch c.active {
	case secondaryPaneBookmark:
		return c.bookmarkSplit
	case secondaryPanePreview:
		return c.previewSplit
	default:
		return nil
	}
}

func (c *secondaryPaneController) previewVisible() bool {
	return c != nil && c.active == secondaryPanePreview && c.preview != nil && c.preview.Visible()
}

func (c *secondaryPaneController) bookmarkVisible() bool {
	return c != nil && c.active == secondaryPaneBookmark && c.bookmark != nil && c.bookmark.Visible()
}

func (c *secondaryPaneController) bookmarkEditing() bool {
	return c.bookmarkVisible() && c.bookmark != nil && c.bookmark.IsEditing()
}

func (c *secondaryPaneController) openBookmark() tea.Cmd {
	if c == nil || c.bookmark == nil {
		return nil
	}
	c.restoreOnClose = secondaryPaneNone
	if c.previewVisible() {
		c.restoreOnClose = secondaryPanePreview
		c.preview.SetVisible(false)
	}
	c.active = secondaryPaneBookmark
	c.bookmarkFocused = true
	c.bookmark.SetFocused(true)
	if c.revisions != nil {
		c.revisions.SetFocused(false)
	}
	return c.bookmark.Open()
}

func (c *secondaryPaneController) closeBookmark() {
	if c == nil || c.bookmark == nil {
		return
	}
	c.bookmark.Close()
	c.bookmarkFocused = false
	c.active = secondaryPaneNone
	if c.revisions != nil {
		c.revisions.SetFocused(true)
	}
	if c.restoreOnClose == secondaryPanePreview && c.preview != nil {
		c.preview.SetVisible(true)
		c.active = secondaryPanePreview
	}
	c.restoreOnClose = secondaryPaneNone
}

func (c *secondaryPaneController) showPreview() {
	if c == nil || c.preview == nil {
		return
	}
	if c.bookmarkVisible() {
		c.closeBookmark()
	}
	c.preview.SetVisible(true)
	c.active = secondaryPanePreview
}

func (c *secondaryPaneController) hidePreview() {
	if c == nil || c.preview == nil {
		return
	}
	c.preview.SetVisible(false)
	if c.active == secondaryPanePreview {
		c.active = secondaryPaneNone
	}
}

func (c *secondaryPaneController) togglePreview() {
	if c == nil || c.preview == nil {
		return
	}
	if c.previewVisible() {
		c.hidePreview()
		return
	}
	c.showPreview()
}

func (c *secondaryPaneController) focusNext() {
	if !c.bookmarkVisible() || c.bookmark == nil {
		return
	}
	c.bookmarkFocused = !c.bookmarkFocused
	c.bookmark.SetFocused(c.bookmarkFocused)
	if c.revisions != nil {
		c.revisions.SetFocused(!c.bookmarkFocused)
	}
}

func (c *secondaryPaneController) primaryScope() keybindings.Scope {
	if c.bookmarkVisible() && c.bookmarkFocused {
		if c.bookmarkEditing() {
			return keybindings.Scope("bookmark_view.filter")
		}
		return keybindings.Scope("bookmark_view")
	}
	return ""
}
