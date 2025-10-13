package view

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
)

type IHasActionMap interface {
	GetActionMap() actions.ActionMap
}

type Waiter struct {
	waitChannel actions.WaitChannel
	action      actions.Action
}

var _ common.ContextProvider = (*Router)(nil)

type Router struct {
	scopes   []Scope
	Scope    Scope
	Views    map[Scope]tea.Model
	waiters  map[string][]Waiter
	previous []string
}

func NewRouter(scope Scope) *Router {
	return &Router{
		waiters: make(map[string][]Waiter),
		scopes:  []Scope{scope},
		Scope:   scope,
		Views:   make(map[Scope]tea.Model),
	}
}

func (r *Router) AddWaiter(event string, action actions.Action) tea.Cmd {
	if r.waiters == nil {
		r.waiters = make(map[string][]Waiter)
	}
	if waiters, ok := r.waiters[event]; ok {
		for _, waiter := range waiters {
			if waiter.action.Id == action.Id {
				log.Println("Waiter already exists for action:", action.Id)
				return nil
			}
		}
	}
	waitChannel, cmd := action.Wait()
	r.waiters[event] = append(r.waiters[event], Waiter{waitChannel, action})
	return cmd
}

func (r *Router) ContinueAction(actionId string) {
	if len(r.waiters) > 0 {
		for k, waiters := range r.waiters {
			if k == actionId {
				var remaining []Waiter
				for _, waiter := range waiters {
					if waiter.action.When == string(r.Scope) || waiter.action.When == "" {
						log.Println("Continuing action:", actionId)
						waiter.waitChannel <- actions.WaitResultContinue
						close(waiter.waitChannel)
					} else {
						remaining = append(remaining, waiter)
					}
				}
				if len(remaining) == 0 {
					delete(r.waiters, k)
				} else {
					r.waiters[k] = remaining
				}
			}
		}
	}
}

func (r *Router) Init() tea.Cmd {
	var cmds []tea.Cmd
	for k := range r.Views {
		cmds = append(cmds, r.Views[k].Init())
	}
	return tea.Batch(cmds...)
}

func (r *Router) handleAndRouteAction(action actions.InvokeActionMsg) (*Router, tea.Cmd) {
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
		return r, r.AddWaiter(message, action.Action)
	}

	var cmds []tea.Cmd
	for k := range r.Views {
		var cmd tea.Cmd
		r.Views[k], cmd = r.Views[k].Update(action)
		cmds = append(cmds, cmd)
	}

	r.ContinueAction(action.Action.Id)

	return r, tea.Batch(cmds...)
}

func (r *Router) Update(msg tea.Msg) (*Router, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		return r.handleAndRouteAction(msg)
	case tea.KeyMsg:
		var cmd tea.Cmd
		if currentView, ok := r.Views[r.Scope]; ok {
			if hasActionMap, ok := currentView.(IHasActionMap); ok {
				actionMap := hasActionMap.GetActionMap()
				currentKey := msg.String()
				matches := actionMap.GetMatch(r.previous, currentKey)
				if len(matches) == 0 && len(r.previous) > 0 {
					r.previous = nil
					return r, func() tea.Msg {
						// No matches, reset the previous keys
						return common.ShowAvailableBindingMatches{Matches: nil}
					}
				}
				if len(matches) > 1 {
					r.previous = append(r.previous, currentKey)
					return r, func() tea.Msg {
						return common.ShowAvailableBindingMatches{Matches: matches}
					}
				}
				if len(matches) == 1 {
					action := matches[0]
					r.previous = nil
					return r, actions.InvokeAction(action.Do)
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

func (r *Router) View() string {
	return ""
}

func (r *Router) Read(value string) string {
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

func (r *Router) Open(scope Scope, model tea.Model) (*Router, tea.Cmd) {
	r.scopes = append(r.scopes, scope)
	r.Scope = scope
	r.Views[scope] = model
	return r, model.Init()
}
