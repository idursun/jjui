package intents

import tea "charm.land/bubbletea/v2"

// Intent represents a high-level action the revisions view can perform.
// It decouples inputs (keyboard/mouse/macros) from the actual capability.
type Intent interface {
	isIntent()
}

func Invoke(intent Intent) tea.Cmd {
	return func() tea.Msg {
		return intent
	}
}
