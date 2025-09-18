package file_search

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/exec_prompt"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/idursun/jjui/internal/ui/view"
)

var (
	_ view.IViewModel = (*FuzzyFilesModel)(nil)
)

type FuzzyFilesModel struct {
	*view.ViewNode
	context *context.MainContext
	keymap  config.KeyMappings[key.Binding]
	input   textinput.Model

	// restore
	revset          string
	commit          *models.Commit
	wasPreviewShown bool

	// enabled with ctrl+t again
	// live preview of revset and rev-diff
	revsetPreview bool
	debounceTag   int

	// search state
	max       int
	styles    fuzzy_search.Styles
	fuzzyView *fuzzy_search.Model
}

func (f *FuzzyFilesModel) GetId() view.ViewId {
	return "fuzzy files"
}

func (f *FuzzyFilesModel) Mount(v *view.ViewNode) {
	v.Id = f.GetId()
	f.ViewNode = v
	f.input.Width = v.Parent.Width - 20
}

func (f *FuzzyFilesModel) Init() tea.Cmd {
	f.fuzzyView.Search("")
	return f.input.Focus()
}

func (f *FuzzyFilesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		fzfKm := f.keymap.FileSearch
		previewKm := f.keymap.Preview

		switch {
		case key.Matches(msg, fzfKm.Up, previewKm.ScrollUp):
			f.fuzzyView.MoveCursor(1)
			return f, nil
		case key.Matches(msg, fzfKm.Down, previewKm.ScrollDown):
			f.fuzzyView.MoveCursor(-1)
			return f, nil
		}

		switch {
		case key.Matches(msg, f.keymap.Cancel):
			f.ViewManager.StopEditing()
			f.ViewManager.UnregisterView(f.GetId())
			return f, nil
		case key.Matches(msg, fzfKm.Edit):
			path := f.fuzzyView.SelectedMatch()
			return f, exec_prompt.ExecLine(f.context, common.ExecShell, config.GetDefaultEditor()+" "+path)
		//case key.Matches(msg, fzfKm.Toggle):
		//	f.revsetPreview = !f.revsetPreview
		//	return tea.Batch(
		//		newCmd(common.ShowPreview(f.revsetPreview)),
		//		f.updateRevSet(),
		//	)
		case key.Matches(msg, fzfKm.Accept):
			return f, f.updateRevSet()
		}

		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		f.fuzzyView.Search(f.input.Value())
		return f, cmd
	}
	return f, nil
}

func (f *FuzzyFilesModel) updateRevSet() tea.Cmd {
	path := f.fuzzyView.SelectedMatch()
	revset := f.revset
	if len(path) > 0 {
		revset = fmt.Sprintf("files(\"%s\")", path)
	}
	f.context.CurrentRevset = revset
	f.ViewManager.UnregisterView(f.GetId())
	f.ViewManager.StopEditing()
	return common.Refresh
}

func (f *FuzzyFilesModel) View() string {
	shown := len(f.fuzzyView.Matches)
	title := f.styles.SelectedMatch.Render(
		"  ",
		strconv.Itoa(shown),
		"of",
		strconv.Itoa(f.fuzzyView.Source.Len()),
		"files present at revision",
		f.commit.GetChangeId(),
		" ",
	)
	entries := f.fuzzyView.View()
	inputView := f.input.View()
	return lipgloss.JoinVertical(0, title, entries, inputView)
}

func joinBindings(help string, a key.Binding, b key.Binding) key.Binding {
	keys := append(a.Keys(), b.Keys()...)
	joined := config.JoinKeys(keys)
	return key.NewBinding(
		key.WithKeys(keys...),
		key.WithHelp(joined, help),
	)
}

func (f *FuzzyFilesModel) ShortHelp() []key.Binding {
	shortHelp := []key.Binding{f.keymap.FileSearch.Edit}
	toggle := f.keymap.FileSearch.Toggle.Keys()[0]
	if f.revsetPreview {
		shortHelp = append(shortHelp,
			// we join some bindings to take less space and help of toggle depending on value
			key.NewBinding(key.WithKeys(toggle), key.WithHelp(toggle, "preview off")),
			joinBindings("move on revset", f.keymap.FileSearch.Up, f.keymap.FileSearch.Down),
			joinBindings("scroll preview", f.keymap.Preview.ScrollUp, f.keymap.Preview.ScrollDown),
		)
	} else {
		shortHelp = append(shortHelp,
			key.NewBinding(key.WithKeys(toggle), key.WithHelp(toggle, "preview on")),
			f.keymap.FileSearch.Accept,
		)
	}

	return shortHelp
}

func (f *FuzzyFilesModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{f.ShortHelp()}
}

type source struct {
	items []string
}

func (s source) String(i int) string {
	return s.items[i]
}

func (s source) Len() int {
	return len(s.items)
}

func NewModel(ctx *context.MainContext) view.IViewModel {
	current := ctx.Revisions.Current().Commit
	out, _ := ctx.RunCommandImmediate(jj.FilesInRevision(current))
	keyMap := config.Current.GetKeyMap()
	i := textinput.New()
	i.Width = 50
	i.Prompt = "> "
	i.Placeholder = "Type to search files"
	i.Cursor.SetMode(cursor.CursorStatic)

	source := source{
		items: strings.Split(string(out), "\n"),
	}

	model := &FuzzyFilesModel{
		context:   ctx,
		keymap:    keyMap,
		input:     i,
		revset:    ctx.CurrentRevset,
		max:       30,
		commit:    ctx.Revisions.Current().Commit,
		styles:    fuzzy_search.NewStyles(),
		fuzzyView: fuzzy_search.NewModel(source, 30),
	}
	return model
}
