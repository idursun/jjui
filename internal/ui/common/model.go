package common

import tea "github.com/charmbracelet/bubbletea"

type Model interface {
	Init() tea.Cmd
	Update(msg tea.Msg) tea.Cmd
	View() string
}
