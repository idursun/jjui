package actions

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var Registry = make(map[string]Action)

type WaitResult int

const (
	WaitResultContinue WaitResult = iota
	WaitResultCancel
)

type WaitChannel chan WaitResult

type Scope string

const (
	ScopeNone      Scope = ""
	ScopeList      Scope = "list"
	ScopeRevisions Scope = "revisions"
	ScopeOplog     Scope = "oplog"
	ScopeDiff      Scope = "diff"
	ScopeRevset    Scope = "revset"
	ScopePreview   Scope = "preview"
	ScopeUndo      Scope = "undo"
	ScopeBookmarks Scope = "bookmarks"
)

type Action struct {
	Id      string         `toml:"id"`
	Args    map[string]any `toml:"args,omitempty"`
	Next    []Action       `toml:"next,omitempty"`
	Output  string         `toml:"output,omitempty"`
	Outputs map[string]string
}

func (a *Action) UnmarshalTOML(data any) error {
	switch value := data.(type) {
	case string:
		a.Id = value
	case map[string]interface{}:
		if id, ok := value["id"]; ok {
			a.Id = id.(string)
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

		if output, ok := value["output"]; ok {
			a.Output = output.(string)
		}
	}
	return nil
}

func (a Action) Process(input string) string {
	if a.Outputs == nil {
		return input
	}
	for k, v := range a.Outputs {
		input = strings.ReplaceAll(input, k, v)
	}
	return input
}

func (a Action) GetNext() tea.Cmd {
	if len(a.Next) == 0 {
		return nil
	}
	nextAction := a.Next[0]
	nextAction.Next = a.Next[1:]
	nextAction.Outputs = a.Outputs
	return InvokeAction(nextAction)
}

func (a Action) Wait() (WaitChannel, tea.Cmd) {
	ch := make(WaitChannel, 1)
	return ch, func() tea.Msg {
		select {
		case <-ch:
			nextAction := a.Next[0]
			nextAction.Next = a.Next[1:]
			nextAction.Outputs = a.Outputs
			return InvokeActionMsg{Action: nextAction}
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
