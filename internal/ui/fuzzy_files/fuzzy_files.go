package fuzzy_files

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/sahilm/fuzzy"
)

type fuzzyFiles struct {
	keyMap  config.KeyMappings[key.Binding]
	inputKm textinput.KeyMap

	// restore
	revset          string
	commit          *jj.Commit
	wasPreviewShown bool

	cursor int
	// enabled with ctrl+t again
	// live preview of revset and rev-diff
	revsetPreview bool
	debounceTag   int

	// search state
	files   []string
	max     int
	matches fuzzy.Matches
	styles  fuzzy_search.Styles
}

var debounceDuration = 250 * time.Millisecond

type debouncePreview int

type initMsg struct{}

func newCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func (fzf *fuzzyFiles) Init() tea.Cmd {
	return newCmd(initMsg{})
}

func (fzf *fuzzyFiles) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		return fzf.handleIntent(msg)
	case initMsg:
		fzf.search("")
	case fuzzy_search.SearchMsg:
		if cmd := fzf.handleKey(msg.Pressed); cmd != nil {
			return cmd
		}
		fzf.search(msg.Input)
		if fzf.revsetPreview {
			fzf.debounceTag++
			tag := debouncePreview(fzf.debounceTag)
			return tea.Tick(debounceDuration, func(_ time.Time) tea.Msg {
				return tag
			})
		}
	case debouncePreview:
		if int(msg) != fzf.debounceTag {
			return nil
		}
		if fzf.revsetPreview {
			return tea.Batch(
				fzf.updateRevSet(),
				newCmd(common.ShowPreview(true)),
			)
		}
	case tea.KeyMsg:
		return fzf.handleKey(msg)
	}
	return nil
}

func (fzf *fuzzyFiles) updateRevSet() tea.Cmd {
	path := fuzzy_search.SelectedMatch(fzf)
	revset := fzf.revset
	if len(path) > 0 {
		revset = fmt.Sprintf("files(%s)", path)
	}
	return common.UpdateRevSet(revset)
}

func (fzf *fuzzyFiles) isInputMovement(k tea.KeyMsg) bool {
	return key.Matches(k,
		fzf.inputKm.CharacterForward,
		fzf.inputKm.CharacterBackward,
		fzf.inputKm.WordForward,
		fzf.inputKm.WordBackward,
		fzf.inputKm.LineStart,
		fzf.inputKm.LineEnd,
		fzf.inputKm.AcceptSuggestion,
	)
}

func skipSearch() tea.Msg {
	return nil
}

func (fzf *fuzzyFiles) handleKey(msg tea.KeyMsg) tea.Cmd {
	fzfKm := fzf.keyMap.FileSearch
	previewKm := fzf.keyMap.Preview
	if fzf.revsetPreview {
		switch {
		case key.Matches(msg, fzfKm.Up, fzfKm.Down):
			return fzf.handleIntent(intents.FileSearchRevisionNavigate{Key: msg})
		case key.Matches(msg, previewKm.ScrollUp):
			return fzf.handleIntent(intents.FileSearchPreviewScroll{Kind: intents.PreviewScrollUp})
		case key.Matches(msg, previewKm.ScrollDown):
			return fzf.handleIntent(intents.FileSearchPreviewScroll{Kind: intents.PreviewScrollDown})
		case key.Matches(msg, previewKm.HalfPageUp):
			return fzf.handleIntent(intents.FileSearchPreviewScroll{Kind: intents.PreviewHalfPageUp})
		case key.Matches(msg, previewKm.HalfPageDown):
			return fzf.handleIntent(intents.FileSearchPreviewScroll{Kind: intents.PreviewHalfPageDown})
		}
	} else {
		switch {
		case key.Matches(msg, fzfKm.Up, previewKm.ScrollUp):
			return fzf.handleIntent(intents.FileSearchNavigate{Delta: 1})
		case key.Matches(msg, fzfKm.Down, previewKm.ScrollDown):
			return fzf.handleIntent(intents.FileSearchNavigate{Delta: -1})
		}
	}

	switch {
	case key.Matches(msg, fzf.keyMap.Cancel):
		return fzf.handleIntent(intents.FileSearchCancel{})
	case key.Matches(msg, fzfKm.Edit):
		return fzf.handleIntent(intents.FileSearchEdit{})
	case key.Matches(msg, fzfKm.Toggle):
		return fzf.handleIntent(intents.FileSearchTogglePreview{})
	case key.Matches(msg, fzfKm.Accept, fzf.inputKm.AcceptSuggestion):
		return fzf.handleIntent(intents.FileSearchAccept{})
	case fzf.isInputMovement(msg):
		return skipSearch
	}

	return nil
}

func (fzf *fuzzyFiles) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent := intent.(type) {
	case intents.FileSearchNavigate:
		fzf.moveCursor(intent.Delta)
		return skipSearch
	case intents.FileSearchCancel:
		return tea.Batch(
			common.UpdateRevSet(fzf.revset),
			newCmd(common.ShowPreview(fzf.wasPreviewShown)),
		)
	case intents.FileSearchEdit:
		path := fuzzy_search.SelectedMatch(fzf)
		return newCmd(common.ExecMsg{
			Line: config.GetDefaultEditor() + " " + path,
			Mode: common.ExecShell,
		})
	case intents.FileSearchTogglePreview:
		fzf.revsetPreview = !fzf.revsetPreview
		return tea.Batch(
			newCmd(common.ShowPreview(fzf.revsetPreview)),
			fzf.updateRevSet(),
		)
	case intents.FileSearchAccept:
		return fzf.updateRevSet()
	case intents.FileSearchPreviewScroll:
		// Dispatch to ui.go which handles preview scroll intents
		return func() tea.Msg {
			return intents.PreviewScroll{Kind: intent.Kind}
		}
	case intents.FileSearchRevisionNavigate:
		return revisions.RevisionsCmd(intent.Key)
	}
	return nil
}

func (fzf *fuzzyFiles) moveCursor(inc int) {
	n := fzf.cursor + inc
	l := len(fzf.matches) - 1
	if n > l {
		n = 0
	}
	if n < 0 {
		n = l
	}
	fzf.cursor = n
}

func (fzf *fuzzyFiles) Styles() fuzzy_search.Styles {
	return fzf.styles
}

func (fzf *fuzzyFiles) Max() int {
	return fzf.max
}

func (fzf *fuzzyFiles) Matches() fuzzy.Matches {
	return fzf.matches
}

func (fzf *fuzzyFiles) SelectedMatch() int {
	return fzf.cursor
}

func (fzf *fuzzyFiles) Len() int {
	return len(fzf.files)
}

func (fzf *fuzzyFiles) String(i int) string {
	n := len(fzf.files)
	if i < 0 || i >= n {
		return ""
	}
	return fzf.files[i]
}

func (fzf *fuzzyFiles) search(input string) {
	src := &fuzzy_search.RefinedSource{Source: fzf}
	fzf.cursor = 0
	fzf.matches = src.Search(input, fzf.max)
}

func (fzf *fuzzyFiles) ViewRect(dl *render.DisplayContext, box layout.Box) {
	content := fzf.viewContent()
	if content == "" {
		return
	}
	_, h := lipgloss.Size(content)
	rect := cellbuf.Rect(box.R.Min.X, box.R.Max.Y-h, box.R.Dx(), h)
	dl.AddDraw(rect, content, 1)
}

func (fzf *fuzzyFiles) viewContent() string {
	shown := len(fzf.matches)
	if shown == 0 {
		return ""
	}
	title := fzf.styles.SelectedMatch.Render(
		"  ",
		strconv.Itoa(shown),
		"of",
		strconv.Itoa(len(fzf.files)),
		"files present at revision",
		fzf.commit.GetChangeId(),
		" ",
	)
	entries := fuzzy_search.View(fzf)
	return lipgloss.JoinVertical(0, title, entries)
}

func joinBindings(help string, a key.Binding, b key.Binding) key.Binding {
	keys := append(a.Keys(), b.Keys()...)
	joined := config.JoinKeys(keys)
	return key.NewBinding(
		key.WithKeys(keys...),
		key.WithHelp(joined, help),
	)
}

func (fzf *fuzzyFiles) ShortHelp() []key.Binding {
	short_help := []key.Binding{fzf.keyMap.FileSearch.Edit}
	toggle := fzf.keyMap.FileSearch.Toggle.Keys()[0]
	if fzf.revsetPreview {
		short_help = append(short_help,
			// we join some bindings to take less space and help of toggle depending on value
			key.NewBinding(key.WithKeys(toggle), key.WithHelp(toggle, "preview off")),
			joinBindings("move on revset", fzf.keyMap.FileSearch.Up, fzf.keyMap.FileSearch.Down),
			joinBindings("scroll preview", fzf.keyMap.Preview.ScrollUp, fzf.keyMap.Preview.ScrollDown),
		)
	} else {
		short_help = append(short_help,
			key.NewBinding(key.WithKeys(toggle), key.WithHelp(toggle, "preview on")),
			fzf.keyMap.FileSearch.Accept,
		)
	}

	return short_help
}

func (fzf *fuzzyFiles) FullHelp() [][]key.Binding {
	return [][]key.Binding{fzf.ShortHelp()}
}

type editStatus func() (help.KeyMap, string)

func (fzf *fuzzyFiles) editStatus() (help.KeyMap, string) {
	return fzf, ""
}

func NewModel(msg common.FileSearchMsg) (fuzzy_search.Model, editStatus) {
	keyMap := config.Current.GetKeyMap()
	inputKm := textinput.DefaultKeyMap
	model := &fuzzyFiles{
		keyMap:          keyMap,
		inputKm:         inputKm,
		revset:          msg.Revset,
		wasPreviewShown: msg.PreviewShown,
		max:             30,
		commit:          msg.Commit,
		files:           strings.Split(string(msg.RawFileOut), "\n"),
		styles:          fuzzy_search.NewStyles(),
	}
	return model, model.editStatus
}
