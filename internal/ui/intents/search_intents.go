package intents

import tea "github.com/charmbracelet/bubbletea"

type QuickSearch struct{}

func (QuickSearch) isIntent() {}

type QuickSearchCycle struct{}

func (QuickSearchCycle) isIntent() {}

type FileSearchToggle struct{}

func (FileSearchToggle) isIntent() {}

type StartAceJump struct{}

func (StartAceJump) isIntent() {}

type FileSearchNavigate struct {
	Delta int
}

func (FileSearchNavigate) isIntent() {}

type FileSearchCancel struct{}

func (FileSearchCancel) isIntent() {}

type FileSearchEdit struct{}

func (FileSearchEdit) isIntent() {}

type FileSearchTogglePreview struct{}

func (FileSearchTogglePreview) isIntent() {}

type FileSearchAccept struct{}

func (FileSearchAccept) isIntent() {}

type FileSearchPreviewScroll struct {
	Kind PreviewScrollKind
}

func (FileSearchPreviewScroll) isIntent() {}

type FileSearchRevisionNavigate struct {
	Key tea.KeyMsg
}

func (FileSearchRevisionNavigate) isIntent() {}
