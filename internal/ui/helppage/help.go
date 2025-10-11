// Package helppage provides a help page model for jjui
package helppage

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type Model struct {
	width        int
	height       int
	keyMap       config.KeyMappings[key.Binding]
	context      *context.MainContext
	styles       styles
	searchActive bool
	searchQuery  string
}

type styles struct {
	border   lipgloss.Style
	title    lipgloss.Style
	text     lipgloss.Style
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
}

type helpEntry struct {
	view   string
	search string
}

func newHelpEntry(view string, parts ...string) helpEntry {
	var normalized []string
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	var search string
	if len(normalized) > 0 {
		search = strings.ToLower(strings.Join(normalized, " "))
	}
	return helpEntry{
		view:   view,
		search: search,
	}
}

func (e helpEntry) matches(query string) bool {
	if query == "" {
		return false
	}
	if e.search == "" {
		return false
	}
	return strings.Contains(e.search, query)
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
	clearSearch := func() {
		h.searchActive = false
		h.searchQuery = ""
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if h.searchActive {
			switch {
			case key.Matches(msg, h.keyMap.Help):
				return h, common.Close
			case key.Matches(msg, h.keyMap.Cancel):
				clearSearch()
				return h, nil
			}

			switch msg.Type {
			case tea.KeyEsc:
				clearSearch()
			case tea.KeyRunes:
				h.searchQuery += msg.String()
			case tea.KeyBackspace, tea.KeyCtrlH, tea.KeyDelete:
				if len(h.searchQuery) > 0 {
					_, size := utf8.DecodeLastRuneInString(h.searchQuery)
					h.searchQuery = h.searchQuery[:len(h.searchQuery)-size]
				}
			default:
				// ignore other keys while in search mode
			}
			return h, nil
		}

		if msg.Type == tea.KeyRunes && msg.String() == "/" {
			h.searchActive = true
			h.searchQuery = ""
			return h, nil
		}

		switch {
		case key.Matches(msg, h.keyMap.Help), key.Matches(msg, h.keyMap.Cancel):
			return h, common.Close
		}
	}
	return h, nil
}

func (h *Model) keyBindingEntry(k key.Binding) helpEntry {
	return h.keyEntry(k.Help().Key, k.Help().Desc)
}

func (h *Model) keyEntry(key string, desc string) helpEntry {
	view := h.printKey(key, desc)
	return newHelpEntry(view, key, desc)
}

func (h *Model) titleEntry(header string) helpEntry {
	return newHelpEntry(h.printTitle(header), header)
}

func (h *Model) modeEntry(binding key.Binding, name string) helpEntry {
	help := binding.Help()
	view := h.printMode(binding, name)
	return newHelpEntry(view, help.Key, help.Desc, name)
}

func (h *Model) blankEntry() helpEntry {
	return newHelpEntry("")
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
	left, middle, right := h.defaultColumns()
	leftStrings := entriesToStrings(left)
	middleStrings := entriesToStrings(middle)
	rightStrings := entriesToStrings(right)

	maxHeight := max(len(leftStrings), len(rightStrings), len(middleStrings))

	leftWidth := 1 + lipgloss.Width(strings.Join(leftStrings, "\n"))
	middleWidth := 1 + lipgloss.Width(strings.Join(middleStrings, "\n"))
	rightWidth := 1 + lipgloss.Width(strings.Join(rightStrings, "\n"))

	if h.searchActive {
		// only display the left column in search mode
		leftStrings = h.searchColumn(maxHeight, left, middle, right)
		middleStrings = make([]string, maxHeight)
		rightStrings = make([]string, maxHeight)
	} else {
		leftStrings = padHeight(leftStrings, maxHeight)
		middleStrings = padHeight(middleStrings, maxHeight)
		rightStrings = padHeight(rightStrings, maxHeight)
	}

	content := lipgloss.JoinHorizontal(lipgloss.Left,
		h.renderColumn(leftWidth, maxHeight, leftStrings...),
		h.renderColumn(middleWidth, maxHeight, middleStrings...),
		h.renderColumn(rightWidth, maxHeight, rightStrings...),
	)
	return h.styles.border.Render(content)
}

func (h *Model) defaultColumns() ([]helpEntry, []helpEntry, []helpEntry) {
	var left []helpEntry
	left = append(left,
		h.titleEntry("UI"),
		h.keyBindingEntry(h.keyMap.Refresh),
		h.keyBindingEntry(h.keyMap.Help),
		h.keyBindingEntry(h.keyMap.Cancel),
		h.keyBindingEntry(h.keyMap.Quit),
		h.keyBindingEntry(h.keyMap.Suspend),
		h.keyBindingEntry(h.keyMap.Revset),
		h.titleEntry("Exec"),
		h.keyBindingEntry(h.keyMap.ExecJJ),
		h.keyBindingEntry(h.keyMap.ExecShell),
		h.titleEntry("Revisions"),
		h.keyEntry(fmt.Sprintf("%s/%s/%s",
			h.keyMap.JumpToParent.Help().Key,
			h.keyMap.JumpToChildren.Help().Key,
			h.keyMap.JumpToWorkingCopy.Help().Key,
		), "jump to parent/child/working-copy"),
		h.keyBindingEntry(h.keyMap.ToggleSelect),
		h.keyBindingEntry(h.keyMap.AceJump),
		h.keyBindingEntry(h.keyMap.QuickSearch),
		h.keyBindingEntry(h.keyMap.QuickSearchCycle),
		h.keyBindingEntry(h.keyMap.FileSearch.Toggle),
		h.keyBindingEntry(h.keyMap.New),
		h.keyBindingEntry(h.keyMap.Commit),
		h.keyBindingEntry(h.keyMap.Describe),
		h.keyBindingEntry(h.keyMap.Edit),
		h.keyBindingEntry(h.keyMap.Diff),
		h.keyBindingEntry(h.keyMap.Diffedit),
		h.keyBindingEntry(h.keyMap.Split),
		h.keyBindingEntry(h.keyMap.Abandon),
		h.keyBindingEntry(h.keyMap.Absorb),
		h.keyBindingEntry(h.keyMap.Undo),
		h.keyBindingEntry(h.keyMap.Redo),
		h.keyBindingEntry(h.keyMap.Details.Mode),
		h.keyBindingEntry(h.keyMap.Bookmark.Set),
		h.keyBindingEntry(h.keyMap.InlineDescribe.Mode),
		h.keyBindingEntry(h.keyMap.SetParents),
	)

	var middle []helpEntry
	middle = append(middle,
		h.modeEntry(h.keyMap.Details.Mode, "Details"),
		h.keyBindingEntry(h.keyMap.Details.Close),
		h.keyBindingEntry(h.keyMap.Details.ToggleSelect),
		h.keyBindingEntry(h.keyMap.Details.Restore),
		h.keyBindingEntry(h.keyMap.Details.Split),
		h.keyBindingEntry(h.keyMap.Details.Squash),
		h.keyBindingEntry(h.keyMap.Details.Diff),
		h.keyBindingEntry(h.keyMap.Details.RevisionsChangingFile),
		h.blankEntry(),
		h.modeEntry(h.keyMap.Evolog.Mode, "Evolog"),
		h.keyBindingEntry(h.keyMap.Evolog.Diff),
		h.keyBindingEntry(h.keyMap.Evolog.Restore),
		h.blankEntry(),
		h.modeEntry(h.keyMap.Squash.Mode, "Squash"),
		h.keyBindingEntry(h.keyMap.Squash.KeepEmptied),
		h.keyBindingEntry(h.keyMap.Squash.Interactive),
		h.blankEntry(),
		h.modeEntry(h.keyMap.Revert.Mode, "Revert"),
		h.blankEntry(),
		h.modeEntry(h.keyMap.Rebase.Mode, "Rebase"),
		h.keyBindingEntry(h.keyMap.Rebase.Revision),
		h.keyBindingEntry(h.keyMap.Rebase.Source),
		h.keyBindingEntry(h.keyMap.Rebase.Branch),
		h.keyBindingEntry(h.keyMap.Rebase.Before),
		h.keyBindingEntry(h.keyMap.Rebase.After),
		h.keyBindingEntry(h.keyMap.Rebase.Onto),
		h.keyBindingEntry(h.keyMap.Rebase.Insert),
		h.blankEntry(),
		h.modeEntry(h.keyMap.Duplicate.Mode, "Duplicate"),
		h.keyBindingEntry(h.keyMap.Duplicate.Onto),
		h.keyBindingEntry(h.keyMap.Duplicate.Before),
		h.keyBindingEntry(h.keyMap.Duplicate.After),
	)

	var right []helpEntry
	right = append(right,
		h.modeEntry(h.keyMap.Preview.Mode, "Preview"),
		h.keyBindingEntry(h.keyMap.Preview.ScrollUp),
		h.keyBindingEntry(h.keyMap.Preview.ScrollDown),
		h.keyBindingEntry(h.keyMap.Preview.HalfPageDown),
		h.keyBindingEntry(h.keyMap.Preview.HalfPageUp),
		h.keyBindingEntry(h.keyMap.Preview.Expand),
		h.keyBindingEntry(h.keyMap.Preview.Shrink),
		h.keyBindingEntry(h.keyMap.Preview.ToggleBottom),
		h.blankEntry(),
		h.modeEntry(h.keyMap.Git.Mode, "Git"),
		h.keyBindingEntry(h.keyMap.Git.Push),
		h.keyBindingEntry(h.keyMap.Git.Fetch),
		h.blankEntry(),
		h.modeEntry(h.keyMap.Bookmark.Mode, "Bookmarks"),
		h.keyBindingEntry(h.keyMap.Bookmark.Move),
		h.keyBindingEntry(h.keyMap.Bookmark.Delete),
		h.keyBindingEntry(h.keyMap.Bookmark.Untrack),
		h.keyBindingEntry(h.keyMap.Bookmark.Track),
		h.keyBindingEntry(h.keyMap.Bookmark.Forget),
		h.modeEntry(h.keyMap.OpLog.Mode, "Oplog"),
		h.keyBindingEntry(h.keyMap.Diff),
		h.keyBindingEntry(h.keyMap.OpLog.Restore),
		h.blankEntry(),
		h.modeEntry(h.keyMap.Leader, "Leader"),
		h.modeEntry(h.keyMap.CustomCommands, "Custom Commands"),
	)

	for _, command := range h.context.CustomCommands {
		right = append(right, h.keyBindingEntry(command.Binding()))
	}

	return left, middle, right
}

func entriesToStrings(entries []helpEntry) []string {
	lines := make([]string, len(entries))
	for i, entry := range entries {
		lines[i] = entry.view
	}
	return lines
}

func (h *Model) searchColumn(height int, columns ...[]helpEntry) []string {
	var allEntries []helpEntry
	for _, column := range columns {
		allEntries = append(allEntries, column...)
	}

	lines := make([]string, 0, height)
	header := fmt.Sprintf("Search: %s", h.searchQuery)
	lines = append(lines, h.styles.title.Render(header))

	if len(lines) < height {
		lines = append(lines, h.styles.dimmed.Render("Esc to cancel. Type to filter help entries."))
	}

	query := strings.ToLower(h.searchQuery)
	start := len(lines)
	if query == "" {
		for _, entry := range allEntries {
			if len(lines) == height {
				break
			}
			lines = append(lines, entry.view)
		}
	} else {
		for _, entry := range allEntries {
			if entry.matches(query) {
				if len(lines) == height {
					break
				}
				lines = append(lines, entry.view)
			}
		}
		if len(lines) == start && len(lines) < height {
			lines = append(lines, h.styles.dimmed.Render("No matching help entries."))
		}
	}

	if len(lines) < height {
		lines = append(lines, make([]string, height-len(lines))...)
	}
	return lines[:height]
}

func padHeight(lines []string, target int) []string {
	out := make([]string, target)
	copy(out, lines)
	return out
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
