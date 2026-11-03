package fuzzy_files

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/theme"
	"github.com/sahilm/fuzzy"
)

var (
	_ common.StackedModel = (*Model)(nil)
	_ common.ScopeHandler = (*Model)(nil)
	_ common.Editable     = (*Model)(nil)
)

var debounceDuration = 250 * time.Millisecond

type filesLoadedMsg struct {
	paths []jj.FileName
}

type debouncePreview int

type itemClickedMsg struct {
	index int
}

type itemScrollMsg struct {
	Delta      int
	Horizontal bool
}

func newItemScrollMsg(msg tea.MouseMsg) tea.Msg {
	delta, horizontal := render.WheelDelta(msg)
	return itemScrollMsg{Delta: delta, Horizontal: horizontal}
}

type Model struct {
	context *context.MainContext
	// restored on cancel
	revset string
	commit *jj.Commit

	input        textinput.Model
	paths        []jj.FileName
	loaded       bool
	src          *fuzzy_search.RefinedSource
	matches      fuzzy.Matches
	cursor       int
	listRenderer *render.ListRenderer
	// set when the cursor moves, cleared after a frame so wheel scrolling sticks
	ensureCursorVisible bool

	// live preview of revset and rev-diff, toggled with ctrl+t
	revsetPreview bool
	debounceTag   int
}

func New(ctx *context.MainContext, revset string, commit *jj.Commit) *Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.SetVirtualCursor(false)
	ti.Focus()

	m := &Model{
		context:      ctx,
		revset:       revset,
		commit:       commit,
		input:        ti,
		listRenderer: render.NewListRenderer(newItemScrollMsg),
	}
	m.src = &fuzzy_search.RefinedSource{Source: m}
	m.listRenderer.Z = render.ZFuzzyOverlay
	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.loadFiles(), textinput.Blink)
}

func (m *Model) loadFiles() tea.Cmd {
	runner := m.context.RunCommandImmediate
	args := jj.FilesInRevision(m.commit)
	return func() tea.Msg {
		out, _ := runner(args)
		return filesLoadedMsg{paths: buildPathEntries(out)}
	}
}

func (m *Model) IsEditing() bool {
	return true
}

func (m *Model) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeFileSearch,
			Leak:    common.LeakNone,
			Handler: m,
		},
	}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
	case filesLoadedMsg:
		m.paths = msg.paths
		m.loaded = true
		m.search()
	case itemClickedMsg:
		m.cursor = msg.index
		m.ensureCursorVisible = true
		return m.accept()
	case itemScrollMsg:
		if !msg.Horizontal {
			m.ensureCursorVisible = false
			m.listRenderer.StartLine = max(0, m.listRenderer.StartLine+msg.Delta)
		}
	case debouncePreview:
		if int(msg) == m.debounceTag && m.revsetPreview {
			return tea.Batch(m.updateRevSet(), showPreview)
		}
	case tea.KeyMsg, tea.PasteMsg:
		previous := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() == previous {
			return cmd
		}
		m.search()
		return tea.Batch(cmd, m.schedulePreview())
	}
	return nil
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Cancel:
		return tea.Batch(common.UpdateRevSet(m.revset), common.Close), true
	case intents.Apply:
		return m.accept(), true
	case intents.FileSearchNavigate:
		if m.revsetPreview {
			return revisions.RevisionsCmd(intents.Navigate{Delta: -intent.Delta}), true
		}
		m.moveCursor(intent.Delta)
		return nil, true
	case intents.FileSearchEdit:
		path := m.selectedPath()
		return func() tea.Msg {
			return common.ExecMsg{
				Line: config.GetDefaultEditor() + " " + path.ShellEscaped(),
				Mode: common.ExecShell,
			}
		}, true
	case intents.FileSearchTogglePreview:
		m.revsetPreview = !m.revsetPreview
		if m.revsetPreview {
			return tea.Batch(showPreview, m.updateRevSet()), true
		}
		return m.updateRevSet(), true
	case intents.FileSearchPreviewScroll:
		if m.revsetPreview {
			return func() tea.Msg { return intents.PreviewScroll(intent) }, true
		}
		switch intent.Kind {
		case intents.PreviewScrollUp:
			m.moveCursor(1)
		case intents.PreviewScrollDown:
			m.moveCursor(-1)
		}
		return nil, true
	}
	return nil, false
}

func showPreview() tea.Msg {
	return common.ShowPreview{}
}

func (m *Model) accept() tea.Cmd {
	return tea.Batch(m.updateRevSet(), common.CloseApplied)
}

func (m *Model) schedulePreview() tea.Cmd {
	if !m.revsetPreview {
		return nil
	}
	m.debounceTag++
	tag := debouncePreview(m.debounceTag)
	return tea.Tick(debounceDuration, func(time.Time) tea.Msg { return tag })
}

func (m *Model) updateRevSet() tea.Cmd {
	path := m.selectedPath()
	revset := m.revset
	if !path.IsEmpty() {
		revset = fmt.Sprintf("files(%s)", path.Escaped())
	}
	return common.UpdateRevSet(revset)
}

func (m *Model) search() {
	m.cursor = 0
	m.matches = m.src.Search(m.input.Value(), len(m.paths))
	m.listRenderer.StartLine = 0
	m.ensureCursorVisible = true
}

// cursor indexes matches; +1 moves towards weaker matches (up on screen)
func (m *Model) moveCursor(delta int) {
	n := len(m.matches)
	if n == 0 {
		return
	}
	m.cursor = ((m.cursor+delta)%n + n) % n
	m.ensureCursorVisible = true
}

func (m *Model) selectedPath() jj.FileName {
	if m.cursor < 0 || m.cursor >= len(m.matches) {
		return jj.FileName{}
	}
	index := m.matches[m.cursor].Index
	if index < 0 || index >= len(m.paths) {
		return jj.FileName{}
	}
	return m.paths[index]
}

func (m *Model) Len() int {
	return len(m.paths)
}

func (m *Model) String(i int) string {
	if i < 0 || i >= len(m.paths) {
		return ""
	}
	return m.paths[i].Display(m.context.Location, m.context.WorkingDirectory)
}

// ViewRect draws bottom-up, fzf style: input on the last row above the
// status line, the best match right above it, and the title on top.
func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	area, _ := box.CutBottom(1)
	if area.R.Dx() <= 0 || area.R.Dy() < 2 {
		return
	}

	background := theme.DefaultPalette.Get("status", "", "text", false)
	titleStyle := theme.DefaultPalette.Get("status", "", "title", false)
	dimmedStyle := theme.DefaultPalette.Get("status", "", "dimmed", false)
	dimmedMatchStyle := theme.DefaultPalette.Get("status", "", "shortcut", false)
	selectedStyle := theme.DefaultPalette.GetBlended("", "", "", true)

	listHeight := min(len(m.matches), max(0, area.R.Dy()/2-2))
	height := listHeight + 2
	rect := layout.Rect(area.R.Min.X, area.R.Max.Y-height, area.R.Dx(), height)
	dl.AddFill(rect, ' ', background, render.ZFuzzyOverlay-1)

	titleBox, rest := layout.NewBox(rect).CutTop(1)
	listBox, inputBox := rest.CutBottom(1)

	title := "  loading paths…"
	if m.loaded {
		title = "  " + strconv.Itoa(len(m.matches)) + " of " + strconv.Itoa(len(m.paths)) +
			" paths present at revision " + m.commit.GetChangeId() + " "
	}
	dl.Text(titleBox.R.Min.X, titleBox.R.Min.Y, render.ZFuzzyOverlay).Styled(title, titleStyle).Done()

	// the list renders top-down, so row i shows match n-1-i
	n := len(m.matches)
	m.listRenderer.Render(
		dl,
		listBox,
		n,
		n-1-m.cursor,
		m.ensureCursorVisible,
		func(int) int { return 1 },
		func(dl *render.DisplayContext, row int, r layout.Rectangle) {
			index := n - 1 - row
			if index < 0 || index >= n {
				return
			}
			match := m.matches[index]
			marker, lineStyle, matchStyle := " ", dimmedStyle, dimmedMatchStyle
			if index == m.cursor {
				marker, lineStyle, matchStyle = "◆", selectedStyle, titleStyle
			}
			entry := fuzzy_search.HighlightMatched(m.String(match.Index), match, lineStyle, matchStyle)
			dl.AddDraw(r, titleStyle.Render(marker)+" "+entry, render.ZFuzzyOverlay)
		},
		func(row int, _ tea.Mouse) tea.Msg { return itemClickedMsg{index: n - 1 - row} },
	)
	m.listRenderer.RegisterScroll(dl, listBox)
	m.ensureCursorVisible = false

	m.input.SetWidth(inputBox.R.Dx())
	dl.AddDraw(inputBox.R, m.input.View(), render.ZFuzzyOverlay)
	dl.SetCursorInRect(m.input.Cursor(), inputBox.R, 0, 0)
}

func buildPathEntries(rawFileOut []byte) []jj.FileName {
	lines := strings.Split(string(rawFileOut), "\n")
	entries := make([]jj.FileName, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))

	add := func(entry string) {
		if entry == "" {
			return
		}
		if _, ok := seen[entry]; ok {
			return
		}
		seen[entry] = struct{}{}
		entries = append(entries, jj.NewFileName(entry))
	}

	for _, file := range lines {
		if file == "" {
			continue
		}

		// jj repo paths are slash-separated on all platforms.
		// Add each ancestor directory (e.g. "a/b/c.go" adds "a/" then "a/b/").
		for i := 0; i < len(file); i++ {
			if file[i] == '/' {
				add(file[:i+1])
			}
		}

		add(file)
	}

	return entries
}
