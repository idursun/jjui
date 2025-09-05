package view

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
)

type ListId int

const (
	ListRevisions ListId = iota
	ListFiles
	ListOplog
	ListEvolog
)

type ViewId string

type BaseView struct {
	tea.Model
	*common.Sizeable
	Id       ViewId
	Visible  bool
	Focused  bool
	LayoutFn func()
}

func (v *BaseView) Layout() {
	if v.LayoutFn != nil {
		v.LayoutFn()
	}
}
