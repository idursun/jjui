package layoutview

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// SplitState tracks the current percentage and provides methods for layout and drag handling.
type SplitState struct {
	Percent    float64
	MinPercent float64
	MaxPercent float64
}

// NewSplitState creates a new SplitState with the given initial percentage.
func NewSplitState(percent float64) *SplitState {
	s := &SplitState{
		Percent:    percent,
		MinPercent: 10,
		MaxPercent: 95,
	}
	s.clamp()
	return s
}

func (s *SplitState) clamp() {
	if s.Percent < s.MinPercent {
		s.Percent = s.MinPercent
	}
	if s.Percent > s.MaxPercent {
		s.Percent = s.MaxPercent
	}
}

// DragTo calculates the new percentage from the given coordinates within the box.
// For vertical splits, y position determines the percentage.
// For horizontal splits, x position determines the percentage.
// Returns true if the percentage changed.
func (s *SplitState) DragTo(box layout.Box, vertical bool, x, y int) bool {
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

// Expand increases the secondary panel size by delta percentage.
func (s *SplitState) Expand(delta float64) {
	s.Percent += delta
	s.clamp()
}

// Shrink decreases the secondary panel size by delta percentage.
func (s *SplitState) Shrink(delta float64) {
	s.Percent -= delta
	s.clamp()
}

// Split represents a resizable split between two areas.
type Split struct {
	State              *SplitState
	Vertical           bool
	Primary            Slot
	Secondary          Slot
	SeparatorVisible   bool
	SeparatorThickness int
	lastBox            layout.Box
	hasLastBox         bool
}

// VSplit creates a vertical split (top/bottom).
func VSplit(state *SplitState, primary, secondary Slot) *Split {
	return &Split{
		State:            state,
		Vertical:         true,
		Primary:          primary,
		Secondary:        secondary,
		SeparatorVisible: true,
	}
}

// HSplit creates a horizontal split (left/right).
func HSplit(state *SplitState, primary, secondary Slot) *Split {
	return &Split{
		State:            state,
		Vertical:         false,
		Primary:          primary,
		Secondary:        secondary,
		SeparatorVisible: true,
	}
}

// Render lays out and renders the split.
func (s *Split) Render(dl *render.DisplayList, box layout.Box) {
	if s.State == nil {
		s.State = NewSplitState(50)
	}
	s.lastBox = box
	s.hasLastBox = true

	primaryVisible := s.Primary.Content != nil && s.Primary.Content.Visible()
	secondaryVisible := s.Secondary.Content != nil && s.Secondary.Content.Visible()

	switch {
	case primaryVisible && secondaryVisible:
		s.renderBoth(dl, box)
	case primaryVisible:
		s.Primary.Content.Render(dl, box)
	case secondaryVisible:
		s.Secondary.Content.Render(dl, box)
	}
}

func (s *Split) renderBoth(dl *render.DisplayList, box layout.Box) {
	primaryPercent := 100 - s.State.Percent
	var boxes []layout.Box
	if s.Vertical {
		boxes = box.V(layout.Percent(primaryPercent), layout.Fill(1))
	} else {
		boxes = box.H(layout.Percent(primaryPercent), layout.Fill(1))
	}
	if len(boxes) < 2 {
		return
	}
	s.Primary.Content.Render(dl, boxes[0])
	s.Secondary.Content.Render(dl, boxes[1])

	if !s.SeparatorVisible {
		return
	}
	thickness := s.SeparatorThickness
	if thickness <= 0 {
		thickness = 3
	}
	sepRect := separatorRect(box, boxes[1], s.Vertical, thickness)
	if sepRect.Dx() <= 0 || sepRect.Dy() <= 0 {
		return
	}
	dl.AddInteraction(sepRect, SplitDragMsg{Split: s}, render.InteractionDrag, 0)
	drawRect, content := separatorContent(sepRect, s.Vertical)
	if drawRect.Dx() > 0 && drawRect.Dy() > 0 && content != "" {
		dl.AddDraw(drawRect, content, 1)
	}
}

// Visible returns true if any child is visible.
func (s *Split) Visible() bool {
	return (s.Primary.Content != nil && s.Primary.Content.Visible()) ||
		(s.Secondary.Content != nil && s.Secondary.Content.Visible())
}

// DragTo updates the split state based on a drag position.
func (s *Split) DragTo(x, y int) bool {
	if s == nil || s.State == nil || !s.hasLastBox {
		return false
	}
	return s.State.DragTo(s.lastBox, s.Vertical, x, y)
}

// SplitDragMsg is sent when a split separator drag starts.
type SplitDragMsg struct {
	Split *Split
	X     int
	Y     int
}

// SetDragStart implements render.DragStartCarrier.
func (m SplitDragMsg) SetDragStart(x, y int) tea.Msg {
	m.X = x
	m.Y = y
	return m
}

func separatorRect(parent layout.Box, second layout.Box, vertical bool, thickness int) cellbuf.Rectangle {
	if thickness < 1 {
		thickness = 1
	}
	if vertical {
		boundaryY := second.R.Min.Y
		startY := boundaryY - thickness/2
		if startY < parent.R.Min.Y {
			startY = parent.R.Min.Y
		}
		if startY+thickness > parent.R.Max.Y {
			startY = parent.R.Max.Y - thickness
		}
		if startY < parent.R.Min.Y {
			startY = parent.R.Min.Y
		}
		height := thickness
		if parent.R.Dy() < height {
			height = parent.R.Dy()
		}
		return cellbuf.Rect(parent.R.Min.X, startY, parent.R.Dx(), height)
	}

	boundaryX := second.R.Min.X
	startX := boundaryX - thickness/2
	if startX < parent.R.Min.X {
		startX = parent.R.Min.X
	}
	if startX+thickness > parent.R.Max.X {
		startX = parent.R.Max.X - thickness
	}
	if startX < parent.R.Min.X {
		startX = parent.R.Min.X
	}
	width := thickness
	if parent.R.Dx() < width {
		width = parent.R.Dx()
	}
	return cellbuf.Rect(startX, parent.R.Min.Y, width, parent.R.Dy())
}

func separatorContent(sepRect cellbuf.Rectangle, vertical bool) (cellbuf.Rectangle, string) {
	if sepRect.Dx() <= 0 || sepRect.Dy() <= 0 {
		return cellbuf.Rectangle{}, ""
	}
	if vertical {
		centerY := sepRect.Min.Y + sepRect.Dy()/2
		drawRect := cellbuf.Rect(sepRect.Min.X, centerY, sepRect.Dx(), 1)
		return drawRect, strings.Repeat("-", drawRect.Dx())
	}
	centerX := sepRect.Min.X + sepRect.Dx()/2
	drawRect := cellbuf.Rect(centerX, sepRect.Min.Y, 1, sepRect.Dy())
	return drawRect, strings.Repeat("|", drawRect.Dy())
}
