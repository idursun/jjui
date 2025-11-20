package common

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Rectangle struct {
	X, Y   int
	Width  int
	Height int
}

type SubModel interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (SubModel, tea.Cmd)
	View(frame Rectangle) *lipgloss.Layer
}
