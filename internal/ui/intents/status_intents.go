package intents

type StatusScrollKind int

const (
	StatusScrollUp StatusScrollKind = iota
	StatusScrollDown
	StatusPageUp
	StatusPageDown
	StatusHalfPageUp
	StatusHalfPageDown
)

//jjui:bind scope=ui action=status_scroll_up set=Kind:StatusScrollUp
//jjui:bind scope=ui action=status_scroll_down set=Kind:StatusScrollDown
//jjui:bind scope=ui action=status_page_up set=Kind:StatusPageUp
//jjui:bind scope=ui action=status_page_down set=Kind:StatusPageDown
//jjui:bind scope=ui action=status_half_page_up set=Kind:StatusHalfPageUp
//jjui:bind scope=ui action=status_half_page_down set=Kind:StatusHalfPageDown
type StatusScroll struct {
	Kind StatusScrollKind
}

func (StatusScroll) isIntent() {}
