package helppage

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	customcommands "github.com/idursun/jjui/internal/ui/custom_commands"
)

type Model struct {
	width   int
	height  int
	keyMap  config.KeyMappings[key.Binding]
	context context.AppContext
}

func (h *Model) Width() int {
	return h.width
}

func (h *Model) Height() int {
	return h.height
}

func (h *Model) SetWidth(w int) {
	h.width = w
}

func (h *Model) SetHeight(height int) {
	h.height = height
}

func (h *Model) ShortHelp() []key.Binding {
	return []key.Binding{h.keyMap.Help, h.keyMap.Cancel}
}

func (h *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{h.ShortHelp()}
}

func (h *Model) Init() tea.Cmd {
	return nil
}

func (h *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, h.keyMap.Help), key.Matches(msg, h.keyMap.Cancel):
			return h, common.Close
		}
	}
	return h, nil
}

var (
	keyStyle  = common.DefaultPalette.ChangeId
	descStyle = common.DefaultPalette.Dimmed
)

func printHelp(k key.Binding) string {
	return printHelpExt(k.Help().Key, k.Help().Desc)
}

func printHelpExt(key string, desc string) string {
	keyAligned := fmt.Sprintf("%9s", key)
	help := fmt.Sprintf("%s %s", keyStyle.Render(keyAligned), descStyle.Render(desc))
	return help
}

func printHeader(header string) string {
	return common.DefaultPalette.EmptyPlaceholder.Render(header)
}

func printMode(key key.Binding, name string) string {
	keyAligned := fmt.Sprintf("%9s", key.Help().Key)
	help := fmt.Sprintf("%v %s", keyStyle.Render(keyAligned), common.DefaultPalette.EmptyPlaceholder.Render(name))
	return help
}

var border = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(2)

func (h *Model) View() string {
	leftView := lipgloss.JoinVertical(lipgloss.Left,
		printHeader("UI"),
		printHelp(h.keyMap.Refresh),
		printHelp(h.keyMap.Help),
		printHelp(h.keyMap.Cancel),
		printHelp(h.keyMap.Quit),
		printHelp(h.keyMap.Revset),
		printHeader("Revisions"),
		printHelp(h.keyMap.ToggleSelect),
		printHelp(h.keyMap.QuickSearch),
		printHelp(h.keyMap.QuickSearchCycle),
		printHelp(h.keyMap.New),
		printHelp(h.keyMap.Commit),
		printHelp(h.keyMap.Describe),
		printHelp(h.keyMap.Edit),
		printHelp(h.keyMap.Diff),
		printHelp(h.keyMap.Diffedit),
		printHelp(h.keyMap.Split),
		printHelp(h.keyMap.Squash),
		printHelp(h.keyMap.Abandon),
		printHelp(h.keyMap.Absorb),
		printHelp(h.keyMap.Undo),
		printHelp(h.keyMap.Details.Mode),
		printHelp(h.keyMap.Evolog),
		printHelp(h.keyMap.Bookmark.Set),
		"",
		printMode(h.keyMap.Preview.Mode, "Preview"),
		printHelp(h.keyMap.Preview.ScrollUp),
		printHelp(h.keyMap.Preview.ScrollDown),
		printHelp(h.keyMap.Preview.HalfPageDown),
		printHelp(h.keyMap.Preview.HalfPageUp),
		printHelp(h.keyMap.Preview.Expand),
		printHelp(h.keyMap.Preview.Shrink),
	)

	rightView := lipgloss.JoinVertical(lipgloss.Left,
		printMode(h.keyMap.Details.Mode, "Details"),
		printHelp(h.keyMap.Details.Close),
		printHelp(h.keyMap.Details.ToggleSelect),
		printHelp(h.keyMap.Details.Restore),
		printHelp(h.keyMap.Details.Split),
		printHelp(h.keyMap.Details.Diff),
		printHelp(h.keyMap.Details.RevisionsChangingFile),
		"",
		printMode(h.keyMap.Git.Mode, "Git"),
		printHelp(h.keyMap.Git.PushFilter),
		printHelp(h.keyMap.Git.FetchFilter),
		printHelp(h.keyMap.Git.PushRevset),
		printHelp(h.keyMap.Git.PushAll),
		printHelp(h.keyMap.Git.Fetch),
		"",
		printMode(h.keyMap.Bookmark.Mode, "Bookmarks"),
		printHelp(h.keyMap.Bookmark.Move),
		printHelp(h.keyMap.Bookmark.Delete),
		printHelp(h.keyMap.Bookmark.Untrack),
		printHelp(h.keyMap.Bookmark.Track),
		printHelp(h.keyMap.Bookmark.Forget),
		"",
		printMode(h.keyMap.Rebase.Mode, "Rebase"),
		printHelp(h.keyMap.Rebase.Revision),
		printHelp(h.keyMap.Rebase.Source),
		printHelp(h.keyMap.Rebase.Branch),
		printHelp(h.keyMap.Rebase.Before),
		printHelp(h.keyMap.Rebase.After),
		printHelp(h.keyMap.Rebase.Onto),
		printHelp(h.keyMap.Rebase.Insert),
		printHelp(h.keyMap.Apply),
		"",
		printMode(h.keyMap.OpLog.Mode, "Oplog"),
		printHelp(h.keyMap.Diff),
		printHelp(h.keyMap.OpLog.Restore),
		printMode(h.keyMap.CustomCommands, "Custom Commands"),
	)

	var customCommands []string
	for command := range customcommands.GetCommandManager().Iter() {
		if command.Key.Enabled() {
			customCommands = append(customCommands, printHelp(command.Key))
		}
	}
	if len(customCommands) > 0 {
		rightView = lipgloss.JoinVertical(lipgloss.Left,
			rightView,
			lipgloss.JoinVertical(lipgloss.Left, customCommands...),
		)
	}

	content := lipgloss.JoinHorizontal(lipgloss.Left, leftView, "  ", rightView)

	return border.Render(content)
}

func New(context context.AppContext) *Model {
	keyMap := context.KeyMap()
	return &Model{
		context: context,
		keyMap:  keyMap,
	}
}
