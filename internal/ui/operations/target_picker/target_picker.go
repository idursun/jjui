package target_picker

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/sahilm/fuzzy"
)

type ItemKind int

const (
	KindBookmark ItemKind = iota
	KindTag
)

const (
	maxWidth  = 80
	maxHeight = 20
	pillWidth = 8
)

type Item struct {
	Name string
	Kind ItemKind
}

type Model struct {
	context             *context.MainContext
	items               []Item
	input               textinput.Model
	cursor              int
	matches             fuzzy.Matches
	styles              styles
	fzfSource           *fuzzy_search.RefinedSource
	listRenderer        *render.ListRenderer
	ensureCursorVisible bool
	keyMap              config.KeyMappings[key.Binding]
}

type styles struct {
	bookmarkPill lipgloss.Style
	tagPill      lipgloss.Style
	selected     lipgloss.Style
	dimmed       lipgloss.Style
	matchStyle   lipgloss.Style
	border       lipgloss.Style
}

type itemsLoadedMsg struct {
	items []Item
}

type itemClickedMsg struct {
	index int
}

type itemScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (m itemScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	m.Delta = delta
	m.Horizontal = horizontal
	return m
}

type TargetSelectedMsg struct {
	Target string
	Force  bool
}

type TargetPickerCancelMsg struct{}

var _ common.ImmediateModel = (*Model)(nil)

func NewModel(ctx *context.MainContext) *Model {
	palette := common.DefaultPalette
	text := palette.Get("picker text")
	dimmed := palette.Get("picker dimmed")
	ti := textinput.New()
	ti.Prompt = "> "
	ti.PromptStyle = dimmed
	ti.TextStyle = text
	ti.CharLimit = 0
	ti.Focus()

	return &Model{
		context: ctx,
		input:   ti,
		cursor:  0,
		keyMap:  config.Current.GetKeyMap(),
		styles: styles{
			bookmarkPill: palette.Get("picker bookmark"),
			tagPill:      palette.Get("picker dimmed"),
			selected:     palette.Get("picker selected"),
			dimmed:       dimmed,
			matchStyle:   palette.Get("picker matched"),
			border:       palette.GetBorder("picker border", lipgloss.NormalBorder()),
		},
		listRenderer:        render.NewListRenderer(itemScrollMsg{}),
		ensureCursorVisible: true,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.fetchItems()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case itemsLoadedMsg:
		m.items = msg.items
		m.fzfSource = &fuzzy_search.RefinedSource{Source: m}
		m.listRenderer.StartLine = 0
		m.search("")
		return textinput.Blink
	case itemClickedMsg:
		m.cursor = msg.index
		m.ensureCursorVisible = true
		return m.accept(false)
	case itemScrollMsg:
		if msg.Horizontal {
			return nil
		}
		m.ensureCursorVisible = false
		m.listRenderer.StartLine += msg.Delta
		if m.listRenderer.StartLine < 0 {
			m.listRenderer.StartLine = 0
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Cancel):
			return TargetPickerCancelCmd()
		case key.Matches(msg, m.keyMap.ForceApply):
			return m.accept(true)
		case key.Matches(msg, m.keyMap.Apply):
			return m.accept(false)
		case msg.Type == tea.KeyUp:
			m.cursorUp()
			return nil
		case msg.Type == tea.KeyDown:
			m.cursorDown()
			return nil
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.search(m.input.Value())
			return cmd
		}
	}
	return nil
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}

	maxW := min(maxWidth, box.R.Dx())
	maxH := min(maxHeight, box.R.Dy())
	centeredBox := box.Center(maxW, maxH)

	borderContent := m.styles.border.Width(centeredBox.R.Dx() - 2).Height(centeredBox.R.Dy() - 2).Render("")
	window := dl.Window(centeredBox.R, render.ZMenuBorder)
	window.AddDraw(centeredBox.R, borderContent, render.ZMenuBorder)
	centeredBox = centeredBox.Inset(1)

	inputBox, listBox := centeredBox.CutTop(1)
	m.input.Width = inputBox.R.Dx()

	window.AddDraw(inputBox.R, m.input.View(), render.ZMenuContent)

	m.listRenderer.Render(
		window,
		listBox,
		len(m.matches),
		m.cursor,
		m.ensureCursorVisible,
		func(_ int) int { return 1 },
		func(dl *render.DisplayContext, index int, rect cellbuf.Rectangle) {
			if index < 0 || index >= len(m.matches) {
				return
			}
			match := m.matches[index]
			item := m.items[match.Index]
			y := rect.Min.Y

			pillText := m.renderPill(item.Kind)
			pillRect := cellbuf.Rect(rect.Min.X, y, pillWidth, 1)
			window.AddDraw(pillRect, pillText, render.ZMenuContent)

			isSelected := index == m.cursor
			lineStyle := m.styles.bookmarkPill
			matchStyle := m.styles.matchStyle
			if isSelected {
				window.AddHighlight(rect, m.styles.selected, render.ZMenuContent+1)
			} else {
				matchStyle = matchStyle.Inherit(lineStyle)
			}
			nameContent := fuzzy_search.HighlightMatched(item.Name, match, lineStyle, matchStyle)
			nameX := rect.Min.X + pillWidth + 1
			nameRect := cellbuf.Rect(nameX, y, rect.Dx()-pillWidth-1, 1)
			window.AddDraw(nameRect, nameContent, render.ZMenuContent)
		},
		func(index int) tea.Msg { return itemClickedMsg{index: index} },
	)
	m.listRenderer.RegisterScroll(window, listBox)
	m.ensureCursorVisible = false
}

func (m *Model) renderPill(kind ItemKind) string {
	switch kind {
	case KindBookmark:
		return m.styles.dimmed.Width(pillWidth).Align(lipgloss.Right).Render("bookmark")
	case KindTag:
		return m.styles.dimmed.Width(pillWidth).Align(lipgloss.Right).Render("tag")
	default:
		return strings.Repeat(" ", pillWidth)
	}
}

func (m *Model) fetchItems() tea.Cmd {
	return func() tea.Msg {
		var items []Item
		if output, err := m.context.RunCommandImmediate(jj.BookmarkListAll()); err == nil {
			bookmarks := jj.ParseBookmarkListOutput(string(output))
			for _, b := range bookmarks {
				if b.Name == "" {
					continue
				}
				if b.Local != nil {
					items = append(items, Item{Name: b.Name, Kind: KindBookmark})
				}
				for _, remote := range b.Remotes {
					nameWithRemote := fmt.Sprintf("%s@%s", b.Name, remote.Remote)
					items = append(items, Item{Name: nameWithRemote, Kind: KindBookmark})
				}
			}
		}

		if output, err := m.context.RunCommandImmediate(jj.TagList()); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, "\t", 2)
				name := strings.TrimSpace(parts[0])
				if name == "" {
					continue
				}
				items = append(items, Item{Name: name, Kind: KindTag})
			}
		}

		return itemsLoadedMsg{items: items}
	}
}

func (m *Model) search(input string) {
	if m.fzfSource == nil {
		return
	}
	m.matches = m.fzfSource.Search(input, len(m.items))
	if len(m.matches) == 0 {
		m.cursor = -1
		m.listRenderer.StartLine = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.matches) {
		m.cursor = len(m.matches) - 1
	}
	m.ensureCursorVisible = true
}

func (m *Model) cursorUp() {
	if len(m.matches) == 0 {
		return
	}
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.matches) - 1
	}
	m.ensureCursorVisible = true
}

func (m *Model) cursorDown() {
	if len(m.matches) == 0 {
		return
	}
	m.cursor++
	if m.cursor >= len(m.matches) {
		m.cursor = 0
	}
	m.ensureCursorVisible = true
}

func (m *Model) accept(force bool) tea.Cmd {
	if m.cursor >= 0 && m.cursor < len(m.matches) {
		item := m.items[m.matches[m.cursor].Index]
		return TargetSelectedCmd(item.Name, force)
	}
	if input := strings.TrimSpace(m.input.Value()); input != "" {
		return TargetSelectedCmd(input, force)
	}
	return nil
}

func TargetSelectedCmd(target string, force bool) tea.Cmd {
	return func() tea.Msg { return TargetSelectedMsg{Target: target, Force: force} }
}

func TargetPickerCancelCmd() tea.Cmd {
	return func() tea.Msg { return TargetPickerCancelMsg{} }
}

func (m *Model) Len() int {
	return len(m.items)
}

func (m *Model) String(i int) string {
	if i < 0 || i >= len(m.items) {
		return ""
	}
	return m.items[i].Name
}
