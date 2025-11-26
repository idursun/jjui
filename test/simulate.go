package test

import (
	"reflect"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
)

func SimulateModel[T interface {
	Update(tea.Msg) tea.Cmd
}](model T, first tea.Cmd, observers ...func(tea.Msg)) {
	drainCmds(first, func(msg tea.Msg) tea.Cmd {
		return model.Update(msg)
	}, observers...)
}

func Type(runes string) tea.Cmd {
	press := func(r rune) tea.Cmd {
		return func() tea.Msg {
			return tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{r},
			}
		}
	}
	var cmds []tea.Cmd
	for _, r := range runes {
		cmds = append(cmds, press(r))
	}
	return tea.Sequence(cmds...)
}

func Press(key tea.KeyType) tea.Cmd {
	return func() tea.Msg {
		return tea.KeyMsg{
			Type: key,
		}
	}
}
func drainCmds(first tea.Cmd, apply func(tea.Msg) tea.Cmd, observers ...func(tea.Msg)) {
	queue := []tea.Cmd{first}

	for len(queue) > 0 {
		var cmd tea.Cmd
		cmd, queue = queue[0], queue[1:]
		if cmd == nil {
			continue
		}
		msg := cmd()
		if msg == nil {
			continue
		}

		switch v := msg.(type) {
		case cursor.BlinkMsg:
			// Ignore cursor blink messages.
		case tea.BatchMsg: // Batch(...)
			queue = append(queue, v...)
			continue
		default:
			if slice, ok := asCmdSlice(msg); ok {
				queue = append(queue, slice...)
				continue
			}
			for _, observe := range observers {
				observe(v)
			}
			if next := apply(v); next != nil {
				queue = append(queue, next)
			}
		}
	}
}

var cmdType = reflect.TypeOf((tea.Cmd)(nil))

// asCmdSlice returns the contents if msg is any named slice whose elements are tea.Cmd.
func asCmdSlice(msg tea.Msg) ([]tea.Cmd, bool) {
	val := reflect.ValueOf(msg)
	if val.Kind() != reflect.Slice || !val.Type().Elem().AssignableTo(cmdType) {
		return nil, false
	}
	out := make([]tea.Cmd, val.Len())
	for i := 0; i < val.Len(); i++ {
		out[i] = val.Index(i).Interface().(tea.Cmd)
	}
	return out, true
}
