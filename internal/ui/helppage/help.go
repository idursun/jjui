package helppage

import (
	"fmt"
	"strings"

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
	text     lipgloss.Style
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
	return lipgloss.JoinHorizontal(0, h.styles.shortcut.Render(keyAligned), h.styles.dimmed.Render(desc))
}

func (h *Model) printHeader(header string) string {
	return h.printMode(key.NewBinding(), header)
}

func (h *Model) printMode(key key.Binding, name string) string {
	keyAligned := fmt.Sprintf("%9s", key.Help().Key)
	return lipgloss.JoinHorizontal(0, h.styles.shortcut.Render(keyAligned), h.styles.title.Render(name))
}

func (h *Model) View() string {
	var left []string
	left = append(left,
		h.printHeader("UI"),
		h.printHelp(h.keyMap.Refresh),
		h.printHelp(h.keyMap.Help),
		h.printHelp(h.keyMap.Cancel),
		h.printHelp(h.keyMap.Quit),
		h.printHelp(h.keyMap.Suspend),
		h.printHelp(h.keyMap.Revset),
		h.printHeader("Exec"),
		h.printHelp(h.keyMap.ExecJJ),
		h.printHelp(h.keyMap.ExecShell),
		h.printHeader("Revisions"),
		h.printHelpExt(fmt.Sprintf("%s/%s/%s",
			h.keyMap.JumpToParent.Help().Key,
			h.keyMap.JumpToChildren.Help().Key,
			h.keyMap.JumpToWorkingCopy.Help().Key,
		), "jump to parent/child/working-copy"),
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

	var middle []string
	middle = append(middle,
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

	var right []string
	right = append(right,
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

	right = append(right, customCommands...)

	maxHeight := max(len(left), len(right), len(middle))
	content := lipgloss.JoinHorizontal(lipgloss.Left,
		h.renderColumn(40, maxHeight, left...),
		h.renderColumn(40, maxHeight, middle...),
		h.renderColumn(40, maxHeight, right...),
	)
	return h.styles.border.Render(content)
}

func (h *Model) renderColumn(width int, height int, lines ...string) string {
	column := lipgloss.Place(width, height, 0, 0, strings.Join(lines, "\n"), lipgloss.WithWhitespaceBackground(h.styles.text.GetBackground()))
	return column
}

func New(context *context.MainContext) *Model {
	styles := styles{
		border:   common.DefaultPalette.GetBorder("help border", lipgloss.NormalBorder()).Padding(1),
		title:    common.DefaultPalette.Get("help title").PaddingLeft(1),
		text:     common.DefaultPalette.Get("help text"),
		dimmed:   common.DefaultPalette.Get("help dimmed").PaddingLeft(1),
		shortcut: common.DefaultPalette.Get("help shortcut"),
	}
	return &Model{
		context: context,
		keyMap:  config.Current.GetKeyMap(),
		styles:  styles,
	}
}
