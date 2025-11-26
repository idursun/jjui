package helppage_test

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"

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

	model := ui.NewUI(ctx)

	test.SimulateModel(model, func() tea.Msg {
		return tea.WindowSizeMsg{Width: 140, Height: 80}
	})

	beforeView := model.View()
	if strings.Contains(beforeView, "Search: ") {
		t.Fatalf("expected main view to not include help search prompt before toggle")
	}

	test.SimulateModel(model, test.Type("?"))

	afterView := model.View()
	if !strings.Contains(afterView, "Search: ") {
		t.Fatalf("expected help overlay to include search prompt, view: \n%q", afterView)
	}
	if !strings.Contains(afterView, "jump to parent/child/working-copy") {
		t.Fatalf("expected help overlay content to be rendered, view: \n%q", afterView)
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
	test.SimulateModel(model, model.Init())

	defaultView := model.View()
	defaultWidth := lipgloss.Width(defaultView)
	defaultHeight := lipgloss.Height(defaultView)

	_ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

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
	test.SimulateModel(model, model.Init())

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
		name        string
		interaction tea.Cmd
	}{
		{
			name:        "help binding",
			interaction: test.Type("?"),
		},
		{
			name:        "cancel binding",
			interaction: test.Press(tea.KeyEscape),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := helppage.New(ctx)
			test.SimulateModel(model, model.Init())
			var msgs []tea.Msg
			test.SimulateModel(model, tc.interaction, func(msg tea.Msg) {
				msgs = append(msgs, msg)
			})
			assert.Contains(t, msgs, common.CloseViewMsg{}, "expected CloseViewMsg when %s pressed", tc.name)
		})
	}
}
