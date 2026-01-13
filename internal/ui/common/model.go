package common

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type Model interface {
	Init() tea.Cmd
	Update(msg tea.Msg) tea.Cmd
	View() string
}

type ImmediateModel interface {
	Init() tea.Cmd
	Update(msg tea.Msg) tea.Cmd
	ViewRect(dl *render.DisplayList, box layout.Box)
}
