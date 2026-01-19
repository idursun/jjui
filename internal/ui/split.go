package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type splitState struct {
	Percent    float64
	MinPercent float64
	MaxPercent float64
}

func newSplitState(percent float64) *splitState {
	s := &splitState{
		Percent:    percent,
		MinPercent: 10,
		MaxPercent: 95,
	}
	s.clamp()
	return s
}

func (s *splitState) clamp() {
	if s.Percent < s.MinPercent {
		s.Percent = s.MinPercent
	}
	if s.Percent > s.MaxPercent {
		s.Percent = s.MaxPercent
	}
}

func (s *splitState) DragTo(box layout.Box, vertical bool, x, y int) bool {
	old := s.Percent
	if vertical {
		total := box.R.Dy()
		if total <= 0 {
			return false
		}
		distanceFromBottom := box.R.Max.Y - y
		s.Percent = float64(distanceFromBottom*100) / float64(total)
	} else {
		total := box.R.Dx()
		if total <= 0 {
			return false
		}
		distanceFromRight := box.R.Max.X - x
		s.Percent = float64(distanceFromRight*100) / float64(total)
	}
	s.clamp()
	return s.Percent != old
}

func (s *splitState) Expand(delta float64) {
	s.Percent += delta
	s.clamp()
}

func (s *splitState) Shrink(delta float64) {
	s.Percent -= delta
	s.clamp()
}

type split struct {
	State              *splitState
	Vertical           bool
	Primary            common.ImmediateModel
	Secondary          common.ImmediateModel
	SeparatorVisible   bool
	SeparatorThickness int
	lastBox            layout.Box
	hasLastBox         bool
}

func newSplit(state *splitState, primary, secondary common.ImmediateModel) *split {
	return &split{
		State:            state,
		Primary:          primary,
		Secondary:        secondary,
		SeparatorVisible: true,
	}
}

func (s *split) Render(dl *render.DisplayContext, box layout.Box) {
	if s.State == nil {
		s.State = newSplitState(50)
	}
	s.lastBox = box
	s.hasLastBox = true

	primaryVisible := isVisible(s.Primary)
	secondaryVisible := isVisible(s.Secondary)

	switch {
	case primaryVisible && secondaryVisible:
		s.renderBoth(dl, box)
	case primaryVisible:
		s.Primary.ViewRect(dl, box)
	case secondaryVisible:
		s.Secondary.ViewRect(dl, box)
	}
}

func (s *split) renderBoth(dl *render.DisplayContext, box layout.Box) {
	primaryPercent := 100 - s.State.Percent
	thickness := s.SeparatorThickness
	if thickness <= 0 {
		thickness = 1
	}
	if !s.SeparatorVisible {
		thickness = 0
	}
	if s.Vertical {
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
			sepRect := cellbuf.Rect(box.R.Min.X, boxes[0].R.Max.Y, box.R.Dx(), thickness)
			secondaryBox := boxes[1]
			secondaryBox.R.Min.Y += thickness
			secondaryBox.R.Max.Y += thickness
			s.Primary.ViewRect(dl, boxes[0])
			s.Secondary.ViewRect(dl, secondaryBox)
			dl.AddInteraction(sepRect, SplitDragMsg{Split: s}, render.InteractionDrag, 0)
			drawRect, content := separatorContent(sepRect, s.Vertical)
			if drawRect.Dx() > 0 && drawRect.Dy() > 0 && content != "" {
				dl.AddDraw(drawRect, content, 1)
			}
			return
		}
		s.Primary.ViewRect(dl, boxes[0])
		s.Secondary.ViewRect(dl, boxes[1])
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
		sepRect := cellbuf.Rect(boxes[0].R.Max.X, box.R.Min.Y, thickness, box.R.Dy())
		secondaryBox := boxes[1]
		secondaryBox.R.Min.X += thickness
		secondaryBox.R.Max.X += thickness
		s.Primary.ViewRect(dl, boxes[0])
		s.Secondary.ViewRect(dl, secondaryBox)
		dl.AddInteraction(sepRect, SplitDragMsg{Split: s}, render.InteractionDrag, 0)
		drawRect, content := separatorContent(sepRect, s.Vertical)
		if drawRect.Dx() > 0 && drawRect.Dy() > 0 && content != "" {
			dl.AddDraw(drawRect, content, 1)
		}
		return
	}
	s.Primary.ViewRect(dl, boxes[0])
	s.Secondary.ViewRect(dl, boxes[1])
}

func (s *split) DragTo(x, y int) bool {
	if s == nil || s.State == nil || !s.hasLastBox {
		return false
	}
	return s.State.DragTo(s.lastBox, s.Vertical, x, y)
}

type SplitDragMsg struct {
	Split *split
	X     int
	Y     int
}

func (m SplitDragMsg) SetDragStart(x, y int) tea.Msg {
	m.X = x
	m.Y = y
	return m
}

func isVisible(m common.ImmediateModel) bool {
	if m == nil {
		return false
	}
	if v, ok := m.(interface{ Visible() bool }); ok {
		return v.Visible()
	}
	return true
}

func separatorContent(sepRect cellbuf.Rectangle, vertical bool) (cellbuf.Rectangle, string) {
	if sepRect.Dx() <= 0 || sepRect.Dy() <= 0 {
		return cellbuf.Rectangle{}, ""
	}
	if vertical {
		centerY := sepRect.Min.Y + sepRect.Dy()/2
		drawRect := cellbuf.Rect(sepRect.Min.X, centerY, sepRect.Dx(), 1)
		return drawRect, strings.Repeat("─", drawRect.Dx())
	}
	centerX := sepRect.Min.X + sepRect.Dx()/2
	drawRect := cellbuf.Rect(centerX, sepRect.Min.Y, 1, sepRect.Dy())
	if drawRect.Dy() == 1 {
		return drawRect, "│"
	}
	return drawRect, strings.Repeat("│\n", drawRect.Dy()-1) + "│"
}
