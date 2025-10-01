package view

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type IHasActionMap interface {
	GetActionMap() map[string]actions.Action
}

var _ common.ContextProvider = (*Router)(nil)

type Router struct {
	context *context.MainContext
	scopes  []Scope
	Scope   Scope
	Views   map[Scope]tea.Model
}

func NewRouter(ctx *context.MainContext, scope Scope) Router {
	return Router{
		context: ctx,
		scopes:  []Scope{scope},
		Scope:   scope,
		Views:   make(map[Scope]tea.Model),
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
	log.Println("handling action:", action.Action.Id)
	if strings.HasPrefix(action.Action.Id, "close ") {
		viewId := strings.TrimPrefix(action.Action.Id, "close ")
		if _, ok := r.Views[Scope(viewId)]; ok {
			delete(r.Views, Scope(viewId))
			r.scopes = r.scopes[:len(r.scopes)-1]
			r.Scope = r.scopes[len(r.scopes)-1]
		}
	}

	if strings.HasPrefix(action.Action.Id, "switch ") {
		viewId := Scope(strings.TrimPrefix(action.Action.Id, "switch "))
		if _, ok := r.Views[viewId]; ok && r.Scope != viewId {
			r.Scope = viewId
		}
	}

	if strings.HasPrefix(action.Action.Id, "wait ") {
		log.Println("Waiting for action:", action.Action.Id)
		message := strings.TrimPrefix(action.Action.Id, "wait ")
		return r, r.context.AddWaiter(message, action.Action)
	}

	var cmds []tea.Cmd
	for k := range r.Views {
		var cmd tea.Cmd
		r.Views[k], cmd = r.Views[k].Update(action)
		cmds = append(cmds, cmd)
	}

	r.context.ContinueAction(action.Action.Id)

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

func (r Router) Open(scope Scope, model tea.Model) (Router, tea.Cmd) {
	r.scopes = append(r.scopes, scope)
	r.Scope = scope
	r.Views[scope] = model
	return r, model.Init()
}
