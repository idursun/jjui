package split

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// SplitRenderer lays out two immediate-mode models with a draggable separator.
// It handles the low-level rectangle splitting, separator drawing, and drag hitbox.
type SplitRenderer struct {
	state              *SplitState
	vertical           bool
	separatorThickness int
	lastBox            layout.Box
	hasLastBox         bool
}

func NewRenderer(state *SplitState) *SplitRenderer {
	return &SplitRenderer{state: state}
}

func (s *SplitRenderer) Render(dl *render.DisplayContext, box layout.Box, primary, secondary common.ImmediateModel, vertical bool) {
	if s == nil {
		primary.ViewRect(dl, box)
		return
	}
	if s.state == nil {
		s.state = NewSplitState(50)
	}
	s.vertical = vertical
	s.lastBox = box
	s.hasLastBox = true

	s.renderBoth(dl, box, primary, secondary)
}

func (s *SplitRenderer) renderBoth(
	dl *render.DisplayContext,
	box layout.Box,
	primary common.ImmediateModel,
	secondary common.ImmediateModel,
) {
	primaryPercent := 100 - s.state.Percent
	thickness := s.separatorThickness
	if thickness <= 0 {
		thickness = 1
	}
	if s.vertical {
		if box.R.Dy() <= 0 {
			return
		}
		if thickness >= box.R.Dy() {
			thickness = 0
		}
		usable := box.R.Dy() - thickness
		splitBox := box
		if thickness > 0 {
			splitBox.R.Max.Y = splitBox.R.Min.Y + usable
		}
		boxes := splitBox.V(layout.Percent(primaryPercent), layout.Fill(1))
		if len(boxes) < 2 {
			return
		}
		if thickness > 0 {
			sepRect := layout.Rect(box.R.Min.X, boxes[0].R.Max.Y, box.R.Dx(), thickness)
			secondaryBox := boxes[1]
			secondaryBox.R.Min.Y += thickness
			secondaryBox.R.Max.Y += thickness
			primary.ViewRect(dl, boxes[0])
			secondary.ViewRect(dl, secondaryBox)
			dl.AddInteraction(sepRect, SplitDragMsg{Renderer: s}, render.InteractionDrag, 0)
			drawRect, content := separatorContent(sepRect, s.vertical)
			if drawRect.Dx() > 0 && drawRect.Dy() > 0 && content != "" {
				dl.AddDraw(drawRect, content, render.ZPreview)
			}
			return
		}
		primary.ViewRect(dl, boxes[0])
		secondary.ViewRect(dl, boxes[1])
		return
	}

	if box.R.Dx() <= 0 {
		return
	}
	if thickness >= box.R.Dx() {
		thickness = 0
	}
	usable := box.R.Dx() - thickness
	splitBox := box
	if thickness > 0 {
		splitBox.R.Max.X = splitBox.R.Min.X + usable
	}
	boxes := splitBox.H(layout.Percent(primaryPercent), layout.Fill(1))
	if len(boxes) < 2 {
		return
	}
	if thickness > 0 {
		sepRect := layout.Rect(boxes[0].R.Max.X, box.R.Min.Y, thickness, box.R.Dy())
		secondaryBox := boxes[1]
		secondaryBox.R.Min.X += thickness
		secondaryBox.R.Max.X += thickness
		primary.ViewRect(dl, boxes[0])
		secondary.ViewRect(dl, secondaryBox)
		dl.AddInteraction(sepRect, SplitDragMsg{Renderer: s}, render.InteractionDrag, 0)
		drawRect, content := separatorContent(sepRect, s.vertical)
		if drawRect.Dx() > 0 && drawRect.Dy() > 0 && content != "" {
			dl.AddDraw(drawRect, content, render.ZPreview)
		}
		return
	}
	primary.ViewRect(dl, boxes[0])
	secondary.ViewRect(dl, boxes[1])
}

func (s *SplitRenderer) DragTo(x, y int) bool {
	if s == nil || s.state == nil || !s.hasLastBox {
		return false
	}
	return s.state.DragTo(s.lastBox, s.vertical, x, y)
}

type SplitDragMsg struct {
	Renderer *SplitRenderer
	X        int
	Y        int
}

func (m SplitDragMsg) SetDragStart(x, y int) tea.Msg {
	m.X = x
	m.Y = y
	return m
}

func separatorContent(sepRect layout.Rectangle, vertical bool) (layout.Rectangle, string) {
	if sepRect.Dx() <= 0 || sepRect.Dy() <= 0 {
		return layout.Rectangle{}, ""
	}
	if vertical {
		centerY := sepRect.Min.Y + sepRect.Dy()/2
		drawRect := layout.Rect(sepRect.Min.X, centerY, sepRect.Dx(), 1)
		return drawRect, strings.Repeat("─", drawRect.Dx())
	}
	centerX := sepRect.Min.X + sepRect.Dx()/2
	drawRect := layout.Rect(centerX, sepRect.Min.Y, 1, sepRect.Dy())
	if drawRect.Dy() == 1 {
		return drawRect, "│"
	}
	return drawRect, strings.Repeat("│\n", drawRect.Dy()-1) + "│"
}
