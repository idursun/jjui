package status

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/fuzzy_files"
	"github.com/idursun/jjui/internal/ui/fuzzy_input"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/idursun/jjui/internal/ui/help"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var expandFallback = help.Entry{Label: "?", Desc: "expand status"}

type FocusKind int

const (
	FocusNone FocusKind = iota
	FocusInput
	FocusFileSearch
	FocusQuickSearch
)

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	context         *context.MainContext
	input           textinput.Model
	groups          []help.ScopeGroup
	mode            string
	focusKind       FocusKind
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
}

func (m *Model) IsFocused() bool {
	return m.focusKind != FocusNone
}

func (m *Model) FocusKind() FocusKind {
	return m.focusKind
}

func (m *Model) Scopes() []dispatch.Scope {
	if m.focusKind == FocusNone {
		return nil
	}
	var scope keybindings.ScopeName
	switch m.focusKind {
	case FocusFileSearch:
		scope = actions.ScopeFileSearch
	case FocusInput:
		scope = actions.ScopeStatusInput
	case FocusQuickSearch:
		scope = actions.ScopeQuickSearchInput
	default:
		return nil
	}
	return []dispatch.Scope{
		{
			Name:    scope,
			Leak:    dispatch.LeakNone,
			Handler: m,
		},
	}
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.Cancel:
		if m.IsFocused() {
			editMode := m.mode
			fuzzy := m.fuzzy
			m.fuzzy = nil
			m.focusKind = FocusNone
			m.input.Reset()
			if fuzzy != nil && strings.HasSuffix(editMode, "file") {
				return fuzzy.Update(intents.FileSearchCancel{}), true
			}
			return nil, true
		}
	case intents.Apply:
		if m.IsFocused() {
			editMode := m.mode
			input := m.input.Value()
			prompt := m.input.Prompt
			fuzzy := m.fuzzy
			if fuzzy != nil {
				if selected := fuzzy_search.SelectedMatch(fuzzy); selected != "" {
					input = strings.Trim(selected, "'")
					m.input.SetValue(input)
				}
			}
			m.saveEditingSuggestions()

			m.fuzzy = nil
			m.focusKind = FocusNone
			m.mode = ""
			m.input.Reset()

			switch {
			case strings.HasSuffix(editMode, "file"):
				if fuzzy != nil {
					return fuzzy.Update(intents.FileSearchAccept{}), true
				}
				return nil, true
			case strings.HasPrefix(editMode, "exec"):
				return func() tea.Msg { return exec_process.ExecMsgFromLine(prompt, input) }, true
			}
			return func() tea.Msg { return common.QuickSearchMsg(input) }, true
		}
	}
	if m.IsFocused() && m.fuzzy != nil {
		return m.fuzzy.Update(intent), true
	}
	return nil, false
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case common.FileSearchMsg:
		m.mode = "rev file"
		m.input.Prompt = "> "
		m.loadEditingSuggestions()
		m.focusKind = FocusFileSearch
		m.fuzzy = fuzzy_files.NewModel(msg)
		return tea.Batch(m.fuzzy.Init(), m.input.Focus())
	case common.ExecProcessCompletedMsg:
		if msg.Err != nil {
			m.mode = "exec " + msg.Msg.Mode.Mode
			m.input.Prompt = msg.Msg.Mode.Prompt
			m.loadEditingSuggestions()
			m.focusKind = FocusInput
			m.fuzzy = fuzzy_input.NewModel(&m.input, m.input.AvailableSuggestions())
			m.input.SetValue(msg.Msg.Line)
			m.input.CursorEnd()

			return tea.Batch(m.fuzzy.Init(), m.input.Focus(), fuzzy_search.Search(m.input.Value()))
		}
		return nil
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
	case tea.KeyMsg, tea.PasteMsg:
		if m.IsFocused() {
			var cmd tea.Cmd
			previous := m.input.Value()
			m.input, cmd = m.input.Update(msg)
			if m.fuzzy != nil && m.input.Value() != previous {
				cmd = tea.Batch(cmd, fuzzy_search.Search(m.input.Value()))
			}
			return cmd
		}
		return nil
	default:
		if m.fuzzy != nil {
			return m.fuzzy.Update(msg)
		}
		return nil
	}
}

func (m *Model) StartExec(mode common.ExecMode) tea.Cmd {
	m.mode = "exec " + mode.Mode
	m.input.Prompt = mode.Prompt
	m.loadEditingSuggestions()
	m.focusKind = FocusInput

	m.fuzzy = fuzzy_input.NewModel(&m.input, m.input.AvailableSuggestions())
	return tea.Batch(m.fuzzy.Init(), m.input.Focus())
}

func (m *Model) StartQuickSearch() tea.Cmd {
	m.focusKind = FocusQuickSearch
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
	m.input.SetSuggestions(history)
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	width := box.R.Dx()
	modeWidth := max(10, len(m.mode)+2)
	mode := m.styles.title.Width(modeWidth).Render(m.mode)

	var statusLine string
	if m.IsFocused() {
		content := m.renderContent(width, modeWidth)
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, mode, m.styles.text.Render(" "), content)
	} else {
		helpBar := m.renderHelpBar(width, modeWidth)
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left, mode, m.styles.text.Render(" "), helpBar)
	}

	dl.AddDraw(box.R, statusLine, 0)
	m.renderExpandedStatus(dl, box, width)
	m.renderFuzzyOverlay(dl, box)
}

// renderHelpBar renders the help keybindings bar when idle.
func (m *Model) renderHelpBar(width, modeWidth int) string {
	if len(m.groups) == 0 || m.statusExpanded {
		return m.styles.text.Render(" ")
	}

	availableWidth := max(0, width-modeWidth-2)
	helpContent, truncated := m.groupedHelpView(m.groups, availableWidth)
	m.statusTruncated = truncated
	return lipgloss.PlaceHorizontal(width, 0, helpContent, lipgloss.WithWhitespaceStyle(m.styles.text))
}

// renderContent handles input display when focused
func (m *Model) renderContent(width, modeWidth int) string {
	var editHelp string
	if len(m.groups) > 0 {
		editHelp, _ = m.groupedHelpView(m.groups, 0)
	}

	promptWidth := render.StringWidth(m.input.Prompt) + 2
	m.input.SetWidth(width - modeWidth - promptWidth - render.StringWidth(editHelp))
	return lipgloss.JoinHorizontal(0, m.input.View(), editHelp)
}

// renderExpandedStatus orchestrates expanded help overlay
func (m *Model) renderExpandedStatus(dl *render.DisplayContext, box layout.Box, width int) {
	if !m.statusExpanded || len(m.groups) == 0 || m.IsFocused() {
		return
	}

	expandedHelp, contentLineCount := m.expandedStatusView(m.groups, max(0, width-4))
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
	borderLine := strings.Repeat("─", max(0, width))
	topBorder := m.styles.dimmed.Render(borderLine)
	borderRect := layout.Rect(box.R.Min.X, startY, width, 1)
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
		padding := max(0, width-render.StringWidth(line)-4)

		// padded line: 2-space indent + content + right padding
		paddedLine := "  " + line + strings.Repeat(" ", padding)

		// render the line with the text style and draw at the overlay z-index
		contentLine := m.styles.text.Render(paddedLine)
		contentRect := layout.Rect(box.R.Min.X, y, width, 1)
		dl.AddDraw(contentRect, contentLine, render.ZExpandedStatus)
	}
}

// renderFuzzyOverlay handles fuzzy search overlay
func (m *Model) renderFuzzyOverlay(dl *render.DisplayContext, box layout.Box) {
	if m.fuzzy == nil {
		return
	}
	overlayRect := layout.Rect(box.R.Min.X, 0, box.R.Dx(), box.R.Min.Y)
	m.fuzzy.ViewRect(dl, layout.Box{R: overlayRect})
}

func (m *Model) SetScopes(scopes []dispatch.Scope) {
	var scopeNames []keybindings.ScopeName
	for _, scope := range dispatch.VisibleScopes(scopes) {
		if scope.Name != "" {
			scopeNames = append(scopeNames, scope.Name)
		}
	}
	groups := help.BuildGroupedFromBindings(scopeNames, config.Current.Bindings)
	help.MarkOverriddenKeys(groups)
	m.setGroups(groups)
}

func (m *Model) SetHelp(entries []help.Entry) {
	if len(entries) == 0 {
		m.setGroups(nil)
		return
	}
	m.setGroups([]help.ScopeGroup{{Entries: entries}})
}

func (m *Model) setGroups(groups []help.ScopeGroup) {
	if len(m.groups) != len(groups) {
		m.statusExpanded = false
	}
	m.groups = groups
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

// SetStatusExpanded forces expanded help visibility.
func (m *Model) SetStatusExpanded(expanded bool) {
	if m.IsFocused() {
		return
	}
	m.statusExpanded = expanded
}

func (m *Model) Help() []help.ScopeGroup {
	return m.groups
}

var modeDisplayNames = map[string]string{
	"normal":          "revisions",
	"inline_describe": "describe",
}

func (m *Model) SetMode(mode string) {
	if !m.IsFocused() {
		if display, ok := modeDisplayNames[mode]; ok {
			m.mode = display
		} else {
			m.mode = mode
		}
	}
}

func (m *Model) Mode() string {
	return m.mode
}

func (m *Model) InputValue() string {
	return m.input.Value()
}

func (m *Model) expandedStatusView(groups []help.ScopeGroup, maxWidth int) (string, int) {
	expandKey := m.expandStatusKey(groups)
	closeHint := m.styles.shortcut.Render(expandKey+"/esc") + m.styles.dimmed.PaddingLeft(1).Render("close help")

	var allLines []string
	for i, group := range groups {
		if i > 0 {
			allLines = append(allLines, "")
		}
		if group.Name != "" {
			header := m.styles.title.Render(group.Name)
			allLines = append(allLines, header)
		}
		rendered, maxEntryWidth := m.collectGroupEntries(group.Entries)
		lines := m.buildHelpGrid(rendered, maxEntryWidth, maxWidth)
		allLines = append(allLines, lines...)
	}
	allLines = append(allLines, "", closeHint)
	return strings.Join(allLines, "\n"), len(allLines)
}

func (m *Model) collectGroupEntries(entries []help.Entry) ([]string, int) {
	var rendered []string
	maxEntryWidth := 0

	for _, entry := range entries {
		if entry.Label == "" || entry.Desc == "" {
			continue
		}
		var e string
		if entry.Overridden {
			e = m.styles.dimmed.Strikethrough(true).Render(entry.Label + " " + entry.Desc)
		} else {
			e = m.styles.shortcut.Render(entry.Label) + m.styles.dimmed.PaddingLeft(1).Render(entry.Desc)
		}
		rendered = append(rendered, e)
		if w := render.StringWidth(e); w > maxEntryWidth {
			maxEntryWidth = w
		}
	}

	return rendered, maxEntryWidth
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
			idx := col*numRows + row
			if idx < len(entries) {
				entry := entries[idx]
				line.WriteString(entry)
				if col < numCols-1 {
					padding := max(0, colWidth-render.StringWidth(entry))
					line.WriteString(strings.Repeat(" ", padding))
				}
			}
		}
		lines = append(lines, line.String())
	}

	return lines
}

// groupedHelpView renders the collapsed help bar with groups separated by │.
func (m *Model) groupedHelpView(groups []help.ScopeGroup, maxWidth int) (string, bool) {
	separator := m.styles.dimmed.Render(" • ")
	groupSeparator := m.styles.dimmed.Render(" │ ")
	expandKey := m.expandStatusKey(groups)
	moreHint := separator + m.styles.shortcut.Render(expandKey) + m.styles.dimmed.PaddingLeft(1).Render("more")

	separatorWidth := render.StringWidth(separator)
	groupSeparatorWidth := render.StringWidth(groupSeparator)
	moreHintWidth := render.StringWidth(moreHint)

	var result strings.Builder
	currentWidth := 0
	entryCount := 0
	truncated := false

	for gi, group := range groups {
		firstInGroup := true
		for ei, entry := range group.Entries {
			if entry.Label == "" || entry.Desc == "" || entry.Overridden {
				continue
			}

			e := m.styles.shortcut.Render(entry.Label) + m.styles.dimmed.PaddingLeft(1).Render(entry.Desc)
			entryWidth := render.StringWidth(e)

			addedWidth := entryWidth
			if entryCount > 0 {
				if firstInGroup {
					addedWidth += groupSeparatorWidth
				} else {
					addedWidth += separatorWidth
				}
			}

			isLast := gi == len(groups)-1 && ei == len(group.Entries)-1
			reservedWidth := 0
			if !isLast {
				reservedWidth = moreHintWidth
			}

			if maxWidth > 0 && currentWidth+addedWidth+reservedWidth > maxWidth {
				truncated = true
				result.WriteString(moreHint)
				return result.String(), truncated
			}

			if entryCount > 0 {
				if firstInGroup {
					result.WriteString(groupSeparator)
				} else {
					result.WriteString(separator)
				}
			}
			result.WriteString(e)
			currentWidth += addedWidth
			entryCount++
			firstInGroup = false
		}
	}

	return result.String(), truncated
}

func (m *Model) expandStatusKey(groups []help.ScopeGroup) string {
	for _, group := range groups {
		for _, entry := range group.Entries {
			if entry.Desc == "expand status" {
				return entry.Label
			}
		}
	}
	return expandFallback.Label
}

func New(context *context.MainContext) *Model {
	styles := styles{
		shortcut: common.DefaultPalette.Get("status shortcut"),
		dimmed:   common.DefaultPalette.Get("status dimmed"),
		text:     common.DefaultPalette.Get("status text"),
		title:    common.DefaultPalette.Get("status title").PaddingLeft(1).PaddingRight(1),
	}

	t := common.TextInputNew()
	t.SetWidth(50)
	ts := t.Styles()
	ts.Focused.Text = styles.text
	ts.Focused.Suggestion = styles.dimmed
	ts.Focused.Placeholder = styles.dimmed
	ts.Blurred.Text = styles.text
	ts.Blurred.Suggestion = styles.dimmed
	ts.Blurred.Placeholder = styles.dimmed
	t.SetStyles(ts)

	return &Model{
		context: context,
		input:   t,
		styles:  styles,
	}
}
