package context

import (
	"log"
	"strings"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
)

var _ common.ContextProvider = (*MainContext)(nil)

type MainContext struct {
	CommandRunner
	Location      string
	Leader        LeaderMap
	JJConfig      *config.JJConfig
	DefaultRevset string
	CurrentRevset string
	Histories     *config.Histories
	ReadFn        func(value string) string
	variables     map[string]string
	Waiters       map[string]actions.WaitChannel
}

func (ctx *MainContext) ContinueAction(actionId string) {
	if len(ctx.Waiters) > 0 {
		for k, ch := range ctx.Waiters {
			if k == actionId {
				log.Println("Continuing action:", actionId)
				ch <- actions.WaitResultContinue
				close(ch)
				delete(ctx.Waiters, k)
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
		Waiters:   make(map[string]actions.WaitChannel),
	}

	m.JJConfig = &config.JJConfig{}
	if output, err := m.RunCommandImmediate(jj.ConfigListAll()); err == nil {
		m.JJConfig, _ = config.DefaultConfig(output)
	}
	return m
}
