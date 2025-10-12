// Package helppage provides a help page model for jjui
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

func (h *Model) printKeyBinding(k key.Binding) string {
	return h.printKey(k.Help().Key, k.Help().Desc)
}

func (h *Model) printKey(key string, desc string) string {
	keyAligned := fmt.Sprintf("%9s", key)
	return lipgloss.JoinHorizontal(0, h.styles.shortcut.Render(keyAligned), h.styles.dimmed.Render(desc))
}

func (h *Model) printTitle(header string) string {
	return h.printMode(key.NewBinding(), header)
}

func (h *Model) printMode(key key.Binding, name string) string {
	keyAligned := fmt.Sprintf("%9s", key.Help().Key)
	return lipgloss.JoinHorizontal(0, h.styles.shortcut.Render(keyAligned), h.styles.title.Render(name))
}

func (h *Model) View() string {
	var left []string
	left = append(left,
		h.printTitle("UI"),
		h.printKeyBinding(h.keyMap.Refresh),
		h.printKeyBinding(h.keyMap.Help),
		h.printKeyBinding(h.keyMap.Cancel),
		h.printKeyBinding(h.keyMap.Quit),
		h.printKeyBinding(h.keyMap.Suspend),
		h.printKeyBinding(h.keyMap.Revset),
		h.printTitle("Exec"),
		h.printKeyBinding(h.keyMap.ExecJJ),
		h.printKeyBinding(h.keyMap.ExecShell),
		h.printTitle("Revisions"),
		h.printKey(fmt.Sprintf("%s/%s/%s",
			h.keyMap.JumpToParent.Help().Key,
			h.keyMap.JumpToChildren.Help().Key,
			h.keyMap.JumpToWorkingCopy.Help().Key,
		), "jump to parent/child/working-copy"),
		h.printKeyBinding(h.keyMap.ToggleSelect),
		h.printKeyBinding(h.keyMap.AceJump),
		h.printKeyBinding(h.keyMap.QuickSearch),
		h.printKeyBinding(h.keyMap.QuickSearchCycle),
		h.printKeyBinding(h.keyMap.FileSearch.Toggle),
		h.printKeyBinding(h.keyMap.New),
		h.printKeyBinding(h.keyMap.Commit),
		h.printKeyBinding(h.keyMap.Describe),
		h.printKeyBinding(h.keyMap.Edit),
		h.printKeyBinding(h.keyMap.Diff),
		h.printKeyBinding(h.keyMap.Diffedit),
		h.printKeyBinding(h.keyMap.Split),
		h.printKeyBinding(h.keyMap.Abandon),
		h.printKeyBinding(h.keyMap.Absorb),
		h.printKeyBinding(h.keyMap.Undo),
		h.printKeyBinding(h.keyMap.Redo),
		h.printKeyBinding(h.keyMap.Details.Mode),
		h.printKeyBinding(h.keyMap.Bookmark.Set),
		h.printKeyBinding(h.keyMap.InlineDescribe.Mode),
		h.printKeyBinding(h.keyMap.SetParents),
	)

	var middle []string
	middle = append(middle,
		h.printMode(h.keyMap.Details.Mode, "Details"),
		h.printKeyBinding(h.keyMap.Details.Close),
		h.printKeyBinding(h.keyMap.Details.ToggleSelect),
		h.printKeyBinding(h.keyMap.Details.Restore),
		h.printKeyBinding(h.keyMap.Details.Split),
		h.printKeyBinding(h.keyMap.Details.Squash),
		h.printKeyBinding(h.keyMap.Details.Diff),
		h.printKeyBinding(h.keyMap.Details.RevisionsChangingFile),
		"",
		h.printMode(h.keyMap.Evolog.Mode, "Evolog"),
		h.printKeyBinding(h.keyMap.Evolog.Diff),
		h.printKeyBinding(h.keyMap.Evolog.Restore),
		"",
		h.printMode(h.keyMap.Squash.Mode, "Squash"),
		h.printKeyBinding(h.keyMap.Squash.KeepEmptied),
		h.printKeyBinding(h.keyMap.Squash.Interactive),
		"",
		h.printMode(h.keyMap.Revert.Mode, "Revert"),
		"",
		h.printMode(h.keyMap.Rebase.Mode, "Rebase"),
		h.printKeyBinding(h.keyMap.Rebase.Revision),
		h.printKeyBinding(h.keyMap.Rebase.Source),
		h.printKeyBinding(h.keyMap.Rebase.Branch),
		h.printKeyBinding(h.keyMap.Rebase.Before),
		h.printKeyBinding(h.keyMap.Rebase.After),
		h.printKeyBinding(h.keyMap.Rebase.Onto),
		h.printKeyBinding(h.keyMap.Rebase.Insert),
		"",
		h.printMode(h.keyMap.Duplicate.Mode, "Duplicate"),
		h.printKeyBinding(h.keyMap.Duplicate.Onto),
		h.printKeyBinding(h.keyMap.Duplicate.Before),
		h.printKeyBinding(h.keyMap.Duplicate.After),
	)

	var right []string
	right = append(right,
		h.printMode(h.keyMap.Preview.Mode, "Preview"),
		h.printKeyBinding(h.keyMap.Preview.ScrollUp),
		h.printKeyBinding(h.keyMap.Preview.ScrollDown),
		h.printKeyBinding(h.keyMap.Preview.HalfPageDown),
		h.printKeyBinding(h.keyMap.Preview.HalfPageUp),
		h.printKeyBinding(h.keyMap.Preview.Expand),
		h.printKeyBinding(h.keyMap.Preview.Shrink),
		h.printKeyBinding(h.keyMap.Preview.ToggleBottom),
		"",
		h.printMode(h.keyMap.Git.Mode, "Git"),
		h.printKeyBinding(h.keyMap.Git.Push),
		h.printKeyBinding(h.keyMap.Git.Fetch),
		"",
		h.printMode(h.keyMap.Bookmark.Mode, "Bookmarks"),
		h.printKeyBinding(h.keyMap.Bookmark.Move),
		h.printKeyBinding(h.keyMap.Bookmark.Delete),
		h.printKeyBinding(h.keyMap.Bookmark.Untrack),
		h.printKeyBinding(h.keyMap.Bookmark.Track),
		h.printKeyBinding(h.keyMap.Bookmark.Forget),
		h.printMode(h.keyMap.OpLog.Mode, "Oplog"),
		h.printKeyBinding(h.keyMap.Diff),
		h.printKeyBinding(h.keyMap.OpLog.Restore),
		h.printMode(h.keyMap.Leader, "Leader"),
		h.printMode(h.keyMap.CustomCommands, "Custom Commands"),
	)

	var customCommands []string
	for _, command := range h.context.CustomCommands {
		customCommands = append(customCommands, h.printKeyBinding(command.Binding()))
	}

	right = append(right, customCommands...)

	maxHeight := max(len(left), len(right), len(middle))
	content := lipgloss.JoinHorizontal(lipgloss.Left,
		h.renderColumn(1+lipgloss.Width(strings.Join(left, "\n")), maxHeight, left...),
		h.renderColumn(1+lipgloss.Width(strings.Join(middle, "\n")), maxHeight, middle...),
		h.renderColumn(1+lipgloss.Width(strings.Join(right, "\n")), maxHeight, right...),
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
