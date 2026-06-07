package split

import "github.com/idursun/jjui/internal/ui/layout"

type Placement int

const (
	PlacementAuto Placement = iota
	PlacementBottom
	PlacementRight
)

type SplitState struct {
	Percent      float64
	MinPercent   float64
	MaxPercent   float64
	AtBottom     bool
	AutoPosition bool
}

func NewSplitState(percent float64) *SplitState {
	s := &SplitState{
		Percent:    percent,
		MinPercent: 10,
		MaxPercent: 95,
	}
	s.clamp()
	return s
}

func (s *SplitState) SetPlacement(placement Placement) {
	s.AutoPosition = placement == PlacementAuto
	s.AtBottom = placement == PlacementBottom
}

func (s *SplitState) TogglePosition() {
	s.AutoPosition = false
	s.AtBottom = !s.AtBottom
}

func (s *SplitState) clamp() {
	if s.Percent < s.MinPercent {
		s.Percent = s.MinPercent
	}
	if s.Percent > s.MaxPercent {
		s.Percent = s.MaxPercent
	}
}

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

func (s *SplitState) Expand(delta float64) {
	s.Percent += delta
	s.clamp()
}

func (s *SplitState) Shrink(delta float64) {
	s.Percent -= delta
	s.clamp()
}
