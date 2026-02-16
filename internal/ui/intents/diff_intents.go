package intents

type DiffScrollKind int

const (
	DiffScrollUp DiffScrollKind = iota
	DiffScrollDown
	DiffPageUp
	DiffPageDown
	DiffHalfPageUp
	DiffHalfPageDown
)

//jjui:bind scope=diff action=scroll_up set=Kind:DiffScrollUp
//jjui:bind scope=diff action=scroll_down set=Kind:DiffScrollDown
//jjui:bind scope=diff action=page_up set=Kind:DiffPageUp
//jjui:bind scope=diff action=page_down set=Kind:DiffPageDown
//jjui:bind scope=diff action=half_page_up set=Kind:DiffHalfPageUp
//jjui:bind scope=diff action=half_page_down set=Kind:DiffHalfPageDown
type DiffScroll struct {
	Kind DiffScrollKind
}

func (DiffScroll) isIntent() {}

type DiffScrollHorizontalKind int

const (
	DiffScrollLeft DiffScrollHorizontalKind = iota
	DiffScrollRight
)

//jjui:bind scope=diff action=left set=Kind:DiffScrollLeft
//jjui:bind scope=diff action=right set=Kind:DiffScrollRight
type DiffScrollHorizontal struct {
	Kind DiffScrollHorizontalKind
}

func (DiffScrollHorizontal) isIntent() {}
