package actions

import (
	"errors"
	"log"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
)

var Registry = make(map[string]Action)

type WaitResult int

const (
	WaitResultContinue WaitResult = iota
	WaitResultCancel
)

type WaitChannel chan WaitResult

type Action struct {
	Id   string         `toml:"id"`
	Desc string         `toml:"desc,omitempty"`
	Args map[string]any `toml:"args,omitempty"`
	Next []Action       `toml:"next,omitempty"`
}

func (a *Action) UnmarshalTOML(data any) error {
	switch value := data.(type) {
	case string:
		a.Id = value
	case map[string]interface{}:
		if id, ok := value["id"]; ok {
			a.Id = id.(string)
		}

		if desc, ok := value["desc"]; ok {
			a.Desc = desc.(string)
		}

		if wait, ok := value["wait"]; ok {
			if message, ok := wait.(string); ok {
				a.Id = "wait " + message
			}
		}

		if jjRunCommandArgs, ok := value["jj"]; ok {
			a.Id = "run"
			if args, ok := jjRunCommandArgs.([]interface{}); ok {
				a.Args = map[string]any{
					"jj": args,
				}
			} else {
				return errors.New("jj arguments needs to an array")
			}
		}

		if next, ok := value["next"]; ok {
			a.Next = []Action{}
			for _, v := range next.([]interface{}) {
				newAction := Action{}
				newAction.UnmarshalTOML(v)
				a.Next = append(a.Next, newAction)
			}
		}

		if args, ok := value["args"]; ok {
			a.Args = args.(map[string]interface{})
		}

		// Implicit args
		if a.Args == nil {
			a.Args = make(map[string]any)
		}
		for k, v := range value {
			if k != "id" && k != "next" && k != "args" && k != "wait" && k != "jj" && k != "desc" {
				a.Args[k] = v
			}
		}
	}
	return nil
}

func (a Action) GetNextCmd() tea.Cmd {
	if len(a.Next) == 0 {
		return nil
	}
	nextAction := a.Next[0]
	if len(nextAction.Next) > 0 {
		a.Next = a.Next[1:]
		return tea.Sequence(InvokeAction(nextAction), a.GetNextCmd())
	}
	nextAction.Next = a.Next[1:]
	return InvokeAction(nextAction)
}

func (a Action) GetNextMsg() tea.Msg {
	if len(a.Next) == 0 {
		return nil
	}
	nextAction := a.Next[0]
	nextAction.Next = a.Next[1:]
	return InvokeActionMsg{Action: nextAction}
}

var ChannelCount atomic.Int32

func (a Action) Wait() (WaitChannel, tea.Cmd) {
	ch := make(WaitChannel, 1)
	ChannelCount.Add(1)
	return ch, func() tea.Msg {
		select {
		case <-ch:
			ChannelCount.Add(-1)
			if len(a.Next) > 0 {
				log.Printf("Continuing action chain for %s", a.Id)
				return a.GetNextMsg()
			}
			return nil
		}
	}
}

func (a Action) Get(name string, defaultValue any) any {
	if a.Args == nil {
		return defaultValue
	}
	if v, ok := a.Args[name]; ok {
		return v
	}
	return defaultValue
}

func (a Action) GetArgs(name string) []string {
	if a.Args == nil {
		return []string{}
	}
	if v, ok := a.Args[name]; ok {
		if args, ok := v.([]any); ok {
			result := make([]string, len(args))
			for i, arg := range args {
				result[i] = arg.(string)
			}
			return result
		}
		if args, ok := v.([]string); ok {
			return args
		}
	}
	return []string{}
}

func InvokeAction(action Action) tea.Cmd {
	if existing, ok := Registry[action.Id]; ok {
		action = existing
	}

	return func() tea.Msg {
		return InvokeActionMsg{Action: action}
	}
}

type InvokeActionMsg struct {
	Action Action
}
