package fuzzy_files

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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

func skipSearch() tea.Msg {
	return nil
}

func (fzf *fuzzyFiles) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent := intent.(type) {
	case intents.FileSearchNavigate:
		if fzf.revsetPreview {
			return revisions.RevisionsCmd(intents.Navigate{Delta: fileSearchNavigateDelta(intent.Delta)})
		}
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
		if !fzf.revsetPreview {
			switch intent.Kind {
			case intents.PreviewScrollUp:
				fzf.moveCursor(1)
				return skipSearch
			case intents.PreviewScrollDown:
				fzf.moveCursor(-1)
				return skipSearch
			default:
				return nil
			}
		}
		// Dispatch to ui.go which handles preview scroll intents
		return func() tea.Msg {
			return intents.PreviewScroll{Kind: intent.Kind}
		}
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
	dl.AddDraw(rect, content, render.ZFuzzyInput)
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

func NewModel(msg common.FileSearchMsg) fuzzy_search.Model {
	model := &fuzzyFiles{
		revset:          msg.Revset,
		wasPreviewShown: msg.PreviewShown,
		max:             30,
		commit:          msg.Commit,
		files:           strings.Split(string(msg.RawFileOut), "\n"),
		styles:          fuzzy_search.NewStyles(),
	}
	return model
}

func fileSearchNavigateDelta(delta int) int {
	if delta < 0 {
		return 1
	}
	return -1
}
