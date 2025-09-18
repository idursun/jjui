package details

import (
	"bufio"
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/squash"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ tea.Model = (*Operation)(nil)
var _ operations.Operation = (*Operation)(nil)
var _ view.IViewModel = (*Operation)(nil)
var _ list.IListProvider = (*Operation)(nil)

type Operation struct {
	*DetailsList
	*view.ViewNode
	context           *context.MainContext
	revision          *models.RevisionItem
	confirmation      *confirmation.Model
	keyMap            config.KeyMappings[key.Binding]
	targetMarkerStyle lipgloss.Style
	selected          *models.RevisionItem
}

func (o *Operation) CurrentItem() models.IItem {
	return o.DetailsList.Current()
}

func (o *Operation) CheckedItems() []models.IItem {
	checkedItems := o.DetailsList.GetCheckedItems()
	var items []models.IItem
	for _, item := range checkedItems {
		items = append(items, item)
	}
	return items
}

func (o *Operation) Mount(v *view.ViewNode) {
	o.ViewNode = v
	v.Id = o.GetId()
	v.Sizeable.SetWidth(80)
	v.Sizeable.SetHeight(25)
}

func (o *Operation) GetId() view.ViewId {
	return view.DetailsViewId
}

func (o *Operation) Init() tea.Cmd {
	return o.context.Files.Load()
}

type styles struct {
	Added    lipgloss.Style
	Deleted  lipgloss.Style
	Modified lipgloss.Style
	Renamed  lipgloss.Style
	Selected lipgloss.Style
	Dimmed   lipgloss.Style
	Text     lipgloss.Style
	Conflict lipgloss.Style
}

func (o *Operation) ShortHelp() []key.Binding {
	if o.confirmation != nil {
		return o.confirmation.ShortHelp()
	}
	return []key.Binding{
		o.keyMap.Cancel,
		o.keyMap.Details.Diff,
		o.keyMap.Details.ToggleSelect,
		o.keyMap.Details.Split,
		o.keyMap.Details.Squash,
		o.keyMap.Details.Restore,
		o.keyMap.Details.Absorb,
		o.keyMap.Details.RevisionsChangingFile,
	}
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func (o *Operation) getSelectedFiles() ([]*models.RevisionFileItem, bool) {
	selectedFiles := make([]*models.RevisionFileItem, 0)
	checkedItems := o.GetCheckedItems()

	if len(checkedItems) == 0 {
		current := o.DetailsList.Current()
		return []*models.RevisionFileItem{current}, true
	}

	for _, f := range checkedItems {
		if f.IsChecked() {
			selectedFiles = append(selectedFiles, f)
		}
	}
	return selectedFiles, false
}

func (o *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("Describe operation update %T", msg)
	switch msg := msg.(type) {
	case common.RefreshMsg:
		return o, o.context.Files.Load()
	case tea.KeyMsg:
		if o.confirmation != nil {
			var cmd tea.Cmd
			o.confirmation, cmd = o.confirmation.Update(msg)
			return o, cmd
		}
		switch {
		case key.Matches(msg, o.keyMap.Up):
			o.CursorUp()
			return o, nil
		case key.Matches(msg, o.keyMap.Down):
			o.CursorDown()
			return o, nil
		case key.Matches(msg, o.keyMap.Cancel), key.Matches(msg, o.keyMap.Details.Close):
			o.ViewManager.UnregisterView(o.GetId())
			return o, nil
		case key.Matches(msg, o.keyMap.Details.Diff):
			selected := o.DetailsList.Current()
			if selected == nil {
				return o, nil
			}
			return o, func() tea.Msg {
				return common.LoadDiffLayoutMsg{
					Args: jj.DiffCommandArgs{
						Source: jj.NewDiffRevisionsSource(jj.NewSingleSourceFromRevision(o.revision)),
						Files:  []models.RevisionFileItem{*selected},
					},
				}
			}
		case key.Matches(msg, o.keyMap.Details.Split):
			selectedFiles, isVirtuallySelected := o.getSelectedFiles()
			o.isVirtuallySelected = isVirtuallySelected
			o.selectedHint = "stays as is"
			o.unselectedHint = "moves to the new revision"
			model := confirmation.New(
				[]string{"Are you sure you want to split the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					tea.Batch(o.revisionsContext.RunInteractiveCommand(jj.Args(jj.SplitArgs{Revision: *o.revision, Files: jj.Convert(selectedFiles)}), common.Refresh), o.close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			o.confirmation = model
			return o, o.confirmation.Init()
		case key.Matches(msg, o.keyMap.Details.Squash):
			o.context.Revisions.JumpToParent(jj.NewSelectedRevisions(o.revision))
			selectedItems, _ := o.getSelectedFiles()
			o.close()
			op := squash.NewOperation(o.context, squash.NewSquashFilesOpts(jj.NewSelectedRevisions(o.revision), selectedItems))
			v := o.ViewManager.CreateChildView(view.RevisionsViewId, op)
			o.ViewManager.FocusView(v.GetId())
			return o, op.Init()
		case key.Matches(msg, o.keyMap.Details.Restore):
			selectedFiles, isVirtuallySelected := o.getSelectedFiles()
			o.isVirtuallySelected = isVirtuallySelected
			o.selectedHint = "gets restored"
			o.unselectedHint = "stays as is"
			model := confirmation.New(
				[]string{"Are you sure you want to restore the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					o.revisionsContext.RunCommand(jj.Args(jj.RestoreArgs{Revision: *o.revision, Files: jj.Convert(selectedFiles)}), common.Refresh, confirmation.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			o.confirmation = model
			return o, o.confirmation.Init()
		case key.Matches(msg, o.keyMap.Details.Absorb):
			selectedFiles, isVirtuallySelected := o.getSelectedFiles()
			o.isVirtuallySelected = isVirtuallySelected
			o.selectedHint = "might get absorbed into parents"
			o.unselectedHint = "stays as is"

			model := confirmation.New(
				[]string{"Are you sure you want to absorb changes from the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					o.revisionsContext.RunCommand(jj.Args(jj.AbsorbArgs{From: *o.revision, Files: selectedFiles}), common.Refresh, confirmation.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			o.confirmation = model
			return o, o.confirmation.Init()
		case key.Matches(msg, o.keyMap.Details.ToggleSelect):
			current := o.DetailsList.Current()
			current.Toggle()
			o.CursorDown()
			return o, nil
		case key.Matches(msg, o.keyMap.Details.RevisionsChangingFile):
			if current := o.DetailsList.Current(); current != nil {
				o.revisionsContext.CurrentRevset = fmt.Sprintf("files(%s)", jj.EscapeFileName(current.FileName))
				return o, common.Refresh
			}
			return o, nil
		}
	case confirmation.CloseMsg:
		o.confirmation = nil
		o.selectedHint = ""
		o.unselectedHint = ""
		return o, nil
	}
	return o, nil
}

func (o *Operation) close() tea.Msg {
	o.ViewManager.UnregisterView(o.GetId())
	return nil
}

func (o *Operation) Render(commit *models.Commit, pos operations.RenderPosition) string {
	isSelected := o.selected.Commit.GetChangeId() == commit.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return ""
	}
	return o.View()
}

func (o *Operation) View() string {
	confirmationView := ""
	ch := 0
	if o.confirmation != nil {
		confirmationView = o.confirmation.View()
		ch = lipgloss.Height(confirmationView)
	}
	if len(o.Items) == 0 {
		return o.styles.Dimmed.Render("No changes\n")
	}
	o.renderer.SetHeight(min(o.Height-5-ch, len(o.Items)))
	filesView := o.renderer.Render()

	rendered := lipgloss.JoinVertical(lipgloss.Top, filesView, confirmationView)
	// We are trimming spaces from each line to prevent visual artefacts
	// Empty lines use the default background colour, and it looks bad if the user has a custom background colour
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(rendered))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		lines = append(lines, line)
	}
	rendered = strings.Join(lines, "\n")
	w, h := lipgloss.Size(rendered)
	return lipgloss.Place(w, h, 0, 0, rendered, lipgloss.WithWhitespaceBackground(o.styles.Text.GetBackground()))
}

func NewOperation(ctx *context.MainContext, selected *models.RevisionItem) *Operation {
	s := styles{
		Added:    common.DefaultPalette.Get("revisions details added"),
		Deleted:  common.DefaultPalette.Get("revisions details deleted"),
		Modified: common.DefaultPalette.Get("revisions details modified"),
		Renamed:  common.DefaultPalette.Get("revisions details renamed"),
		Selected: common.DefaultPalette.Get("revisions details selected"),
		Dimmed:   common.DefaultPalette.Get("revisions details dimmed"),
		Text:     common.DefaultPalette.Get("revisions details text"),
		Conflict: common.DefaultPalette.Get("revisions details conflict"),
	}
	ctx.Files.CheckableList.SetItems(nil)

	dl := &DetailsList{
		CheckableList: ctx.Files.CheckableList,
		styles:        s,
	}
	dl.renderer = list.NewRenderer[*models.RevisionFileItem](dl.List, dl, view.NewSizeable(30, 20))
	m := &Operation{
		revision:          ctx.Revisions.Current(),
		DetailsList:       dl,
		context:           ctx,
		selected:          selected,
		keyMap:            config.Current.GetKeyMap(),
		targetMarkerStyle: common.DefaultPalette.Get("details target_marker"),
	}
	return m
}
