package aliases

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/menu"
	"github.com/idursun/jjui/internal/ui/context"
)

type aliasItem struct {
	name        string
	command     []string
	needsParams bool
	paramCount  int
}

type paramState struct {
	paramIndex int
	params     []string
	prompt     string
}

func (i aliasItem) ShortCut() string {
	return ""
}

func (i aliasItem) FilterValue() string {
	return i.name
}

func (i aliasItem) Title() string {
	return i.name
}

func (i aliasItem) Description() string {
	cmdStr := strings.Join(i.command, " ")
	if i.needsParams {
		return fmt.Sprintf("%s (needs %d parameter(s))", cmdStr, i.paramCount)
	}
	return cmdStr
}

type Model struct {
	context      *context.MainContext
	keymap       config.KeyMappings[key.Binding]
	menu         menu.Menu
	help         help.Model
	paramMode    bool
	textInput    textinput.Model
	selectedItem aliasItem
	paramState   paramState
	width        int
	height       int
}

func (m *Model) Width() int {
	return m.width
}

func (m *Model) Height() int {
	return m.height
}

func (m *Model) SetWidth(w int) {
	m.width = w
	m.menu.SetWidth(w)
}

func (m *Model) SetHeight(h int) {
	m.height = h
	m.menu.SetHeight(h)
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.paramMode {
		return m.updateParamMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.menu.List.SettingFilter() {
			break
		}
		switch {
		case key.Matches(msg, m.keymap.Apply):
			if item, ok := m.menu.List.SelectedItem().(aliasItem); ok {
				return m.executeAlias(item)
			}
		case key.Matches(msg, m.keymap.Cancel):
			if m.menu.Filter != "" || m.menu.List.IsFiltered() {
				m.menu.List.ResetFilter()
				return m, m.menu.Filtered("")
			}
			return m, common.Close
		}
	}
	var cmd tea.Cmd
	m.menu.List, cmd = m.menu.List.Update(msg)
	return m, cmd
}

func (m *Model) updateParamMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Apply):
			// Store current parameter
			m.paramState.params[m.paramState.paramIndex] = strings.TrimSpace(m.textInput.Value())
			m.paramState.paramIndex++

			// Check if we need more parameters
			if m.paramState.paramIndex < m.selectedItem.paramCount {
				m.paramState.prompt = fmt.Sprintf("param%d:", m.paramState.paramIndex+1)
				m.textInput.Reset()
				prompt := m.paramState.prompt
				if m.selectedItem.paramCount > 1 {
					prompt += fmt.Sprintf(" (%d/%d)", m.paramState.paramIndex+1, m.selectedItem.paramCount)
				}
				m.textInput.Prompt = prompt + " "
				return m, nil
			}

			// All parameters collected, execute
			return m.executeAliasWithParams()
		case key.Matches(msg, m.keymap.Cancel):
			m.paramMode = false
			m.textInput.Reset()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) executeAlias(item aliasItem) (tea.Model, tea.Cmd) {
	if item.needsParams {
		m.selectedItem = item
		m.paramMode = true
		m.paramState = paramState{
			paramIndex: 0,
			params:     make([]string, item.paramCount),
			prompt:     "param1:",
		}
		m.textInput.Reset()
		prompt := m.paramState.prompt
		if m.selectedItem.paramCount > 1 {
			prompt += fmt.Sprintf(" (%d/%d)", m.paramState.paramIndex+1, m.selectedItem.paramCount)
		}
		m.textInput.Prompt = prompt + " "
		m.textInput.Placeholder = ""
		m.textInput.Focus()
		return m, textinput.Blink
	}

	// Build the command
	cmd := m.buildCommand(item, nil)
	return m, tea.Batch(func() tea.Msg {
		return common.ExecMsg{
			Line: cmd,
			Mode: common.ExecJJ,
		}
	}, common.Close)
}

func (m *Model) executeAliasWithParams() (tea.Model, tea.Cmd) {
	cmd := m.buildCommand(m.selectedItem, m.paramState.params)
	m.paramMode = false
	m.textInput.Reset()
	return m, tea.Batch(func() tea.Msg {
		return common.ExecMsg{
			Line: cmd,
			Mode: common.ExecJJ,
		}
	}, common.Close)
}

func (m *Model) buildCommand(item aliasItem, params []string) string {
	cmd := make([]string, len(item.command))
	copy(cmd, item.command)

	// Replace positional parameters ($1, $2, etc.)
	for i, part := range cmd {
		for j := 1; j <= len(params); j++ {
			placeholder := fmt.Sprintf("$%d", j)
			if strings.Contains(part, placeholder) {
				cmd[i] = strings.ReplaceAll(part, placeholder, params[j-1])
			}
		}
	}

	return strings.Join(cmd, " ")
}

func (m *Model) View() string {
	if m.paramMode {
		// Show menu with parameter input below
		baseMenu := m.menu.View(nil)
		inputLine := m.renderParameterInput()
		return baseMenu + "\n" + inputLine
	}
	return m.menu.View(nil)
}

func (m *Model) renderParameterInput() string {
	// Create parameter input line similar to filter
	promptStyle := common.DefaultPalette.Get("menu matched")
	textStyle := common.DefaultPalette.Get("menu text")

	// Build the input line
	inputLine := promptStyle.Render(m.textInput.Prompt) + textStyle.Render(m.textInput.Value())
	if m.textInput.Focused() {
		cursor := promptStyle.Render("█")
		inputLine += cursor
	}

	// Add padding to the input line
	return textStyle.PaddingLeft(1).Render(inputLine)
}

// analyzeAlias determines if an alias needs parameters and how many
func analyzeAlias(command []string) (bool, int) {
	maxParam := 0

	for _, part := range command {
		// Look for $1, $2, etc.
		for i := 1; i <= 10; i++ { // reasonable limit
			placeholder := fmt.Sprintf("$%d", i)
			if strings.Contains(part, placeholder) {
				if i > maxParam {
					maxParam = i
				}
			}
		}
	}

	return maxParam > 0, maxParam
}

func NewModel(ctx *context.MainContext, width int, height int) *Model {
	var items []list.Item

	// Get aliases from jj config and sort them lexicographically
	var aliasNames []string
	for name := range ctx.JJConfig.Aliases {
		aliasNames = append(aliasNames, name)
	}

	// Sort alias names lexicographically
	sort.Strings(aliasNames)

	// Create items in sorted order
	for _, name := range aliasNames {
		command := ctx.JJConfig.Aliases[name]
		if len(command) == 0 {
			continue
		}

		needsParams, paramCount := analyzeAlias(command)

		items = append(items, aliasItem{
			name:        name,
			command:     command,
			needsParams: needsParams,
			paramCount:  paramCount,
		})
	}

	keyMap := config.Current.GetKeyMap()
	menu := menu.NewMenu(items, width, height, keyMap, menu.WithStylePrefix("menu"))
	menu.Title = "JJ Aliases"
	menu.ShowShortcuts(false)
	menu.FilterMatches = func(i list.Item, filter string) bool {
		return strings.Contains(strings.ToLower(i.FilterValue()), strings.ToLower(filter))
	}

	textInput := textinput.New()
	textInput.Width = width - 8
	textInput.Prompt = "param1: "

	m := &Model{
		context:   ctx,
		keymap:    keyMap,
		menu:      menu,
		help:      help.New(),
		textInput: textInput,
		width:     width,
		height:    height,
	}
	m.SetWidth(width)
	m.SetHeight(height)
	return m
}
