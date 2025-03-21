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
	width  int
	height int
	keyMap config.KeyMappings[key.Binding]
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
		"",
		printHeader("Preview"),
		printHelpExt("tab", "focus preview"),
		printHelpExt("ctrl+p", "line up"),
		printHelpExt("ctrl+n", "line down"),
		printHelpExt("ctrl+d", "half page down"),
		printHelpExt("ctrl+u", "half page up"),
		"",
		printHeader("Preview (when focused)"),
		printHelpExt("tab", "unfocus preview"),
		printHelpExt("k", "line up"),
		printHelpExt("j", "line down"),
		printHelpExt("d", "half page down"),
		printHelpExt("u", "half page up"),
		"",
		printHeader("Revisions"),
		printHelp(h.keyMap.ToggleSelect),
		printHelp(h.keyMap.New),
		printHelp(h.keyMap.Describe),
		printHelp(h.keyMap.Edit),
		printHelp(h.keyMap.Diff),
		printHelp(h.keyMap.Diffedit),
		printHelp(h.keyMap.Split),
		printHelp(h.keyMap.Abandon),
		printHelp(h.keyMap.Absorb),
		printHelp(h.keyMap.Undo),
		printHelp(h.keyMap.Details.Mode),
	)

	rightView := lipgloss.JoinVertical(lipgloss.Left,
		printHeader("Revset"),
		printHelp(h.keyMap.Revset),
		"",
		printMode(h.keyMap.Details.Mode, "Details"),
		printHelp(h.keyMap.Details.ToggleSelect),
		printHelp(h.keyMap.Details.Restore),
		printHelp(h.keyMap.Details.Split),
		printHelp(h.keyMap.Details.Diff),
		"",
		printMode(h.keyMap.Git.Mode, "Git"),
		printHelp(h.keyMap.Git.Push),
		printHelp(h.keyMap.Git.Fetch),
		"",
		printMode(h.keyMap.Bookmark.Mode, "Bookmarks"),
		printHelp(h.keyMap.Bookmark.Move),
		printHelp(h.keyMap.Bookmark.Set),
		printHelp(h.keyMap.Bookmark.Delete),
		"",
		printMode(h.keyMap.Rebase.Mode, "Rebase"),
		printHelp(h.keyMap.Rebase.Revision),
		printHelp(h.keyMap.Rebase.Source),
		printHelp(h.keyMap.Rebase.Branch),
		printHelp(h.keyMap.Rebase.Before),
		printHelp(h.keyMap.Rebase.After),
		printHelp(h.keyMap.Rebase.Onto),
		printHelp(h.keyMap.Apply),
		"",
		printMode(h.keyMap.Squash, "Squash"),
		"",
		printMode(h.keyMap.Evolog, "Evolog"),
		printHelp(h.keyMap.Diff),
	)

	content := lipgloss.JoinHorizontal(lipgloss.Left, leftView, "  ", rightView)

	return border.Render(content)
}

func New(context context.AppContext) *Model {
	keyMap := context.KeyMap()
	return &Model{
		keyMap: keyMap,
	}
}
