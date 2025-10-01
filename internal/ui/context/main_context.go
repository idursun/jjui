package context

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
)

var _ common.ContextProvider = (*MainContext)(nil)

type Waiter struct {
	waitChannel actions.WaitChannel
	action      actions.Action
}

type MainContext struct {
	CommandRunner
	Location      string
	JJConfig      *config.JJConfig
	DefaultRevset string
	CurrentRevset string
	Histories     *config.Histories
	ReadFn        func(value string) string
	variables     map[string]string
	waiters       map[string][]Waiter
}

func (ctx *MainContext) AddWaiter(event string, action actions.Action) tea.Cmd {
	if ctx.waiters == nil {
		ctx.waiters = make(map[string][]Waiter)
	}
	if waiters, ok := ctx.waiters[event]; ok {
		for _, waiter := range waiters {
			if waiter.action.Id == action.Id {
				log.Println("Waiter already exists for action:", action.Id)
				return nil
			}
		}
	}
	waitChannel, cmd := action.Wait()
	ctx.waiters[event] = append(ctx.waiters[event], Waiter{waitChannel, action})
	return cmd
}

func (ctx *MainContext) ContinueAction(actionId string) {
	if len(ctx.waiters) > 0 {
		for k, waiters := range ctx.waiters {
			if k == actionId {
				delete(ctx.waiters, k)

				for _, waiter := range waiters {
					log.Println("Continuing action:", actionId)
					waiter.waitChannel <- actions.WaitResultContinue
					close(waiter.waitChannel)
				}
			}
		}
	}
}

func (ctx *MainContext) Set(key string, value string) {
	ctx.variables[key] = value
}

func (ctx *MainContext) Read(value string) string {
	if ctx.ReadFn != nil {
		return ctx.ReadFn(value)
	}
	if v, ok := ctx.variables[value]; ok {
		return v
	}
	return value
}

func (ctx *MainContext) GetVariables() map[string]string {
	return ctx.variables
}

func (ctx *MainContext) ReplaceWithVariables(input string) string {
	for k, v := range ctx.variables {
		input = strings.ReplaceAll(input, k, v)
	}
	return input
}

func NewAppContext(location string) *MainContext {
	m := &MainContext{
		CommandRunner: &MainCommandRunner{
			Location: location,
		},
		Location:  location,
		Histories: config.NewHistories(),
		variables: make(map[string]string),
		waiters:   make(map[string][]Waiter),
	}

	m.JJConfig = &config.JJConfig{}
	if output, err := m.RunCommandImmediate(jj.ConfigListAll()); err == nil {
		m.JJConfig, _ = config.DefaultConfig(output)
	}
	return m
}
