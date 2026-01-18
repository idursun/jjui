package intents

type PreviewToggle struct{}

func (PreviewToggle) isIntent() {}

type PreviewToggleBottom struct{}

func (PreviewToggleBottom) isIntent() {}

type PreviewExpand struct{}

func (PreviewExpand) isIntent() {}

type PreviewShrink struct{}

func (PreviewShrink) isIntent() {}

type PreviewScrollKind int

const (
	PreviewScrollUp PreviewScrollKind = iota
	PreviewScrollDown
	PreviewHalfPageUp
	PreviewHalfPageDown
)

type PreviewScroll struct {
	Kind PreviewScrollKind
}

func (PreviewScroll) isIntent() {}
