package confirmation

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
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

	cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd != nil {
		cmd()
	}
	assert.True(t, cmdCalled)
}

func TestLegacyAddOption(t *testing.T) {
	var cmdCalled bool
	testCmd := func() tea.Msg {
		cmdCalled = true
		return nil
	}

	model := New([]string{"Test message"}, WithOption("Yes", testCmd, key.NewBinding(key.WithKeys("y"))))

	assert.Equal(t, 1, len(model.options))
	assert.Equal(t, "Yes", model.options[0].label)

	cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd != nil {
		cmd()
	}
	assert.True(t, cmdCalled)
}
