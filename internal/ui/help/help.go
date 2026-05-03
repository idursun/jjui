package help

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var (
	_ common.StackedModel = (*Model)(nil)
	_ common.Editable     = (*Model)(nil)
	_ common.Focusable    = (*Model)(nil)
)

var scopeDisplayNames = map[string]string{
	"ui":                             "Global",
	"ui.preview":                     "Preview",
	"revisions":                      "Revisions",
	"revisions.rebase":               "Rebase",
	"revisions.squash":               "Squash",
	"revisions.revert":               "Revert",
	"revisions.duplicate":            "Duplicate",
	"revisions.abandon":              "Abandon",
	"revisions.set_parents":          "Set Parents",
	"revisions.details":              "Details",
	"revisions.details.confirmation": "Details Confirmation",
	"revisions.evolog":               "Evolog",
	"revisions.inline_describe":      "Inline Describe",
	"revisions.set_bookmark":         "Set Bookmark",
	"revisions.target_picker":        "Target Picker",
	"revisions.ace_jump":             "Ace Jump",
	"revisions.quick_search":         "Quick Search",
	"revisions.quick_search.input":   "Quick Search Input",
	"bookmarks":                      "Bookmarks",
	"bookmarks.filter":               "Bookmarks Filter",
	"git":                            "Git",
	"git.filter":                     "Git Filter",
	"oplog":                          "Oplog",
	"oplog.quick_search":             "Oplog Search",
	"diff":                           "Diff",
	"undo":                           "Undo",
	"redo":                           "Redo",
	"revset":                         "Revset",
	"command_history":                "Command History",
	"file_search":                    "File Search",
	"status.input":                   "Status Input",
	"input":                          "Input",
	"password":                       "Password",
	"choose":                         "Choose",
	"choose.filter":                  "Choose Filter",
}

// scopeOrder defines the display order of scopes in the help view.
var scopeOrder = []string{
	"revisions",
	"revisions.rebase",
	"revisions.squash",
	"revisions.revert",
	"revisions.duplicate",
	"revisions.abandon",
	"revisions.set_parents",
	"revisions.details",
	"revisions.details.confirmation",
	"revisions.evolog",
	"revisions.inline_describe",
	"revisions.set_bookmark",
	"revisions.target_picker",
	"revisions.ace_jump",
	"revisions.quick_search",
	"revisions.quick_search.input",
	"bookmarks",
	"bookmarks.filter",
	"git",
	"git.filter",
	"oplog",
	"oplog.quick_search",
	"diff",
	"file_search",
	"command_history",
	"undo",
	"redo",
	"revset",
	"status.input",
	"ui",
	"ui.preview",
}

type Model struct {
	groups    []ScopeGroup
	scroll    int
	input     textinput.Model
	filtering bool
	filtered  []ScopeGroup
}

type helpScrollMsg struct{}

func (helpScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	if horizontal {
		return nil
	}
	return intents.HelpScroll{Delta: delta}
}

func (m *Model) IsEditing() bool {
	return m.filtering
}

func (m *Model) IsFocused() bool {
	return m.filtering
}

func (m *Model) Scopes() []common.Scope {
	if m.IsEditing() {
		return []common.Scope{
			{
				Name:    actions.ScopeHelp + ".filter",
				Leak:    common.LeakNone,
				Handler: m,
			},
			{
				Name:    actions.ScopeHelp,
				Leak:    common.LeakNone,
				Handler: m,
			},
		}
	}
	return []common.Scope{
		{
			Name:    actions.ScopeHelp,
			Leak:    common.LeakAll,
			Handler: m,
		},
	}
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch msg := intent.(type) {
	case intents.Cancel:
		if m.filtering {
			m.filtering = false
			m.input.Reset()
			m.input.Blur()
			m.filtered = nil
			m.scroll = 0
			return nil, true
		}
		return common.Close, true
	case intents.Apply:
		if m.filtering {
			m.filtering = false
			m.input.Blur()
			return nil, true
		}
		return nil, false
	case intents.HelpClose:
		return common.Close, true
	case intents.HelpFilter:
		m.filtering = true
		return m.input.Focus(), true
	case intents.HelpScroll:
		if msg.Delta == 0 {
			m.scroll = 0
		} else {
			m.scroll = max(0, m.scroll+msg.Delta)
		}
		return nil, true
	}
	return nil, false
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
	case tea.KeyMsg, tea.PasteMsg:
		if m.filtering {
			prev := m.input.Value()
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			if m.input.Value() != prev {
				m.applyFilter()
				m.scroll = 0
			}
			return cmd
		}
	}
	return nil
}

func (m *Model) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if query == "" {
		m.filtered = nil
		return
	}
	var result []ScopeGroup
	for _, group := range m.groups {
		var matched []Entry
		for _, e := range group.Entries {
			if strings.Contains(strings.ToLower(e.Desc), query) ||
				strings.Contains(strings.ToLower(e.Label), query) ||
				strings.Contains(strings.ToLower(group.Name), query) {
				matched = append(matched, e)
			}
		}
		if len(matched) > 0 {
			result = append(result, ScopeGroup{Name: group.Name, Entries: matched})
		}
	}
	m.filtered = result
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	borderStyle := common.DefaultPalette.GetBorder("help border", lipgloss.NormalBorder()).Padding(0)
	titleStyle := common.DefaultPalette.Get("help title")
	headingStyle := titleStyle
	shortcutStyle := common.DefaultPalette.Get("help shortcut")
	dimmedStyle := common.DefaultPalette.Get("help dimmed")
	descStyle := common.DefaultPalette.Get("help desc").Inherit(dimmedStyle)

	inputStyles := m.input.Styles()
	inputStyles.Focused.Text = shortcutStyle
	inputStyles.Focused.Placeholder = dimmedStyle
	inputStyles.Blurred.Text = shortcutStyle
	inputStyles.Blurred.Placeholder = dimmedStyle
	m.input.SetStyles(inputStyles)

	pw, ph := box.R.Dx(), box.R.Dy()
	contentWidth := max(min(pw, 90)-4, 0)
	contentHeight := max(min(ph, 50)-4, 0)
	menuWidth := max(contentWidth+2, 0)
	menuHeight := max(contentHeight+2, 0)
	frame := box.Center(menuWidth, menuHeight)
	if frame.R.Dx() <= 0 || frame.R.Dy() <= 0 {
		return
	}

	dl.AddBackdrop(box.R, render.ZMenuBorder-1)
	contentBox := frame.Inset(1)
	if contentBox.R.Dx() <= 0 || contentBox.R.Dy() <= 0 {
		return
	}
	dl.AddFill(contentBox.R, ' ', dimmedStyle, render.ZMenuContent)

	borderBase := lipgloss.NewStyle().Width(contentBox.R.Dx()).Height(contentBox.R.Dy()).Render("")
	dl.AddDraw(frame.R, borderStyle.Render(borderBase), render.ZMenuBorder)

	titleBox, contentBox := contentBox.CutTop(1)
	title := titleStyle.Render("  Keybindings  ")
	dl.AddDraw(titleBox.R, title, render.ZMenuContent)

	filterBox, contentBox := contentBox.CutTop(1)
	filterLine := "  " + m.input.View()
	dl.AddDraw(filterBox.R, filterLine, render.ZMenuContent)

	_, contentBox = contentBox.CutTop(1)

	groups := m.groups
	if m.filtered != nil {
		groups = m.filtered
	}
	lines := m.renderGroups(groups, contentBox.R.Dx(), headingStyle, shortcutStyle, descStyle)

	// clamp scroll
	maxScroll := max(0, len(lines)-contentBox.R.Dy())
	m.scroll = min(m.scroll, maxScroll)

	// Enable mouse wheel scrolling in the help content area.
	dl.AddInteraction(contentBox.R, helpScrollMsg{}, render.InteractionScroll, render.ZMenuContent)

	visible := lines[m.scroll:]
	if len(visible) > contentBox.R.Dy() {
		visible = visible[:contentBox.R.Dy()]
	}

	for i, line := range visible {
		y := contentBox.R.Min.Y + i
		rect := layout.Rect(contentBox.R.Min.X, y, contentBox.R.Dx(), 1)
		dl.AddDraw(rect, line, render.ZMenuContent)
	}
}

func (m *Model) renderGroups(groups []ScopeGroup, width int, headingStyle, shortcutStyle, descStyle lipgloss.Style) []string {
	var lines []string
	for i, group := range groups {
		if i > 0 {
			lines = append(lines, "")
		}
		header := headingStyle.Width(width).Render("  " + group.Name + " ")
		lines = append(lines, header)

		entryLines := m.renderEntries(group.Entries, width, shortcutStyle, descStyle)
		lines = append(lines, entryLines...)
	}
	return lines
}

func (m *Model) renderEntries(entries []Entry, width int, shortcutStyle, descStyle lipgloss.Style) []string {
	maxLabelWidth := 0
	for _, e := range entries {
		if w := render.StringWidth(e.Label); w > maxLabelWidth {
			maxLabelWidth = w
		}
	}

	colWidth := maxLabelWidth + 2 + 20 // label + gap + desc estimate
	numCols := min(max(width/colWidth, 1), 3)
	actualColWidth := width / numCols
	numRows := (len(entries) + numCols - 1) / numCols

	var lines []string
	for row := range numRows {
		var line strings.Builder
		for col := range numCols {
			idx := col*numRows + row
			if idx >= len(entries) {
				continue
			}
			e := entries[idx]
			label := shortcutStyle.Width(maxLabelWidth + 1).Render(e.Label)
			desc := descStyle.Render(e.Desc)
			entry := "  " + label + " " + desc
			entryWidth := render.StringWidth(entry)
			if col < numCols-1 {
				entry += strings.Repeat(" ", max(0, actualColWidth-entryWidth))
			}
			line.WriteString(entry)
		}
		lines = append(lines, line.String())
	}
	return lines
}

func New() *Model {
	groups := buildGroups(config.Current.Bindings)

	ti := textinput.New()
	ti.Placeholder = "search"
	ti.Prompt = "/ "
	ti.SetWidth(40)

	return &Model{
		groups: groups,
		input:  ti,
	}
}

var skipScopes = map[string]bool{
	"help":          true,
	"input":         true,
	"password":      true,
	"choose":        true,
	"choose.filter": true,
}

func buildGroups(bindings []config.BindingConfig) []ScopeGroup {
	byScope := make(map[string][]config.BindingConfig)
	for _, b := range bindings {
		scope := strings.TrimSpace(b.Scope)
		if skipScopes[scope] {
			continue
		}
		byScope[scope] = append(byScope[scope], b)
	}

	var groups []ScopeGroup
	seen := make(map[string]bool)

	for _, scope := range scopeOrder {
		scopeBindings, ok := byScope[scope]
		if !ok {
			continue
		}
		seen[scope] = true
		entries := bindingsToEntries(scopeBindings)
		if len(entries) == 0 {
			continue
		}
		name := ScopeDisplayName(scope)
		groups = append(groups, ScopeGroup{Name: name, Entries: entries})
	}

	// Any scopes not in scopeOrder
	for scope, scopeBindings := range byScope {
		if seen[scope] {
			continue
		}
		entries := bindingsToEntries(scopeBindings)
		if len(entries) == 0 {
			continue
		}
		name := ScopeDisplayName(scope)
		groups = append(groups, ScopeGroup{Name: name, Entries: entries})
	}

	return groups
}

func bindingsToEntries(bindings []config.BindingConfig) []Entry {
	var entries []Entry
	seenActions := make(map[string]bool)
	for _, b := range bindings {
		action := strings.TrimSpace(b.Action)
		if action == "" {
			continue
		}
		label := BindingLabel(b)
		if label == "" {
			continue
		}
		desc := strings.TrimSpace(b.Desc)
		if desc == "" {
			desc = descFromAction(action)
		}

		key := action + "|" + desc
		if seenActions[key] {
			continue
		}
		seenActions[key] = true

		entries = append(entries, Entry{Label: label, Desc: desc})
	}
	return entries
}

func ScopeDisplayName(scope string) string {
	if name, ok := scopeDisplayNames[scope]; ok {
		return name
	}
	// Derive from scope string
	parts := strings.Split(scope, ".")
	last := parts[len(parts)-1]
	return strings.ReplaceAll(last, "_", " ")
}
