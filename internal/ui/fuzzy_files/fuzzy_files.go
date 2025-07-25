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
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/sahilm/fuzzy"
)

type fuzzyFiles struct {
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

func (fzf *fuzzyFiles) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initMsg:
		fzf.search("")
	case fuzzy_search.SearchMsg:
		if cmd := fzf.handleKey(msg.Pressed); cmd != nil {
			return fzf, cmd
		}
		fzf.search(msg.Input)
		if fzf.revsetPreview {
			fzf.debounceTag++
			tag := debouncePreview(fzf.debounceTag)
			return fzf, tea.Tick(debounceDuration, func(_ time.Time) tea.Msg {
				return tag
			})
		}
	case debouncePreview:
		if int(msg) != fzf.debounceTag {
			return fzf, nil
		}
		if fzf.revsetPreview {
			return fzf, tea.Batch(
				fzf.updateRevSet(),
				newCmd(common.ShowPreview(true)),
			)
		}
	case tea.KeyMsg:
		return fzf, fzf.handleKey(msg)
	}
	return fzf, nil
}

func (fzf *fuzzyFiles) updateRevSet() tea.Cmd {
	path := fuzzy_search.SelectedMatch(fzf)
	revset := fzf.revset
	if len(path) > 0 {
		revset = fmt.Sprintf("files(\"%s\")", path)
	}
	return common.UpdateRevSet(revset)
}

var km = config.Current.GetKeyMap()
var inputKm = textinput.DefaultKeyMap
var up = key.NewBinding(key.WithKeys("up"))
var down = key.NewBinding(key.WithKeys("down"))
var enter = key.NewBinding(key.WithKeys("enter"))
var edit = key.NewBinding(key.WithKeys("alt+e"))

func isInputMovement(k tea.KeyMsg) bool {
	return key.Matches(k,
		inputKm.CharacterForward,
		inputKm.CharacterBackward,
		inputKm.WordForward,
		inputKm.WordBackward,
		inputKm.LineStart,
		inputKm.LineEnd,
		inputKm.AcceptSuggestion,
	)
}

func skipSearch() tea.Msg {
	return nil
}

func (fzf *fuzzyFiles) handleKey(msg tea.KeyMsg) tea.Cmd {
	if fzf.revsetPreview {
		switch {
		case key.Matches(msg, up, down):
			return revisions.RevisionsCmd(msg)
		case key.Matches(msg, km.Preview.ScrollUp, km.Preview.ScrollDown, km.Preview.HalfPageUp, km.Preview.HalfPageDown):
			return preview.PreviewCmd(msg)
		}
	} else {
		switch {
		case key.Matches(msg, up, km.Preview.ScrollUp):
			fzf.moveCursor(1)
			return skipSearch
		case key.Matches(msg, down, km.Preview.ScrollDown):
			fzf.moveCursor(-1)
			return skipSearch
		}
	}

	switch {
	case key.Matches(msg, km.Cancel):
		return tea.Batch(
			common.UpdateRevSet(fzf.revset),
			newCmd(common.ShowPreview(fzf.wasPreviewShown)),
		)
	case key.Matches(msg, edit):
		path := fuzzy_search.SelectedMatch(fzf)
		return newCmd(common.ExecMsg{
			Line: config.GetDefaultEditor() + " " + path,
			Mode: common.ExecShell,
		})
	case key.Matches(msg, km.FileSearch):
		fzf.revsetPreview = !fzf.revsetPreview
		return tea.Batch(
			newCmd(common.ShowPreview(fzf.revsetPreview)),
			fzf.updateRevSet(),
		)
	case key.Matches(msg, enter, inputKm.AcceptSuggestion):
		return fzf.updateRevSet()
	case isInputMovement(msg):
		return skipSearch
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

func (fzf *fuzzyFiles) View() string {
	shown := len(fzf.matches)
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

func (fzf *fuzzyFiles) ShortHelp() []key.Binding {
	keys := []key.Binding{}
	addKey := func(ks string, help string) {
		keys = append(keys, key.NewBinding(key.WithKeys(ks), key.WithHelp(ks, help)))
	}

	addKey(km.FileSearch.Keys()[0], "live mode")
	addKey("alt+e", "edit file")

	if fzf.revsetPreview {
		addKey("up/down", "move on revset")
		addKey("ctrl+n/ctrl+p", "scroll preview")
	} else {
		addKey("enter", "file revset")
	}

	return keys
}

func (fzf *fuzzyFiles) FullHelp() [][]key.Binding {
	return [][]key.Binding{fzf.ShortHelp()}
}

type editStatus func() (help.KeyMap, string)

func (fzf *fuzzyFiles) editStatus() (help.KeyMap, string) {
	return fzf, ""
}

func NewModel(msg common.FileSearchMsg) (fuzzy_search.Model, editStatus) {
	model := &fuzzyFiles{
		revset:          msg.Revset,
		wasPreviewShown: msg.PreviewShown,
		max:             30,
		commit:          msg.Commit,
		files:           strings.Split(string(msg.RawFileOut), "\n"),
		styles:          fuzzy_search.NewStyles(),
	}
	return model, model.editStatus
}
