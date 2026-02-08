package status

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/fuzzy_files"
	"github.com/idursun/jjui/internal/ui/fuzzy_input"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
)

var accept = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept"))

type commandStatus int

const (
	none commandStatus = iota
	commandRunning
	commandCompleted
	commandFailed
)

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	context         *context.MainContext
	spinner         spinner.Model
	input           textinput.Model
	keyMap          help.KeyMap
	command         string
	status          commandStatus
	running         bool
	mode            string
	editStatus      editStatus
	history         map[string][]string
	fuzzy           fuzzy_search.Model
	styles          styles
	statusExpanded  bool
	statusTruncated bool
}

type styles struct {
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
	text     lipgloss.Style
	title    lipgloss.Style
	success  lipgloss.Style
	error    lipgloss.Style
}

// a function that will be used to show
// dynamic help when editing is focused.
type editStatus = func() (help.KeyMap, string)

func emptyEditStatus() (help.KeyMap, string) {
	return nil, ""
}

func (m *Model) IsFocused() bool {
	return m.editStatus != nil
}

const CommandClearDuration = 3 * time.Second

type clearMsg string

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	km := config.Current.GetKeyMap()
	switch msg := msg.(type) {
	case clearMsg:
		if m.command == string(msg) {
			m.command = ""
			m.status = none
		}
		return nil
	case common.CommandRunningMsg:
		m.command = string(msg)
		m.status = commandRunning
		return m.spinner.Tick
	case common.CommandCompletedMsg:
		if msg.Err != nil {
			m.status = commandFailed
		} else {
			m.status = commandCompleted
		}
		commandToBeCleared := m.command
		return tea.Tick(CommandClearDuration, func(time.Time) tea.Msg {
			return clearMsg(commandToBeCleared)
		})
	case common.FileSearchMsg:
		m.mode = "rev file"
		m.input.Prompt = "> "
		m.loadEditingSuggestions()
		m.fuzzy, m.editStatus = fuzzy_files.NewModel(msg)
		return tea.Batch(m.fuzzy.Init(), m.input.Focus())
	case common.ExecProcessCompletedMsg:
		if msg.Err != nil {
			m.mode = "exec " + msg.Msg.Mode.Mode
			m.input.Prompt = msg.Msg.Mode.Prompt
			m.loadEditingSuggestions()
			m.fuzzy, m.editStatus = fuzzy_input.NewModel(&m.input, m.input.AvailableSuggestions())

			// Avoid to change the current behavior when coming back from exec process
			focusCmd := m.input.Focus()
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(msg.Msg.Line)}
			updateCmd := m.Update(keyMsg)
			return tea.Batch(m.fuzzy.Init(), focusCmd, updateCmd)
		}
		return nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, km.Cancel) && m.statusExpanded:
			m.statusExpanded = false
			return nil
		case key.Matches(msg, km.Cancel) && m.IsFocused():
			var cmd tea.Cmd
			if m.fuzzy != nil {
				cmd = m.fuzzy.Update(msg)
			}

			m.fuzzy = nil
			m.editStatus = nil
			m.input.Reset()
			return cmd
		case key.Matches(msg, accept) && m.IsFocused():
			editMode := m.mode
			input := m.input.Value()
			prompt := m.input.Prompt
			fuzzy := m.fuzzy
			// If there's a selected match in the fuzzy list, use that instead
			if fuzzy != nil {
				if selected := fuzzy_search.SelectedMatch(fuzzy); selected != "" {
					// SelectedMatch returns the value wrapped in single quotes, remove them
					input = strings.Trim(selected, "'")
					m.input.SetValue(input)
				}
			}
			m.saveEditingSuggestions()

			m.fuzzy = nil
			m.command = ""
			m.editStatus = nil
			m.mode = ""
			m.input.Reset()

			switch {
			case strings.HasSuffix(editMode, "file"):
				cmd := fuzzy.Update(msg)
				return cmd
			case strings.HasPrefix(editMode, "exec"):
				return func() tea.Msg { return exec_process.ExecMsgFromLine(prompt, input) }
			}
			return func() tea.Msg { return common.QuickSearchMsg(input) }
		default:
			if m.IsFocused() {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				if m.fuzzy != nil {
					cmd = tea.Batch(cmd, fuzzy_search.Search(m.input.Value(), msg))
				}
				return cmd
			}
		}
		return nil
	default:
		var cmd tea.Cmd
		if m.status == commandRunning {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		if m.fuzzy != nil {
			cmd = m.fuzzy.Update(msg)
		}
		return cmd
	}
}

func (m *Model) StartExec(mode common.ExecMode) tea.Cmd {
	m.mode = "exec " + mode.Mode
	m.input.Prompt = mode.Prompt
	m.loadEditingSuggestions()

	m.fuzzy, m.editStatus = fuzzy_input.NewModel(&m.input, m.input.AvailableSuggestions())
	return tea.Batch(m.fuzzy.Init(), m.input.Focus())
}

func (m *Model) StartQuickSearch() tea.Cmd {
	m.editStatus = emptyEditStatus
	m.mode = "search"
	m.input.Prompt = "> "
	m.loadEditingSuggestions()
	return m.input.Focus()
}

func (m *Model) saveEditingSuggestions() {
	input := m.input.Value()
	if len(strings.TrimSpace(input)) == 0 {
		return
	}
	h := m.context.Histories.GetHistory(config.HistoryKey(m.mode), true)
	h.Append(input)
}

func (m *Model) loadEditingSuggestions() {
	h := m.context.Histories.GetHistory(config.HistoryKey(m.mode), true)
	history := h.Entries()
	m.input.ShowSuggestions = true
	m.input.SetSuggestions([]string(history))
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	width := box.R.Dx()
	modeWidth := max(10, len(m.mode)+2)
	mode := m.styles.title.Width(modeWidth).Render(" ", m.mode)

	var statusLine string
	switch {
	case m.IsFocused():
		content := m.renderContent(width, modeWidth)
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, mode, m.styles.text.Render(" "), content)
	case m.status != none:
		statusMark := m.renderStatusMark()
		content := m.renderContent(width, modeWidth)
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, mode, m.styles.text.Render(" "), statusMark, content)
	default:
		helpBar := m.renderHelpBar(width, modeWidth)
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, mode, m.styles.text.Render(" "), helpBar)
	}

	dl.AddDraw(box.R, statusLine, 0)
	m.renderExpandedStatus(dl, box, width)
	m.renderFuzzyOverlay(dl, box)
}

// renderStatusMark returns the command status indicator (spinner/success/error).
func (m *Model) renderStatusMark() string {
	switch m.status {
	case commandRunning:
		return m.styles.text.Render(m.spinner.View())
	case commandFailed:
		return m.styles.error.Render("✗ ")
	case commandCompleted:
		return m.styles.success.Render("✓ ")
	}
	return ""
}

// renderHelpBar renders the help keybindings bar when idle.
func (m *Model) renderHelpBar(width, modeWidth int) string {
	if m.keyMap == nil || m.statusExpanded {
		return m.styles.text.Render(" ")
	}

	availableWidth := max(0, width-modeWidth-2)
	helpContent, truncated := m.helpView(m.keyMap, availableWidth)
	m.statusTruncated = truncated
	return lipgloss.PlaceHorizontal(width, 0, helpContent, lipgloss.WithWhitespaceBackground(m.styles.text.GetBackground()))
}

// renderContent handles input vs command display
func (m *Model) renderContent(width, modeWidth int) string {
	if !m.IsFocused() {
		return m.styles.text.Render(strings.ReplaceAll(m.command, "\n", "⏎"))
	}

	editKeys, editHelp := m.editStatus()
	if editKeys != nil {
		editHelpText, _ := m.helpView(editKeys, 0)
		editHelp = lipgloss.JoinHorizontal(0, editHelpText, editHelp)
	}

	promptWidth := len(m.input.Prompt) + 2
	m.input.Width = width - modeWidth - promptWidth - lipgloss.Width(editHelp)
	return lipgloss.JoinHorizontal(0, m.input.View(), editHelp)
}

// renderExpandedStatus orchestrates expanded help overlay
func (m *Model) renderExpandedStatus(dl *render.DisplayContext, box layout.Box, width int) {
	if !m.statusExpanded || m.keyMap == nil || m.IsFocused() {
		return
	}

	expandedHelp, contentLineCount := m.expandedStatusView(m.keyMap, max(0, width-4))
	expandedLines := strings.Split(expandedHelp, "\n")
	startY := box.R.Min.Y - contentLineCount

	m.renderExpandedStatusBorder(dl, box, width, startY)
	m.renderExpandedStatusContent(dl, box, width, startY, expandedLines)
}

// renderExpandedStatusBorder draws the top border of expanded status
func (m *Model) renderExpandedStatusBorder(dl *render.DisplayContext, box layout.Box, width, startY int) {
	if startY < 0 {
		return
	}
	modeLabel := m.styles.title.Render("  " + m.mode + "  ")
	borderLine := strings.Repeat("─", max(0, width-lipgloss.Width(modeLabel)))
	topBorder := modeLabel + m.styles.dimmed.Render(borderLine)
	borderRect := cellbuf.Rect(box.R.Min.X, startY, width, 1)
	dl.AddDraw(borderRect, topBorder, render.ZExpandedStatus)
}

// renderExpandedStatusContent draws the content for the expanded status
// Each line is a single row, positioned below the border (hence startY + 1)
//
// Each line is left-padded with 2 spaces and right-padded to fill the
// available width, accounting for 4 total characters of horizontal padding
// (2 left + 2 reserved for borders).
func (m *Model) renderExpandedStatusContent(dl *render.DisplayContext, box layout.Box, width, startY int, lines []string) {
	for i, line := range lines {
		// Position each line below the border, offset by its index
		y := startY + 1 + i

		// Skip lines that would render above the visible area
		if y < 0 {
			continue
		}

		// calculate right padding
		// subtract 4 for: 2 chars left padding + 2 chars border space
		padding := max(0, width-lipgloss.Width(line)-4)

		// padded line: 2-space indent + content + right padding
		paddedLine := "  " + line + strings.Repeat(" ", padding)

		// render the line with the text style and draw at the overlay z-index
		contentLine := m.styles.text.Render(paddedLine)
		contentRect := cellbuf.Rect(box.R.Min.X, y, width, 1)
		dl.AddDraw(contentRect, contentLine, render.ZExpandedStatus)
	}
}

// renderFuzzyOverlay handles fuzzy search overlay
func (m *Model) renderFuzzyOverlay(dl *render.DisplayContext, box layout.Box) {
	if m.fuzzy == nil {
		return
	}
	overlayRect := cellbuf.Rect(box.R.Min.X, 0, box.R.Dx(), box.R.Min.Y)
	m.fuzzy.ViewRect(dl, layout.Box{R: overlayRect})
}

func (m *Model) SetHelp(keyMap help.KeyMap) {
	if m.keyMap != keyMap {
		m.statusExpanded = false
	}
	m.keyMap = keyMap
}

// StatusExpanded returns whether the help overlay is currently expanded.
func (m *Model) StatusExpanded() bool {
	return m.statusExpanded
}

// StatusTruncated returns whether the help text is currently truncated.
func (m *Model) StatusTruncated() bool {
	return m.statusTruncated
}

// ToggleStatusExpand toggles the expanded footer help view.
func (m *Model) ToggleStatusExpand() {
	if m.IsFocused() {
		return
	}
	if m.statusExpanded || m.statusTruncated {
		m.statusExpanded = !m.statusExpanded
	}
}

func (m *Model) Help() help.KeyMap {
	return m.keyMap
}

func (m *Model) SetMode(mode string) {
	if !m.IsFocused() {
		m.mode = mode
	}
}

func (m *Model) Mode() string {
	return m.mode
}

func (m *Model) expandedStatusView(keyMap help.KeyMap, maxWidth int) (string, int) {
	entries, maxEntryWidth := m.collectHelpEntries(keyMap)
	lines := m.buildHelpGrid(entries, maxEntryWidth, maxWidth)
	return strings.Join(lines, "\n"), len(lines)
}

// collectHelpEntries gathers all help entries from the keymap and returns them
// along with the maximum entry width for column layout calculation.
func (m *Model) collectHelpEntries(keyMap help.KeyMap) ([]string, int) {
	shortHelp := keyMap.ShortHelp()
	expandKey := config.Current.GetKeyMap().ExpandStatus.Help().Key
	closeHint := m.styles.shortcut.Render(expandKey+"/esc") + m.styles.dimmed.PaddingLeft(1).Render("close help")

	var entries []string
	maxEntryWidth := 0

	for _, binding := range shortHelp {
		if !binding.Enabled() {
			continue
		}
		h := binding.Help()
		entry := m.styles.shortcut.Render(h.Key) + m.styles.dimmed.PaddingLeft(1).Render(h.Desc)
		entries = append(entries, entry)
		if w := lipgloss.Width(entry); w > maxEntryWidth {
			maxEntryWidth = w
		}
	}

	if w := lipgloss.Width(closeHint); w > maxEntryWidth {
		maxEntryWidth = w
	}
	entries = append(entries, closeHint)

	return entries, maxEntryWidth
}

// buildHelpGrid arranges entries into a multi-column grid that fits within
// maxWidth.
func (m *Model) buildHelpGrid(entries []string, maxEntryWidth, maxWidth int) []string {
	minColWidth := maxEntryWidth + 2
	numCols := max(maxWidth/minColWidth, 1)
	colWidth := maxWidth / numCols
	numRows := (len(entries) + numCols - 1) / numCols

	var lines []string
	for row := range numRows {
		var line strings.Builder
		for col := range numCols {
			idx := row*numCols + col
			if idx < len(entries) {
				entry := entries[idx]
				line.WriteString(entry)
				if col < numCols-1 {
					padding := max(0, colWidth-lipgloss.Width(entry))
					line.WriteString(strings.Repeat(" ", padding))
				}
			}
		}
		lines = append(lines, line.String())
	}

	return lines
}

func (m *Model) helpView(keyMap help.KeyMap, maxWidth int) (string, bool) {
	separator := m.styles.dimmed.Render(" • ")
	expandKey := config.Current.GetKeyMap().ExpandStatus.Help().Key
	moreHint := separator + m.styles.shortcut.Render(expandKey) + m.styles.dimmed.PaddingLeft(1).Render("more")

	entries, truncated := m.collectHelpEntriesWithLimit(keyMap, maxWidth, lipgloss.Width(separator), lipgloss.Width(moreHint))

	help := strings.Join(entries, separator)
	if truncated {
		help += moreHint
	}
	return help, truncated
}

// collectHelpEntriesWithLimit gathers help entries that fit within maxWidth,
// accounting for separators and the "more" hint when truncation occurs.
func (m *Model) collectHelpEntriesWithLimit(keyMap help.KeyMap, maxWidth, separatorWidth, moreHintWidth int) ([]string, bool) {
	shortHelp := keyMap.ShortHelp()
	var entries []string
	currentWidth := 0

	for i, binding := range shortHelp {
		if !binding.Enabled() {
			continue
		}

		h := binding.Help()
		entry := m.styles.shortcut.Render(h.Key) + m.styles.dimmed.PaddingLeft(1).Render(h.Desc)
		entryWidth := lipgloss.Width(entry)

		addedWidth := entryWidth
		if len(entries) > 0 {
			addedWidth += separatorWidth
		}

		reservedWidth := 0
		if i < len(shortHelp)-1 {
			reservedWidth = moreHintWidth
		}

		if maxWidth > 0 && currentWidth+addedWidth+reservedWidth > maxWidth {
			return entries, true
		}

		entries = append(entries, entry)
		currentWidth += addedWidth
	}

	return entries, false
}

func New(context *context.MainContext) *Model {
	styles := styles{
		shortcut: common.DefaultPalette.Get("status shortcut"),
		dimmed:   common.DefaultPalette.Get("status dimmed"),
		text:     common.DefaultPalette.Get("status text"),
		title:    common.DefaultPalette.Get("status title"),
		success:  common.DefaultPalette.Get("status success"),
		error:    common.DefaultPalette.Get("status error"),
	}
	s := spinner.New()
	s.Spinner = spinner.Dot

	t := textinput.New()
	t.Width = 50
	t.TextStyle = styles.text
	t.CompletionStyle = styles.dimmed
	t.PlaceholderStyle = styles.dimmed

	return &Model{
		context: context,
		spinner: s,
		command: "",
		status:  none,
		input:   t,
		keyMap:  nil,
		styles:  styles,
	}
}
