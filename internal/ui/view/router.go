package view

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
)

type IHasActionMap interface {
	GetActionMap() map[string]common.Action
}

type Router struct {
	Scope common.Scope
	Views map[common.Scope]tea.Model
}

func NewRouter(scope common.Scope) Router {
	return Router{
		Scope: scope,
		Views: make(map[common.Scope]tea.Model),
	}
}

func (r Router) Init() tea.Cmd {
	var cmds []tea.Cmd
	for k := range r.Views {
		cmds = append(cmds, r.Views[k].Init())
	}
	return tea.Batch(cmds...)
}

func (r Router) UpdateTargetView(action common.InvokeActionMsg) (Router, tea.Cmd) {
	if strings.HasPrefix(action.Action.Id, "close") {
		viewId := strings.TrimPrefix(action.Action.Id, "close ")
		if _, ok := r.Views[common.Scope(viewId)]; ok {
			delete(r.Views, common.Scope(viewId))
			r.Scope = action.Action.Switch
			return r, nil
		}
	}

	if strings.HasPrefix(action.Action.Id, "switch") {
		viewId := common.Scope(strings.TrimPrefix(action.Action.Id, "switch "))
		if _, ok := r.Views[viewId]; ok {
			r.Scope = viewId
			return r, nil
		}
	}

	var cmds []tea.Cmd
	for k := range r.Views {
		var cmd tea.Cmd
		r.Views[k], cmd = r.Views[k].Update(action)
		cmds = append(cmds, cmd)
	}
	if _, ok := r.Views[action.Action.Switch]; ok {
		r.Scope = action.Action.Switch
	}
	return r, tea.Batch(cmds...)
}

func (r Router) Update(msg tea.Msg) (Router, tea.Cmd) {
	switch msg := msg.(type) {
	case common.InvokeActionMsg:
		return r.UpdateTargetView(msg)
	case tea.KeyMsg:
		var cmd tea.Cmd
		if currentView, ok := r.Views[r.Scope]; ok {
			if hasActionMap, ok := currentView.(IHasActionMap); ok {
				actionMap := hasActionMap.GetActionMap()
				if action, ok := actionMap[msg.String()]; ok {
					return r, common.InvokeAction(action)
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
