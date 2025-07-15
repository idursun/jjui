package helppage

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type Model struct {
	width   int
	height  int
	keyMap  config.KeyMappings[key.Binding]
	context *context.MainContext
	styles  styles
}
type styles struct {
	border   lipgloss.Style
	title    lipgloss.Style
	help     lipgloss.Style
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
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

func (h *Model) printHelp(k key.Binding) string {
	return h.printHelpExt(k.Help().Key, k.Help().Desc)
}

func (h *Model) printHelpExt(key string, desc string) string {
	keyAligned := fmt.Sprintf("%9s", key)
	help := fmt.Sprintf("%s %s", h.styles.shortcut.Render(keyAligned), h.styles.dimmed.Render(desc))
	return help
}

func (h *Model) printHeader(header string) string {
	return h.printMode(key.NewBinding(), header)
}

func (h *Model) printMode(key key.Binding, name string) string {
	keyAligned := fmt.Sprintf("%9s", key.Help().Key)
	help := fmt.Sprintf("%v %s", h.styles.shortcut.Render(keyAligned), h.styles.title.Render(name))
	return help
}

var border = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(2)

func (h *Model) View() string {
	leftView := lipgloss.JoinVertical(lipgloss.Left,
		h.printHeader("UI"),
		h.printHelp(h.keyMap.Refresh),
		h.printHelp(h.keyMap.Help),
		h.printHelp(h.keyMap.Cancel),
		h.printHelp(h.keyMap.Quit),
		h.printHelp(h.keyMap.Suspend),
		h.printHelp(h.keyMap.Revset),
		h.printHeader("Revisions"),
		h.printHelp(h.keyMap.JumpToParent),
		h.printHelp(h.keyMap.JumpToWorkingCopy),
		h.printHelp(h.keyMap.ToggleSelect),
		h.printHelp(h.keyMap.QuickSearch),
		h.printHelp(h.keyMap.QuickSearchCycle),
		h.printHelp(h.keyMap.New),
		h.printHelp(h.keyMap.Commit),
		h.printHelp(h.keyMap.Describe),
		h.printHelp(h.keyMap.Edit),
		h.printHelp(h.keyMap.Diff),
		h.printHelp(h.keyMap.Diffedit),
		h.printHelp(h.keyMap.Split),
		h.printHelp(h.keyMap.Abandon),
		h.printHelp(h.keyMap.Absorb),
		h.printHelp(h.keyMap.Undo),
		h.printHelp(h.keyMap.Details.Mode),
		h.printHelp(h.keyMap.Evolog),
		h.printHelp(h.keyMap.Bookmark.Set),
		h.printHelp(h.keyMap.InlineDescribe.Mode),
	)

	middleView := lipgloss.JoinVertical(lipgloss.Left,
		h.printMode(h.keyMap.Preview.Mode, "Preview"),
		h.printHelp(h.keyMap.Preview.ScrollUp),
		h.printHelp(h.keyMap.Preview.ScrollDown),
		h.printHelp(h.keyMap.Preview.HalfPageDown),
		h.printHelp(h.keyMap.Preview.HalfPageUp),
		h.printHelp(h.keyMap.Preview.Expand),
		h.printHelp(h.keyMap.Preview.Shrink),
		"",
		h.printMode(h.keyMap.Details.Mode, "Details"),
		h.printHelp(h.keyMap.Details.Close),
		h.printHelp(h.keyMap.Details.ToggleSelect),
		h.printHelp(h.keyMap.Details.Restore),
		h.printHelp(h.keyMap.Details.Split),
		h.printHelp(h.keyMap.Details.Diff),
		h.printHelp(h.keyMap.Details.RevisionsChangingFile),
		"",
		h.printMode(h.keyMap.Git.Mode, "Git"),
		h.printHelp(h.keyMap.Git.Push),
		h.printHelp(h.keyMap.Git.Fetch),
		"",
		h.printMode(h.keyMap.Bookmark.Mode, "Bookmarks"),
		h.printHelp(h.keyMap.Bookmark.Move),
		h.printHelp(h.keyMap.Bookmark.Delete),
		h.printHelp(h.keyMap.Bookmark.Untrack),
		h.printHelp(h.keyMap.Bookmark.Track),
		h.printHelp(h.keyMap.Bookmark.Forget),
	)

	rightView := lipgloss.JoinVertical(lipgloss.Left,
		h.printMode(h.keyMap.Squash.Mode, "Squash"),
		h.printHelp(h.keyMap.Squash.KeepEmptied),
		h.printHelp(h.keyMap.Squash.Interactive),
		"",
		h.printMode(h.keyMap.Rebase.Mode, "Rebase"),
		h.printHelp(h.keyMap.Rebase.Revision),
		h.printHelp(h.keyMap.Rebase.Source),
		h.printHelp(h.keyMap.Rebase.Branch),
		h.printHelp(h.keyMap.Rebase.Before),
		h.printHelp(h.keyMap.Rebase.After),
		h.printHelp(h.keyMap.Rebase.Onto),
		h.printHelp(h.keyMap.Rebase.Insert),
		"",
		h.printMode(h.keyMap.Duplicate.Mode, "Duplicate"),
		h.printHelp(h.keyMap.Duplicate.Onto),
		h.printHelp(h.keyMap.Duplicate.Before),
		h.printHelp(h.keyMap.Duplicate.After),
		"",
		h.printMode(h.keyMap.OpLog.Mode, "Oplog"),
		h.printHelp(h.keyMap.Diff),
		h.printHelp(h.keyMap.OpLog.Restore),
		h.printMode(h.keyMap.Leader, "Leader"),
		h.printMode(h.keyMap.CustomCommands, "Custom Commands"),
	)

	var customCommands []string
	for _, command := range h.context.CustomCommands {
		customCommands = append(customCommands, h.printHelp(command.Binding()))
	}

	if len(customCommands) > 0 {
		rightView = lipgloss.JoinVertical(lipgloss.Left,
			rightView,
			lipgloss.JoinVertical(lipgloss.Left, customCommands...),
		)
	}

	content := lipgloss.JoinHorizontal(lipgloss.Left, leftView, "  ", middleView, "  ", rightView)

	return border.Render(content)
}

func New(context *context.MainContext) *Model {
	styles := styles{
		title:    common.DefaultPalette.Get("help title"),
		dimmed:   common.DefaultPalette.Get("help dimmed"),
		border:   common.DefaultPalette.Get("help border"),
		help:     common.DefaultPalette.Get("help text"),
		shortcut: common.DefaultPalette.Get("help shortcut"),
	}
	return &Model{
		context: context,
		keyMap:  config.Current.GetKeyMap(),
		styles:  styles,
	}
}
