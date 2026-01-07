package common

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/ops"
)

type Model interface {
	Init() tea.Cmd
	Update(msg tea.Msg) tea.Cmd
	ViewRect(box layout.Box) *ops.DisplayList
}
