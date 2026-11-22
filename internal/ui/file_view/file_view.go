package file_view

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/syntax"
	"github.com/idursun/jjui/internal/ui/theme"
	"github.com/sahilm/fuzzy"
)

var (
	_ common.StackedModel      = (*Model)(nil)
	_ common.ScopeHandler      = (*Model)(nil)
	_ common.SelectionProvider = (*Model)(nil)
	_ common.Editable          = (*Model)(nil)
)

type filesLoadedMsg struct {
	paths []string
	err   error
}

type annotationLoadedMsg struct {
	frame *frame
	lines []line
	err   error
}

// jumpMsg selects a change in the log, adding it to the revset first when
// the revset does not show it.
type jumpMsg struct {
	id string
	// selected is the id as the log prints it, which is what selection matches.
	selected string
	shown    bool
}

type rowClickedMsg struct {
	index int
}

type scrollMsg struct {
	delta      int
	horizontal bool
}

func newScrollMsg(msg tea.MouseMsg) tea.Msg {
	delta, horizontal := render.WheelDelta(msg)
	return scrollMsg{delta: delta, horizontal: horizontal}
}

// frame is the file annotated at one revision. Going back in history pushes
// a frame, so returning restores the cursor and scroll position.
type frame struct {
	revision string
	label    string
	lines    []line
	loading  bool
	err      error
	cursor   int
	// target is the line number the cursor moves to once the lines load.
	target      int
	scroll      int
	highlighted map[int]string
	// maxWidth is the width of the widest line, with tabs expanded.
	maxWidth    int
	gutterWidth int
}

// Model shows one file annotated line by line, with a path line on top for
// picking the file, the way the revset line sits above the revisions.
type Model struct {
	context *context.MainContext
	commit  *jj.Commit
	path    jj.FileName

	paths    []string
	pathsErr error
	input    textinput.Model
	editing  bool
	matches  fuzzy.Matches
	// matchCursor indexes matches, best first.
	matchCursor int

	stack []*frame

	searchInput textinput.Model
	searching   bool
	// query is the lowercased text the lines are searched for.
	query string

	listRenderer        *render.ListRenderer
	ensureCursorVisible bool
	highlighter         *syntax.Highlighter
	scrollX             int
	contentWidth        int
	wrap                bool
}

// New opens the view on path at commit, or on the path line when path is empty.
func New(ctx *context.MainContext, commit *jj.Commit, path jj.FileName) *Model {
	input := textinput.New()
	input.Prompt = ""
	input.SetVirtualCursor(false)
	searchInput := textinput.New()
	searchInput.Prompt = ""
	searchInput.SetVirtualCursor(false)
	m := &Model{
		context:      ctx,
		commit:       commit,
		path:         path,
		input:        input,
		searchInput:  searchInput,
		listRenderer: render.NewListRenderer(newScrollMsg),
		highlighter:  newHighlighter(),
	}
	if path.IsEmpty() {
		m.editing = true
		m.input.Focus()
	} else {
		m.stack = []*frame{m.rootFrame()}
	}
	return m
}

func newHighlighter() *syntax.Highlighter {
	return syntax.New(theme.DefaultPalette.IsDark(), "file_view", theme.DefaultPalette.TerminalPalette())
}

func (m *Model) rootFrame() *frame {
	return &frame{revision: m.commit.CommitId, label: shortId(m.commit.GetChangeId()), loading: true, target: 1}
}

func (m *Model) IsEditing() bool {
	return m.editing || m.searching
}

func (m *Model) Init() tea.Cmd {
	runner := m.context.RunCommandImmediate
	args := jj.FilesInRevision(m.commit)
	loadFiles := func() tea.Msg {
		out, err := runner(args)
		if err != nil {
			return filesLoadedMsg{err: err}
		}
		return filesLoadedMsg{paths: strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")}
	}
	cmds := []tea.Cmd{loadFiles}
	if f := m.current(); f != nil {
		cmds = append(cmds, m.load(f))
	}
	if m.editing {
		cmds = append(cmds, textinput.Blink)
	}
	return tea.Batch(cmds...)
}

func (m *Model) load(f *frame) tea.Cmd {
	runner := m.context.RunCommandImmediate
	args := jj.FileAnnotate(f.revision, m.path)
	return func() tea.Msg {
		out, err := runner(args)
		if err != nil {
			return annotationLoadedMsg{frame: f, err: err}
		}
		return annotationLoadedMsg{frame: f, lines: parseAnnotation(out)}
	}
}

func (m *Model) Scopes() []common.Scope {
	if m.editing {
		return []common.Scope{{Name: actions.ScopeFileViewPath, Leak: common.LeakNone, Handler: m}}
	}
	if m.searching {
		return []common.Scope{{Name: actions.ScopeFileViewSearchInput, Leak: common.LeakNone, Handler: m}}
	}
	return []common.Scope{{Name: actions.ScopeFileView, Leak: common.LeakAll, Handler: m}}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
	case filesLoadedMsg:
		m.paths, m.pathsErr = msg.paths, msg.err
		m.search()
	case jumpMsg:
		// The log only receives refreshes once this view has closed.
		if msg.shown {
			return tea.Sequence(common.Close, common.RefreshAndSelect(msg.selected))
		}
		widened := "(" + m.context.CurrentRevset + ") | " + msg.id
		return tea.Sequence(common.Close, common.UpdateRevSet(widened), common.RefreshAndSelect(msg.selected))
	case annotationLoadedMsg:
		f := msg.frame
		f.loading, f.lines, f.err = false, msg.lines, msg.err
		for _, l := range f.lines {
			f.maxWidth = max(f.maxWidth, ansi.StringWidth(expandTabs(l.content)))
			f.gutterWidth = max(f.gutterWidth, ansi.StringWidth(l.gutter))
		}
		f.cursor = min(max(f.target-1, 0), max(len(f.lines)-1, 0))
		if f == m.current() {
			m.ensureCursorVisible = true
		}
	case rowClickedMsg:
		if m.editing {
			m.matchCursor = msg.index
			return m.accept()
		}
		if f := m.current(); f != nil && msg.index < len(f.lines) {
			f.cursor = msg.index
		}
	case scrollMsg:
		if msg.horizontal {
			m.scrollHorizontal(msg.delta)
			return nil
		}
		m.ensureCursorVisible = false
		m.listRenderer.SetScrollOffset(max(m.listRenderer.GetScrollOffset()+msg.delta, 0))
	case common.ThemeChangedMsg:
		m.highlighter = newHighlighter()
		for _, f := range m.stack {
			f.highlighted = nil
		}
	case tea.KeyMsg, tea.PasteMsg:
		if m.searching {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return cmd
		}
		if !m.editing {
			return nil
		}
		previous := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() != previous {
			m.search()
		}
		return cmd
	}
	return nil
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.FileViewNavigate:
		m.navigate(intent)
		return nil, true
	case intents.FileViewScrollHorizontal:
		m.scrollHorizontal(intent.Delta)
		return nil, true
	case intents.FileViewToggleWrap:
		m.wrap = !m.wrap
		m.scrollX = 0
		m.ensureCursorVisible = true
		return nil, true
	case intents.FileViewMoveChange:
		m.moveChange(intent.Delta)
		return nil, true
	case intents.FileViewOlder:
		return m.older(), true
	case intents.FileViewNewer:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			m.listRenderer.SetScrollOffset(m.current().scroll)
		}
		return nil, true
	case intents.FileViewSearch:
		m.searching = true
		m.searchInput.SetValue("")
		m.searchInput.Placeholder = m.query
		return m.searchInput.Focus(), true
	case intents.FileViewSearchApply:
		m.searching = false
		m.searchInput.Blur()
		if query := strings.ToLower(m.searchInput.Value()); query != "" {
			m.query = query
		}
		m.searchFrom(0, false)
		return nil, true
	case intents.FileViewSearchCancel:
		m.searching = false
		m.searchInput.Blur()
		return nil, true
	case intents.QuickSearchCycle:
		offset := 1
		if intent.Reverse {
			offset = -1
		}
		m.searchFrom(offset, intent.Reverse)
		return nil, true
	case intents.FileViewJumpToChange:
		l := m.currentLine()
		if l == nil {
			return nil, true
		}
		return m.jump(*l), true
	case intents.FileViewFilterRevset:
		if m.path.IsEmpty() {
			return nil, true
		}
		files := fmt.Sprintf("files(%s)", m.path.Escaped())
		return tea.Batch(common.UpdateRevSet(m.context.NarrowedRevset(files)), common.Close), true
	case intents.FileViewEditPath:
		m.editing = true
		m.input.SetValue("")
		m.input.Placeholder = m.path.Path()
		m.search()
		return m.input.Focus(), true
	case intents.FileViewClose:
		if m.query != "" {
			m.query = ""
			return nil, true
		}
		return common.Close, true
	case intents.FileViewPathAccept:
		return m.accept(), true
	case intents.FileViewPathCancel:
		if m.path.IsEmpty() {
			return common.Close, true
		}
		m.editing = false
		m.input.Blur()
		m.ensureCursorVisible = true
		return nil, true
	}
	return nil, false
}

// jump checks whether the current revset shows the line's change. If the
// check fails, it selects the change without widening.
func (m *Model) jump(l line) tea.Cmd {
	id := l.id()
	if id == "" {
		return nil
	}
	runner := m.context.RunCommandImmediate
	shown := jj.GetIdsFromRevset("(" + m.context.CurrentRevset + ") & " + id)
	return func() tea.Msg {
		out, err := runner(shown)
		if err != nil {
			return jumpMsg{id: id, selected: id, shown: true}
		}
		if selected := strings.TrimSpace(string(out)); selected != "" {
			return jumpMsg{id: id, selected: selected, shown: true}
		}
		out, err = runner(jj.GetIdsFromRevset(id))
		selected := strings.TrimSpace(string(out))
		if err != nil || selected == "" {
			selected = id
		}
		return jumpMsg{id: id, selected: selected}
	}
}

// accept opens the highlighted match, annotated at the revision the view
// opened from.
func (m *Model) accept() tea.Cmd {
	path, ok := m.selectedMatch()
	if !ok {
		return nil
	}
	m.editing = false
	m.input.Blur()
	if path == m.path.Path() && len(m.stack) > 0 {
		return nil
	}
	m.path = jj.NewFileName(path)
	root := m.rootFrame()
	m.stack = []*frame{root}
	m.listRenderer.SetScrollOffset(0)
	return m.load(root)
}

// older annotates the file at the first parent of the change under the
// cursor, with the cursor on the same line number.
func (m *Model) older() tea.Cmd {
	l := m.currentLine()
	if l == nil || l.id() == "" {
		return nil
	}
	current := m.current()
	current.scroll = m.listRenderer.GetScrollOffset()
	f := &frame{
		revision: "first_parent(" + l.id() + ")",
		label:    "before " + shortId(l.id()),
		loading:  true,
		target:   l.number,
	}
	m.stack = append(m.stack, f)
	m.listRenderer.SetScrollOffset(0)
	return m.load(f)
}

func (m *Model) search() {
	m.matchCursor = 0
	source := &fuzzy_search.RefinedSource{Source: pathSource(m.paths)}
	m.matches = source.Search(m.input.Value(), len(m.paths))
	m.ensureCursorVisible = true
}

type pathSource []string

func (s pathSource) Len() int            { return len(s) }
func (s pathSource) String(i int) string { return s[i] }

func (m *Model) selectedMatch() (string, bool) {
	if m.matchCursor < 0 || m.matchCursor >= len(m.matches) {
		return "", false
	}
	return m.paths[m.matches[m.matchCursor].Index], true
}

func (m *Model) current() *frame {
	if len(m.stack) == 0 {
		return nil
	}
	return m.stack[len(m.stack)-1]
}

func (m *Model) currentLine() *line {
	f := m.current()
	if f == nil || f.cursor < 0 || f.cursor >= len(f.lines) {
		return nil
	}
	return &f.lines[f.cursor]
}

func (m *Model) navigate(intent intents.FileViewNavigate) {
	delta := intent.Delta
	switch {
	case intent.ToEnd:
		delta *= math.MaxInt32
	case intent.IsPage, intent.IsHalfPage:
		span := max(m.listRenderer.GetLastRowIndex()-m.listRenderer.GetFirstRowIndex()-1, 1)
		if intent.IsHalfPage {
			span = max(span/2, 1)
		}
		if delta < 0 {
			delta = -span
		} else {
			delta = span
		}
	}
	if m.editing {
		if len(m.matches) > 0 {
			m.matchCursor = min(max(m.matchCursor+delta, 0), len(m.matches)-1)
			m.ensureCursorVisible = true
		}
		return
	}
	if f := m.current(); f != nil && len(f.lines) > 0 {
		f.cursor = min(max(f.cursor+delta, 0), len(f.lines)-1)
		m.ensureCursorVisible = true
	}
}

// searchFrom moves the cursor to the first line containing the query,
// starting offset lines from the cursor and wrapping around the file.
func (m *Model) searchFrom(offset int, backward bool) {
	f := m.current()
	if m.query == "" || f == nil || len(f.lines) == 0 {
		return
	}
	n := len(f.lines)
	for i := range n {
		step := i
		if backward {
			step = -i
		}
		index := ((f.cursor+offset+step)%n + n) % n
		if strings.Contains(strings.ToLower(expandTabs(f.lines[index].content)), m.query) {
			f.cursor = index
			m.ensureCursorVisible = true
			return
		}
	}
}

func (m *Model) scrollHorizontal(delta int) {
	f := m.current()
	if m.editing || m.wrap || f == nil {
		return
	}
	m.scrollX = min(max(m.scrollX+delta, 0), max(f.maxWidth-m.contentWidth, 0))
}

// moveChange moves to the first line of the next run of lines from one
// change, or back to the start of the current or previous run.
func (m *Model) moveChange(delta int) {
	f := m.current()
	if m.editing || f == nil || len(f.lines) == 0 {
		return
	}
	i := f.cursor
	if delta > 0 {
		for i++; i < len(f.lines) && !f.lines[i].first; i++ {
		}
		if i >= len(f.lines) {
			return
		}
	} else {
		if i > 0 && f.lines[i].first {
			i--
		}
		for i > 0 && !f.lines[i].first {
			i--
		}
	}
	f.cursor = i
	m.ensureCursorVisible = true
}

func (m *Model) Selection() common.SelectionSnapshot {
	if m.editing {
		path, ok := m.selectedMatch()
		if !ok {
			return common.SelectionSnapshot{}
		}
		return common.SelectionSnapshot{Highlighted: common.SelectedFile{
			ChangeId: m.commit.GetChangeId(), CommitId: m.commit.CommitId, File: jj.NewFileName(path),
		}}
	}
	l := m.currentLine()
	if l == nil {
		return common.SelectionSnapshot{}
	}
	return common.SelectionSnapshot{Highlighted: common.SelectedFile{
		ChangeId: l.changeId, CommitId: l.commitId, File: m.path,
	}}
}

func expandTabs(s string) string {
	return strings.ReplaceAll(s, "\t", "    ")
}

func shortId(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	titleStyle := theme.DefaultPalette.Get("file_view", "", "title", false)
	textStyle := theme.DefaultPalette.Get("file_view", "", "text", false)
	dimmedStyle := theme.DefaultPalette.Get("file_view", "", "dimmed", false)

	pathBox, body := box.CutTop(1)
	tb := dl.Text(pathBox.R.Min.X, pathBox.R.Min.Y, 0).Styled("file: ", titleStyle)
	if m.editing {
		x, _ := tb.Measure()
		inputRect := layout.Rect(pathBox.R.Min.X+x, pathBox.R.Min.Y, max(pathBox.R.Dx()-x, 0), 1)
		m.input.SetWidth(inputRect.Dx())
		tb.Done()
		dl.AddDraw(inputRect, m.input.View(), 0)
		dl.SetCursorInRect(m.input.Cursor(), inputRect, 0, 0)
		m.renderMatches(dl, body)
		return
	}
	f := m.current()
	if m.searching {
		tb.Done()
		tb = dl.Text(pathBox.R.Min.X, pathBox.R.Min.Y, 0).Styled("search: ", titleStyle)
		x, _ := tb.Measure()
		inputRect := layout.Rect(pathBox.R.Min.X+x, pathBox.R.Min.Y, max(pathBox.R.Dx()-x, 0), 1)
		m.searchInput.SetWidth(inputRect.Dx())
		tb.Done()
		dl.AddDraw(inputRect, m.searchInput.View(), 0)
		dl.SetCursorInRect(m.searchInput.Cursor(), inputRect, 0, 0)
	} else {
		tb.Styled(m.path.Path(), textStyle).Styled("  at "+f.label, dimmedStyle)
		if len(m.stack) > 1 {
			tb.Styled("  ("+strconv.Itoa(len(m.stack)-1)+" back)", dimmedStyle)
		}
		if m.query != "" {
			tb.Styled("  search: "+m.query, dimmedStyle)
		}
		tb.Done()
	}

	switch {
	case f.loading:
		dl.AddDraw(body.R, lipgloss.Place(body.R.Dx(), body.R.Dy(), lipgloss.Center, lipgloss.Center, "loading"), 0)
	case f.err != nil:
		dl.AddDraw(body.R, f.err.Error(), 0)
	default:
		m.renderLines(dl, body, f)
	}
}

func (m *Model) renderMatches(dl *render.DisplayContext, box layout.Box) {
	textStyle := theme.DefaultPalette.Get("file_view", "", "text", false)
	matchedStyle := theme.DefaultPalette.Get("file_view", "", "matched", false)
	selectedStyle := theme.DefaultPalette.GetBlended("file_view", "", "", true)
	if m.pathsErr != nil {
		dl.AddDraw(box.R, m.pathsErr.Error(), 0)
		return
	}
	m.listRenderer.Render(
		dl,
		box,
		len(m.matches),
		m.matchCursor,
		m.ensureCursorVisible,
		func(int) int { return 1 },
		func(dl *render.DisplayContext, index int, r layout.Rectangle) {
			match := m.matches[index]
			style, matchStyle := textStyle, matchedStyle
			if index == m.matchCursor {
				style, matchStyle = selectedStyle, matchedStyle.Inherit(selectedStyle)
			}
			entry := fuzzy_search.HighlightMatched(m.paths[match.Index], match, style, matchStyle)
			dl.AddDraw(r, lipgloss.PlaceHorizontal(r.Dx(), 0, entry, lipgloss.WithWhitespaceStyle(style)), 0)
		},
		func(index int, _ tea.Mouse) render.ClickMessage { return rowClickedMsg{index: index} },
	)
	m.listRenderer.RegisterScroll(dl, box)
	m.ensureCursorVisible = false
}

func (m *Model) renderLines(dl *render.DisplayContext, box layout.Box, f *frame) {
	selectedStyle := theme.DefaultPalette.GetBlended("file_view", "", "", true)
	matchedStyle := theme.DefaultPalette.Get("file_view", "search", "matched", false)

	prefixWidth := min(f.gutterWidth, box.R.Dx()/2)
	m.contentWidth = max(box.R.Dx()-prefixWidth, 1)
	m.scrollX = min(m.scrollX, max(f.maxWidth-m.contentWidth, 0))
	if f.highlighted == nil {
		f.highlighted = make(map[int]string)
	}
	height := func(int) int { return 1 }
	if m.wrap {
		height = func(index int) int {
			width := ansi.StringWidth(expandTabs(f.lines[index].content))
			return max((width+m.contentWidth-1)/m.contentWidth, 1)
		}
	}
	m.listRenderer.Render(
		dl,
		box,
		len(f.lines),
		f.cursor,
		m.ensureCursorVisible,
		height,
		func(dl *render.DisplayContext, index int, r layout.Rectangle) {
			l := f.lines[index]
			gutter := lipgloss.PlaceHorizontal(prefixWidth, 0, ansi.Truncate(l.gutter, prefixWidth, ""))
			content, ok := f.highlighted[index]
			if !ok {
				content = m.highlighter.Highlight(m.path.Path(), expandTabs(l.content))
				f.highlighted[index] = content
			}
			dl.AddDraw(layout.Rect(r.Min.X, r.Min.Y, r.Dx(), 1), gutter, 0)
			for i, segment := range m.segments(content) {
				if i >= r.Dy() {
					break
				}
				dl.AddDraw(layout.Rect(r.Min.X+prefixWidth, r.Min.Y+i, m.contentWidth, 1), segment, 0)
			}
			m.markMatches(dl, layout.Rect(r.Min.X+prefixWidth, r.Min.Y, m.contentWidth, r.Dy()), expandTabs(l.content), matchedStyle)
			if index == f.cursor {
				dl.AddPaint(r, selectedStyle, 0)
			}
		},
		func(index int, _ tea.Mouse) render.ClickMessage { return rowClickedMsg{index: index} },
	)
	m.listRenderer.RegisterScroll(dl, box)
	m.ensureCursorVisible = false
}

// markMatches styles each match of the query in a line drawn into rect,
// following the horizontal scroll or the wrapped rows.
func (m *Model) markMatches(dl *render.DisplayContext, rect layout.Rectangle, content string, style lipgloss.Style) {
	for _, match := range common.MatchRanges(content, m.query) {
		start, end := ansi.StringWidth(content[:match[0]]), ansi.StringWidth(content[:match[1]])
		if !m.wrap {
			start, end = max(start-m.scrollX, 0), min(end-m.scrollX, m.contentWidth)
			if start < end {
				dl.AddAttributes(layout.Rect(rect.Min.X+start, rect.Min.Y, end-start, 1), style, 1)
			}
			continue
		}
		for col := start; col < end; {
			row, x := col/m.contentWidth, col%m.contentWidth
			width := min(end-col, m.contentWidth-x)
			if row < rect.Dy() {
				dl.AddAttributes(layout.Rect(rect.Min.X+x, rect.Min.Y+row, width, 1), style, 1)
			}
			col += width
		}
	}
}

// segments cuts highlighted content into the rows it is drawn on: one row
// scrolled by scrollX, or one row per contentWidth columns when wrapping.
func (m *Model) segments(content string) []string {
	if !m.wrap {
		return []string{ansi.Cut(content, m.scrollX, m.scrollX+m.contentWidth)}
	}
	width := ansi.StringWidth(content)
	var rows []string
	for start := 0; start < width; start += m.contentWidth {
		rows = append(rows, ansi.Cut(content, start, min(start+m.contentWidth, width)))
	}
	return rows
}
