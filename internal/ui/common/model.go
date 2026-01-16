package common

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type ImmediateModel interface {
	Init() tea.Cmd
	Update(msg tea.Msg) tea.Cmd
	ViewRect(dl *render.DisplayContext, box layout.Box)
}
