package intents

//jjui:bind scope=ui action=preview_toggle
type PreviewToggle struct{}

func (PreviewToggle) isIntent() {}

//jjui:bind scope=ui action=preview_toggle_bottom
type PreviewToggleBottom struct{}

func (PreviewToggleBottom) isIntent() {}

//jjui:bind scope=ui action=preview_expand
type PreviewExpand struct{}

func (PreviewExpand) isIntent() {}

//jjui:bind scope=ui action=preview_shrink
type PreviewShrink struct{}

func (PreviewShrink) isIntent() {}

type PreviewScrollKind int

const (
	PreviewScrollUp PreviewScrollKind = iota
	PreviewScrollDown
	PreviewHalfPageUp
	PreviewHalfPageDown
)

//jjui:bind scope=ui action=preview_scroll_up set=Kind:PreviewScrollUp
//jjui:bind scope=ui action=preview_scroll_down set=Kind:PreviewScrollDown
//jjui:bind scope=ui action=preview_half_page_up set=Kind:PreviewHalfPageUp
//jjui:bind scope=ui action=preview_half_page_down set=Kind:PreviewHalfPageDown
type PreviewScroll struct {
	Kind PreviewScrollKind
}

func (PreviewScroll) isIntent() {}
