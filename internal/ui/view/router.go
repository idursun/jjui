package view

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
)

type IHasActionMap interface {
	GetActionMap() map[string]actions.Action
}

var _ common.ContextProvider = (*Router)(nil)

type Router struct {
	Scope actions.Scope
	Views map[actions.Scope]tea.Model
}

func NewRouter(scope actions.Scope) Router {
	return Router{
		Scope: scope,
		Views: make(map[actions.Scope]tea.Model),
	}
}

func (r Router) Init() tea.Cmd {
	var cmds []tea.Cmd
	for k := range r.Views {
		cmds = append(cmds, r.Views[k].Init())
	}
	return tea.Batch(cmds...)
}

func (r Router) UpdateTargetView(action actions.InvokeActionMsg) (Router, tea.Cmd) {
	if strings.HasPrefix(action.Action.Id, "close ") {
		viewId := strings.TrimPrefix(action.Action.Id, "close ")
		if _, ok := r.Views[actions.Scope(viewId)]; ok {
			delete(r.Views, actions.Scope(viewId))
			return r, action.Action.GetNext()
		}
	}

	if strings.HasPrefix(action.Action.Id, "switch ") {
		viewId := actions.Scope(strings.TrimPrefix(action.Action.Id, "switch "))
		if _, ok := r.Views[viewId]; ok {
			r.Scope = viewId
			return r, action.Action.GetNext()
		}
	}

	var cmds []tea.Cmd
	for k := range r.Views {
		var cmd tea.Cmd
		r.Views[k], cmd = r.Views[k].Update(action)
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
}

func (r Router) Update(msg tea.Msg) (Router, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		return r.UpdateTargetView(msg)
	case tea.KeyMsg:
		var cmd tea.Cmd
		if currentView, ok := r.Views[r.Scope]; ok {
			if hasActionMap, ok := currentView.(IHasActionMap); ok {
				actionMap := hasActionMap.GetActionMap()
				if action, ok := actionMap[msg.String()]; ok {
					return r, actions.InvokeAction(action)
				}
			}
			r.Views[r.Scope], cmd = r.Views[r.Scope].Update(msg)
			return r, cmd
		}
	}

	var cmds []tea.Cmd
	for k := range r.Views {
		var cmd tea.Cmd
		r.Views[k], cmd = r.Views[k].Update(msg)
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
}

func (r Router) View() string {
	return ""
}

func (r Router) Read(value string) string {
	for _, v := range r.Views {
		if v, ok := v.(common.ContextProvider); ok {
			ret := v.Read(value)
			if ret != "" {
				return ret
			}
		}
	}
	return ""
}
