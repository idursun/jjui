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
	scopes  []actions.Scope
	Scope   actions.Scope
	Views   map[actions.Scope]tea.Model
	waiters map[string]actions.WaitChannel
}

func NewRouter(scope actions.Scope) Router {
	return Router{
		scopes:  []actions.Scope{scope},
		Scope:   scope,
		Views:   make(map[actions.Scope]tea.Model),
		waiters: make(map[string]actions.WaitChannel),
	}
}

func (r Router) Init() tea.Cmd {
	var cmds []tea.Cmd
	for k := range r.Views {
		cmds = append(cmds, r.Views[k].Init())
	}
	return tea.Batch(cmds...)
}

func (r Router) handleAndRouteAction(action actions.InvokeActionMsg) (Router, tea.Cmd) {
	if strings.HasPrefix(action.Action.Id, "close ") {
		viewId := strings.TrimPrefix(action.Action.Id, "close ")
		if _, ok := r.Views[actions.Scope(viewId)]; ok {
			delete(r.Views, actions.Scope(viewId))
			r.scopes = r.scopes[:len(r.scopes)-1]
			r.Scope = r.scopes[len(r.scopes)-1]
		}
	}

	if strings.HasPrefix(action.Action.Id, "switch ") {
		viewId := actions.Scope(strings.TrimPrefix(action.Action.Id, "switch "))
		if _, ok := r.Views[viewId]; ok && r.Scope != viewId {
			r.Scope = viewId
		}
	}

	if len(r.waiters) > 0 {
		for k, ch := range r.waiters {
			if k == action.Action.Id {
				ch <- actions.WaitResultContinue
				close(ch)
				delete(r.waiters, k)
			}
		}
	}

	if strings.HasPrefix(action.Action.Id, "wait") {
		message := strings.TrimPrefix(action.Action.Id, "wait ")
		var waitCmd tea.Cmd
		r.waiters[message], waitCmd = action.Action.Wait()
		return r, waitCmd
	}

	var cmds []tea.Cmd
	//cmds = append(cmds, action.Action.GetNext())
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
		return r.handleAndRouteAction(msg)
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

func (r Router) Open(scope actions.Scope, model tea.Model) (Router, tea.Cmd) {
	r.scopes = append(r.scopes, scope)
	r.Scope = scope
	r.Views[scope] = model
	return r, model.Init()
}
