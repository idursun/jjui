package intents

import tea "github.com/charmbracelet/bubbletea"

// Intent represents a high-level action the revisions view can perform.
// It decouples inputs (keyboard/mouse/macros) from the actual capability.
type Intent interface {
	//apply(*Model) tea.Cmd
	isIntent()
}

func Invoke(intent Intent) tea.Cmd {
	return func() tea.Msg {
		return intent
	}
}
