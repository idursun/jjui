package confirmation

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
)

const (
	White = "7"
	Red   = "1"
	Green = "2"
	Blue  = "4"
)

func TestConfirmationWithoutStylePrefix(t *testing.T) {
	palette := common.NewPalette()
	palette.Update(map[string]config.Color{
		"confirmation text":             {Fg: White},
		"confirmation selected":         {Fg: Green},
		"details confirmation text":     {Fg: Blue},
		"details confirmation selected": {Fg: Red},
	})

	originalPalette := common.DefaultPalette
	common.DefaultPalette = palette
	defer func() { common.DefaultPalette = originalPalette }()

	defaultModel := New([]string{"Test message"})
	assert.Equal(t, lipgloss.Color(White), defaultModel.Styles.Text.GetForeground())
	assert.Equal(t, lipgloss.Color(Green), defaultModel.Styles.Selected.GetForeground())
}

func TestConfirmationWithStylePrefix(t *testing.T) {
	palette := common.NewPalette()
	palette.Update(map[string]config.Color{
		"confirmation text":             {Fg: White},
		"confirmation selected":         {Fg: Green},
		"details confirmation text":     {Fg: Blue},
		"details confirmation selected": {Fg: Red},
	})

	originalPalette := common.DefaultPalette
	common.DefaultPalette = palette
	defer func() { common.DefaultPalette = originalPalette }()

	detailsModel := New(
		[]string{"Test message"},
		WithStylePrefix("details"),
	)

	assert.Equal(t, lipgloss.Color(Blue), detailsModel.Styles.Text.GetForeground())
	assert.Equal(t, lipgloss.Color(Red), detailsModel.Styles.Selected.GetForeground())
}

func TestConfirmationWithOption(t *testing.T) {
	var cmdCalled bool
	testCmd := func() tea.Msg {
		cmdCalled = true
		return nil
	}

	model := New(
		[]string{"Test message"},
		WithOption("Yes", testCmd, key.NewBinding(key.WithKeys("y"))),
		WithOption("No", nil, key.NewBinding(key.WithKeys("n"))),
	)

	assert.Equal(t, 2, len(model.options))
	assert.Equal(t, "Yes", model.options[0].label)
	assert.Equal(t, "No", model.options[1].label)

	cmd := model.Update(intents.Apply{})
	if cmd != nil {
		cmd()
	}
	assert.True(t, cmdCalled)
}

func TestDispatcherIntentFlow(t *testing.T) {
	var selected string
	model := New(
		[]string{"Test message"},
		WithOption("Yes", func() tea.Msg {
			selected = "yes"
			return nil
		}, key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		WithAltOption("No", func() tea.Msg {
			selected = "no"
			return nil
		}, func() tea.Msg {
			selected = "no-alt"
			return nil
		}, key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
	)

	_ = model.Update(intents.Navigate{Delta: 1})
	cmd := model.Update(intents.Apply{Force: true})
	if cmd != nil {
		cmd()
	}
	assert.Equal(t, "no-alt", selected)

	cmd = model.Update(intents.Cancel{})
	if cmd != nil {
		cmd()
	}
	assert.Equal(t, "no", selected)
}

func TestViewRect_DefaultRendersAtZBase(t *testing.T) {
	model := New([]string{"Test message"})

	dl := render.NewDisplayContext()
	box := layout.Box{R: cellbuf.Rect(0, 0, 50, 20)}
	model.ViewRect(dl, box)

	draws := dl.DrawList()
	assert.NotEmpty(t, draws, "Expected confirmation to produce draw operations")

	for _, draw := range draws {
		assert.Less(t, draw.Z, render.ZPreview,
			"Default confirmation should render below preview (ZBase level)")
	}
}

func TestWithZIndex_RendersAtSpecifiedZIndex(t *testing.T) {
	model := New([]string{"Test message"}, WithZIndex(render.ZDialogs))

	dl := render.NewDisplayContext()
	box := layout.Box{R: cellbuf.Rect(0, 0, 50, 20)}
	model.ViewRect(dl, box)

	draws := dl.DrawList()
	assert.NotEmpty(t, draws, "Expected confirmation to produce draw operations")

	for _, draw := range draws {
		assert.GreaterOrEqual(t, draw.Z, render.ZDialogs,
			"Confirmation with WithZIndex(ZDialogs) should render above preview")
	}
}
