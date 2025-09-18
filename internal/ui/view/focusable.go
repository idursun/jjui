package view

import tea "github.com/charmbracelet/bubbletea"

// Focusable is an interface for views that can receive and lose focus
type Focusable interface {
	OnFocus() tea.Cmd // Called when the view gains focus
	OnBlur() tea.Cmd  // Called when the view loses focus
}
