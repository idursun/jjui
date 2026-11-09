package status

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actionmeta"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/help"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
)

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	context         *context.MainContext
	groups          []help.ScopeGroup
	mode            string
	statusExpanded  bool
	statusTruncated bool
	checkedCount    int
	conditionCache  map[string]actionmeta.Condition
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(tea.Msg) tea.Cmd {
	return nil
}

// ViewRect renders the mode and keybinding help row.
func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	shortcutStyle := theme.DefaultPalette.Get("status", "", "shortcut", false)
	dimmedStyle := theme.DefaultPalette.Get("status", "", "dimmed", false)
	textStyle := theme.DefaultPalette.Get("status", "", "text", false)
	titleStyle := theme.DefaultPalette.Get("status", "", "title", false).PaddingLeft(1).PaddingRight(1)

	width := box.R.Dx()
	dl.AddFill(box.R, ' ', textStyle, 0)
	// without a mode label, indent by the label's left padding to line up with the prompt label above
	prefix := textStyle.Render(" ")
	if m.mode != "" {
		prefix = titleStyle.Width(max(10, len(m.mode)+2)).Render(m.mode) + prefix
	}
	if m.checkedCount > 0 {
		prefix += shortcutStyle.Render(fmt.Sprintf("%d checked", m.checkedCount)) + textStyle.Render(" ")
	}
	statusLine := prefix + m.renderHelpBar(max(0, width-lipgloss.Width(prefix)), textStyle, shortcutStyle, dimmedStyle)
	dl.AddDraw(box.R, statusLine, 0)
	m.renderExpandedStatus(dl, box, width, textStyle, titleStyle, shortcutStyle, dimmedStyle)
}

// renderHelpBar renders the help keybindings bar.
func (m *Model) renderHelpBar(width int, textStyle, shortcutStyle, dimmedStyle lipgloss.Style) string {
	if len(m.groups) == 0 || m.statusExpanded {
		return textStyle.Render(" ")
	}

	helpContent, truncated := m.groupedHelpView(m.groups, width, shortcutStyle, dimmedStyle)
	m.statusTruncated = truncated
	return lipgloss.PlaceHorizontal(width, 0, helpContent, lipgloss.WithWhitespaceStyle(textStyle))
}

// renderExpandedStatus orchestrates expanded help overlay
func (m *Model) renderExpandedStatus(dl *render.DisplayContext, box layout.Box, width int, textStyle, titleStyle, shortcutStyle, dimmedStyle lipgloss.Style) {
	if !m.statusExpanded || len(m.groups) == 0 {
		return
	}

	// Keep at least one row of the main view visible above the expanded help.
	// The full keybinding list is available from the dedicated help view.
	maxLines := max(1, box.R.Min.Y-1)
	expandedHelp, contentLineCount := m.expandedStatusView(m.groups, max(0, width-4), maxLines, titleStyle, shortcutStyle, dimmedStyle)
	expandedLines := strings.Split(expandedHelp, "\n")
	startY := box.R.Min.Y - contentLineCount

	background := lipgloss.NewStyle().Background(textStyle.GetBackground())
	backgroundRect := layout.Rect(box.R.Min.X, startY, width, contentLineCount+1)
	dl.AddFill(backgroundRect, ' ', background, render.ZExpandedStatus)
	m.renderExpandedStatusBorder(dl, box, width, startY, dimmedStyle)
	m.renderExpandedStatusContent(dl, box, width, startY, expandedLines, textStyle)
	dl.AddPaint(backgroundRect, background, render.ZExpandedStatus)
}

// renderExpandedStatusBorder draws the top border of expanded status
func (m *Model) renderExpandedStatusBorder(dl *render.DisplayContext, box layout.Box, width, startY int, dimmedStyle lipgloss.Style) {
	if startY < 0 {
		return
	}
	borderLine := strings.Repeat("─", max(0, width))
	topBorder := dimmedStyle.Render(borderLine)
	borderRect := layout.Rect(box.R.Min.X, startY, width, 1)
	dl.AddDraw(borderRect, topBorder, render.ZExpandedStatus)
}

// renderExpandedStatusContent draws each line below the top border, with a
// 2-column margin on both sides.
func (m *Model) renderExpandedStatusContent(dl *render.DisplayContext, box layout.Box, width, startY int, lines []string, textStyle lipgloss.Style) {
	for i, line := range lines {
		// Position each line below the border, offset by its index
		y := startY + 1 + i

		// Skip lines that would render above the visible area
		if y < 0 {
			continue
		}

		padding := max(0, width-render.StringWidth(line)-4)

		paddedLine := "  " + line + strings.Repeat(" ", padding)

		contentLine := textStyle.Render(paddedLine)
		contentRect := layout.Rect(box.R.Min.X, y, width, 1)
		dl.AddDraw(contentRect, contentLine, render.ZExpandedStatus)
	}
}

func (m *Model) SetCheckedCount(count int) {
	m.checkedCount = count
}

func (m *Model) Sync(scopes []common.Scope, sequenceHelp []help.Entry) {
	m.setModeFromScopes(scopes)
	if sequenceHelp != nil {
		m.setHelp(sequenceHelp)
		return
	}
	m.setHelpFromScopes(scopes)
}

func (m *Model) setHelpFromScopes(scopes []common.Scope) {
	var scopeNames []keybindings.ScopeName
	for _, scope := range common.VisibleScopes(scopes) {
		if scope.Name != "" && !scope.HideFromHelp {
			scopeNames = append(scopeNames, scope.Name)
		}
	}
	visible := make(map[string]bool, len(scopeNames))
	for _, scope := range scopeNames {
		visible[string(scope)] = true
	}
	if m.conditionCache == nil {
		m.conditionCache = make(map[string]actionmeta.Condition)
	}
	visibleBindings := make([]config.BindingConfig, 0, len(config.Current.Bindings))
	for _, binding := range config.Current.Bindings {
		if visible[strings.TrimSpace(binding.Scope)] {
			visibleBindings = append(visibleBindings, binding)
		}
	}
	groups := help.BuildGroupedFromBindingsWithAvailability(scopeNames, visibleBindings, func(binding config.BindingConfig) bool {
		actionCondition, valid := m.condition(config.Current.ActionWhen(binding.Action))
		if !valid || !actionCondition.Matches(m.context) {
			return false
		}
		condition, valid := m.condition(binding.When)
		return valid && condition.Matches(m.context)
	})
	help.MarkOverriddenKeys(groups)
	m.setGroups(groups)
}

func (m *Model) condition(source string) (actionmeta.Condition, bool) {
	condition, ok := m.conditionCache[source]
	if ok {
		return condition, true
	}
	condition, err := actionmeta.ParseCondition(source)
	if err != nil {
		return actionmeta.Condition{}, false
	}
	m.conditionCache[source] = condition
	return condition, true
}

func (m *Model) setModeFromScopes(scopes []common.Scope) {
	scope, ok := firstNamedScope(common.VisibleScopes(scopes))
	if !ok {
		return
	}
	// prompts draw their own mode label next to their input
	if scope.Name == actions.ScopeStatusInput || scope.Name == actions.ScopeQuickSearchInput {
		m.mode = ""
		return
	}
	m.mode = modeNameForScope(scope.Name)
}

func (m *Model) setHelp(entries []help.Entry) {
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
	if m.statusExpanded || m.statusTruncated {
		m.statusExpanded = !m.statusExpanded
	}
}

// SetStatusExpanded forces expanded help visibility.
func (m *Model) SetStatusExpanded(expanded bool) {
	m.statusExpanded = expanded
}

func (m *Model) Help() []help.ScopeGroup {
	return m.groups
}

func firstNamedScope(scopes []common.Scope) (common.Scope, bool) {
	for _, scope := range scopes {
		if scope.Name != "" && !scope.HideFromHelp {
			return scope, true
		}
	}
	return common.Scope{}, false
}

func modeNameForScope(scope keybindings.ScopeName) string {
	return strings.ToLower(help.ScopeDisplayName(string(scope)))
}

func (m *Model) Mode() string {
	return m.mode
}

func (m *Model) expandedStatusView(groups []help.ScopeGroup, maxWidth, maxLines int, titleStyle, shortcutStyle, dimmedStyle lipgloss.Style) (string, int) {
	expandKey := expandStatusKey(groups)
	closeHint := shortcutStyle.Render(expandKey+"/esc") + dimmedStyle.PaddingLeft(1).Render("close help")
	fullHelpHint := shortcutStyle.Render(fullHelpKey(groups)) + dimmedStyle.PaddingLeft(1).Render("full help")

	renderedGroups := make([][]string, len(groups))
	maxEntryWidth := 0
	for i, group := range groups {
		rendered, groupMaxEntryWidth := m.collectGroupEntries(group.Entries, shortcutStyle, dimmedStyle)
		renderedGroups[i] = rendered
		maxEntryWidth = max(maxEntryWidth, groupMaxEntryWidth)
	}

	var allLines []string
	for i, group := range groups {
		if i > 0 {
			allLines = append(allLines, "")
		}
		if group.Name != "" {
			header := titleStyle.Render(group.Name)
			allLines = append(allLines, header)
		}
		lines := m.buildHelpGrid(renderedGroups[i], maxEntryWidth, maxWidth)
		allLines = append(allLines, lines...)
	}
	allLines = append(allLines, "", closeHint)
	if maxLines > 0 && len(allLines) > maxLines {
		// Reserve space for a full-help hint and the close hint when the
		// expanded status cannot fit in the available viewport.
		if maxLines == 1 {
			allLines = []string{closeHint}
		} else {
			const hintLines = 2
			contentLines := maxLines - hintLines
			allLines = append(allLines[:contentLines], fullHelpHint, closeHint)
		}
	}
	return strings.Join(allLines, "\n"), len(allLines)
}

func (m *Model) collectGroupEntries(entries []help.Entry, shortcutStyle, dimmedStyle lipgloss.Style) ([]string, int) {
	var rendered []string
	maxEntryWidth := 0

	for _, entry := range entries {
		if entry.Label == "" || entry.Desc == "" {
			continue
		}
		var e string
		if entry.Overridden || entry.Disabled {
			e = dimmedStyle.Strikethrough(true).Render(entry.Label + " " + entry.Desc)
		} else {
			e = shortcutStyle.Render(entry.Label) + dimmedStyle.PaddingLeft(1).Render(entry.Desc)
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
	colWidth := minColWidth
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
func (m *Model) groupedHelpView(groups []help.ScopeGroup, maxWidth int, shortcutStyle, dimmedStyle lipgloss.Style) (string, bool) {
	separator := dimmedStyle.Render(" • ")
	groupSeparator := dimmedStyle.Render(" │ ")
	expandKey := expandStatusKey(groups)
	moreHint := separator + shortcutStyle.Render(expandKey) + dimmedStyle.PaddingLeft(1).Render("more")

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
			if entry.Label == "" || entry.Desc == "" || entry.Overridden || entry.Disabled {
				continue
			}

			e := shortcutStyle.Render(entry.Label) + dimmedStyle.PaddingLeft(1).Render(entry.Desc)
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

func expandStatusKey(groups []help.ScopeGroup) string {
	return actionKey(groups, "ui.expand_status", "?")
}

func fullHelpKey(groups []help.ScopeGroup) string {
	return actionKey(groups, "ui.open_help", "f1")
}

// actionKey returns the label of the first active entry bound to action.
func actionKey(groups []help.ScopeGroup, action, fallback string) string {
	for _, entry := range help.FlatEntries(groups) {
		if entry.Action == action && !entry.Overridden && !entry.Disabled {
			return entry.Label
		}
	}
	return fallback
}

func New(context *context.MainContext) *Model {
	return &Model{
		context: context,
	}
}
