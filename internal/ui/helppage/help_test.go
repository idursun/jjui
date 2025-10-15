package helppage_test

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/helppage"
	"github.com/idursun/jjui/test"
)

func TestHelpMenuTriggeredFromMainUI(t *testing.T) {
	origConfig := *config.Current
	defer func() {
		*config.Current = origConfig
	}()

	config.Current.Revisions.LogBatching = false
	config.Current.Limit = 0

	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.CustomCommands = map[string]appContext.CustomCommand{}
	ctx.JJConfig = &config.JJConfig{}
	ctx.Histories = config.NewHistories()
	ctx.DefaultRevset = "@"
	ctx.CurrentRevset = "@"

	model := ui.New(ctx)

	var cmd tea.Cmd
	model, cmd = model.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	model = applyCmds(model, cmd)

	beforeView := model.View()
	if strings.Contains(beforeView, "Search: ") {
		t.Fatalf("expected main view to not include help search prompt before toggle")
	}

	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	model = applyCmds(model, cmd)

	afterView := model.View()
	if !strings.Contains(afterView, "Search: ") {
		t.Fatalf("expected help overlay to include search prompt, view: %q", afterView)
	}
	if !strings.Contains(afterView, "jump to parent/child/working-copy") {
		t.Fatalf("expected help overlay content to be rendered, view: %q", afterView)
	}
}

func TestHelpMenuLayoutStaysFixedWhileFiltering(t *testing.T) {
	origConfig := *config.Current
	defer func() {
		*config.Current = origConfig
	}()

	config.Current.Revisions.LogBatching = false

	ctx := &appContext.MainContext{
		CustomCommands: map[string]appContext.CustomCommand{},
	}

	model := helppage.New(ctx)
	model.SetWidth(90)
	model.SetHeight(32)

	defaultView := model.View()
	defaultWidth := lipgloss.Width(defaultView)
	defaultHeight := lipgloss.Height(defaultView)

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	filteredView := model.View()
	filteredWidth := lipgloss.Width(filteredView)
	filteredHeight := lipgloss.Height(filteredView)

	if filteredWidth != defaultWidth {
		t.Fatalf("expected filtered view width to remain %d, got %d", defaultWidth, filteredWidth)
	}
	if filteredHeight != defaultHeight {
		t.Fatalf("expected filtered view height to remain %d, got %d", defaultHeight, filteredHeight)
	}
}

func TestHelpModelHelpBindings(t *testing.T) {
	ctx := &appContext.MainContext{
		CustomCommands: map[string]appContext.CustomCommand{},
	}
	model := helppage.New(ctx)

	short := model.ShortHelp()
	if len(short) != 2 {
		t.Fatalf("expected short help to contain 2 bindings, got %d", len(short))
	}

	full := model.FullHelp()
	if len(full) != 1 {
		t.Fatalf("expected full help to contain a single row, got %d", len(full))
	}
	if !reflect.DeepEqual(full[0], short) {
		t.Fatalf("expected full help row to mirror short help bindings")
	}
}

func TestHelpModelCloseCommands(t *testing.T) {
	ctx := &appContext.MainContext{
		CustomCommands: map[string]appContext.CustomCommand{},
	}

	testCases := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{
			name: "help binding",
			msg: tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{'?'},
			},
		},
		{
			name: "cancel binding",
			msg: tea.KeyMsg{
				Type: tea.KeyEsc,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := helppage.New(ctx)
			_, cmd := model.Update(tc.msg)
			if cmd == nil {
				t.Fatalf("expected non-nil command when %s pressed", tc.name)
			}

			msg := cmd()
			if _, ok := msg.(common.CloseViewMsg); !ok {
				t.Fatalf("expected CloseViewMsg when %s pressed, got %T", tc.name, msg)
			}
		})
	}
}

func applyCmds(model tea.Model, cmd tea.Cmd) tea.Model {
	if cmd == nil {
		return model
	}
	msgs := collectMsgs(cmd)
	for _, msg := range msgs {
		var nextCmd tea.Cmd
		model, nextCmd = model.Update(msg)
		model = applyCmds(model, nextCmd)
	}
	return model
}

func collectMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}

	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, collectMsgs(c)...)
		}
		return out
	}

	val := reflect.ValueOf(msg)
	if val.Kind() == reflect.Slice && val.Type().Elem() == reflect.TypeOf((tea.Cmd)(nil)) {
		var out []tea.Msg
		for i := 0; i < val.Len(); i++ {
			out = append(out, collectMsgs(val.Index(i).Interface().(tea.Cmd))...)
		}
		return out
	}

	return []tea.Msg{msg}
}
