package revisions

import (
	"log"
	"strings"
	"sync/atomic"

	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/bookmarks"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/exec_prompt"
	"github.com/idursun/jjui/internal/ui/file_search"
	"github.com/idursun/jjui/internal/ui/git"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/ace_jump"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/operations/duplicate"
	"github.com/idursun/jjui/internal/ui/operations/quick_search"
	"github.com/idursun/jjui/internal/ui/operations/revert"
	"github.com/idursun/jjui/internal/ui/operations/set_parents"
	"github.com/idursun/jjui/internal/ui/operations/squash"
	"github.com/idursun/jjui/internal/ui/undo"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/operations/describe"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations/abandon"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/evolog"
	"github.com/idursun/jjui/internal/ui/operations/rebase"
)

var _ tea.Model = (*Model)(nil)
var _ view.IViewModel = (*Model)(nil)
var _ list.IListProvider = (*Model)(nil)

type Model struct {
	*view.ViewNode
	*RevisionList
	context         *appContext.MainContext
	keymap          config.KeyMappings[key.Binding]
	output          string
	err             error
	previousOpLogId string
	isLoading       bool
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	v.Height = v.ViewManager.Height
	v.Width = v.ViewManager.Width
	m.renderer.Sizeable = v.Sizeable
	v.Id = m.GetId()
}

func (m *Model) GetId() view.ViewId {
	return view.RevisionsViewId
}

type revisionsMsg struct {
	msg tea.Msg
}

func (m *Model) SelectedRevision() *models.Commit {
	if current := m.Current(); current != nil {
		return current.Commit
	}
	return nil
}

func (m *Model) SelectedRevisions() jj.SelectedRevisions {
	checked := m.GetCheckedItems()
	if len(checked) == 0 {
		checked = append(checked, m.Current())
	}

	return checked
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keymap.Up,
		m.keymap.Down,
		m.keymap.Quit,
		m.keymap.Help,
		m.keymap.Refresh,
		m.keymap.Preview.Mode,
		m.keymap.Revset,
		m.keymap.Details.Mode,
		m.keymap.Evolog.Mode,
		m.keymap.Rebase.Mode,
		m.keymap.Squash.Mode,
		m.keymap.Bookmark.Mode,
		m.keymap.Git.Mode,
		m.keymap.OpLog.Mode,
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	return common.RefreshAndSelect("@")
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("Revisions Update: %T\n", msg)
	if k, ok := msg.(revisionsMsg); ok {
		msg = k.msg
	}
	switch msg := msg.(type) {
	case common.CommandCompletedMsg:
		m.output = msg.Output
		m.err = msg.Err
		return m, nil
	case common.RefreshMsg:
		if !msg.KeepSelections {
			m.context.Revisions.ClearCheckedItems()
		}
		m.isLoading = true
		if config.Current.Revisions.LogBatching {
			return m, m.context.Revisions.LoadStreaming(m.context.CurrentRevset, msg.SelectedRevision)
		} else {
			return m, m.context.Revisions.Load(m.context.CurrentRevset, msg.SelectedRevision)
		}
	case appContext.UpdateRevisionsMsg:
		m.isLoading = false
		return m, m.highlightChanges
	}

	if len(m.Items) == 0 {
		return m, nil
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Up):
			return m, m.context.Revisions.CursorUp()
		case key.Matches(msg, m.keymap.Down):
			return m, m.context.Revisions.CursorDown()
		case key.Matches(msg, m.keymap.JumpToParent):
			m.context.Revisions.JumpToParent(m.SelectedRevisions())
		case key.Matches(msg, m.keymap.JumpToChildren):
			immediate, _ := m.context.RunCommandImmediate(jj.GetFirstChild(m.SelectedRevision()))
			index := m.context.Revisions.FindRevision(string(immediate))
			if index != -1 {
				m.Cursor = index
			}
		case key.Matches(msg, m.keymap.JumpToWorkingCopy):
			workingCopyIndex := m.context.Revisions.FindRevision("@")
			if workingCopyIndex != -1 {
				m.Cursor = workingCopyIndex
			}
		case key.Matches(msg, m.keymap.AceJump):
			model := ace_jump.NewOperation(m.context.Revisions.List, m.renderer)
			v := m.ViewManager.CreateChildView(m.GetId(), model)
			m.ViewManager.FocusView(v.GetId())
			m.ViewManager.StartEditing(v.GetId())
			return m, model.Init()
		case key.Matches(msg, m.keymap.QuickSearch):
			model := quick_search.NewOperation(m.context.Revisions.List)
			v := m.ViewManager.CreateChildView(m.GetId(), model)
			m.ViewManager.AddModal(v, view.Left(len(model.GetId())+3), view.Bottom())
			m.ViewManager.StartEditing(v.GetId())
			return m, model.Init()
		default:
			if subView := m.ViewManager.GetChildView(m.GetId()); subView != nil && subView.Visible {
				subView.Model, cmd = subView.Model.Update(msg)
				return m, cmd
			}

			switch {
			case key.Matches(msg, m.keymap.OpLog.Mode):
				return m, func() tea.Msg {
					return common.LoadOplogLayoutMsg{}
				}
			case key.Matches(msg, m.keymap.ToggleSelect):
				current := m.Current()
				current.Toggle()
				m.context.Revisions.JumpToParent(jj.NewSelectedRevisions(current))
			case key.Matches(msg, m.keymap.Details.Mode):
				op := details.NewOperation(m.context, m.context.Revisions.Current())
				v := m.ViewManager.CreateView(op)
				m.ViewManager.FocusView(v.GetId())
				return m, op.Init()
			case key.Matches(msg, m.keymap.InlineDescribe.Mode):
				op := describe.NewOperation(m.context, m.Current())
				v := m.ViewManager.CreateChildView(m.GetId(), op)
				m.ViewManager.FocusView(v.GetId())
				m.ViewManager.StartEditing(v.GetId())
				return m, op.Init()
			case key.Matches(msg, m.keymap.New):
				cmd = m.context.RunCommand(jj.Args(jj.NewArgs{Revisions: m.SelectedRevisions()}), common.RefreshAndSelect("@"))
			case key.Matches(msg, m.keymap.Commit):
				cmd = m.context.RunInteractiveCommand(jj.Args(jj.CommitArgs{}), common.Refresh)
			case key.Matches(msg, m.keymap.Edit, m.keymap.ForceEdit):
				ignoreImmutable := key.Matches(msg, m.keymap.ForceEdit)
				cmd = m.context.RunCommand(jj.Args(jj.EditArgs{
					Revision:        *m.Current(),
					GlobalArguments: jj.GlobalArguments{IgnoreImmutable: ignoreImmutable},
				}), common.Refresh)
			case key.Matches(msg, m.keymap.Diffedit):
				cmd = m.context.RunInteractiveCommand(jj.Args(jj.DiffEditArgs{Revision: *m.Current()}), common.Refresh)
			case key.Matches(msg, m.keymap.Absorb):
				cmd = m.context.RunCommand(jj.Args(jj.AbsorbArgs{From: *m.Current()}), common.Refresh)
			case key.Matches(msg, m.keymap.Abandon):
				selections := m.SelectedRevisions()
				op := abandon.NewOperation(m.context, selections)
				v := m.ViewManager.CreateChildView(m.GetId(), op)
				m.ViewManager.FocusView(v.GetId())
				return m, op.Init()
			case key.Matches(msg, m.keymap.Bookmark.Set):
				op := bookmark.NewSetBookmarkOperation(m.context, m.Current())
				v := m.ViewManager.CreateChildView(m.GetId(), op)
				m.ViewManager.StartEditing(v.Id)
				m.ViewManager.FocusView(v.Id)
				return m, op.Init()
			case key.Matches(msg, m.keymap.Split):
				return m, m.context.RunInteractiveCommand(jj.Args(jj.SplitArgs{Revision: *m.Current()}), common.Refresh)
			case key.Matches(msg, m.keymap.Describe):
				return m, m.context.RunInteractiveCommand(jj.Args(jj.DescribeArgs{Revisions: m.SelectedRevisions()}), common.Refresh)
			case key.Matches(msg, m.keymap.Evolog.Mode):
				op := evolog.NewOperation(m.context, m.Current())
				v := m.ViewManager.CreateChildView(m.GetId(), op)
				m.ViewManager.FocusView(v.GetId())
				return m, op.Init()
			case key.Matches(msg, m.keymap.Refresh):
				cmd = common.Refresh
			case key.Matches(msg, m.keymap.Diff):
				current := m.context.Revisions.Current()
				return m, func() tea.Msg {
					return common.LoadDiffLayoutMsg{
						Args: jj.DiffCommandArgs{Source: jj.NewDiffRevisionsSource(jj.NewSingleSourceFromRevision(current))},
					}
				}
			case key.Matches(msg, m.keymap.Git.Mode):
				current := m.context.Revisions.Current()
				model := git.NewModel(m.context, current)
				m.createModalView(model)
				return m, model.Init()
			case key.Matches(msg, m.keymap.Bookmark.Mode):
				changeIds := m.context.Revisions.GetCommitIds()
				current := m.context.Revisions.Current()
				model := bookmarks.NewModel(m.context, current, changeIds)
				m.createModalView(model)
				return m, model.Init()
			case key.Matches(msg, m.keymap.Undo):
				model := undo.NewModel(m.context)
				m.createModalView(model)
				return m, model.Init()
			case key.Matches(msg, m.keymap.Squash.Mode):
				selectedRevisions := m.SelectedRevisions()
				parent, _ := m.context.RunCommandImmediate(jj.GetParent(selectedRevisions).GetArgs())
				parentIdx := m.context.Revisions.FindRevision(string(parent))
				if parentIdx != -1 {
					m.Cursor = parentIdx
				} else if m.Cursor < len(m.Items)-1 {
					m.Cursor++
				}
				op := squash.NewOperation(m.context, squash.NewSquashRevisionsOpts(selectedRevisions))
				v := m.ViewManager.CreateChildView(m.GetId(), op)
				m.ViewManager.FocusView(v.GetId())
				return m, op.Init()
			case key.Matches(msg, m.keymap.Revert.Mode):
				op := revert.NewOperation(m.context, m.SelectedRevisions())
				v := m.ViewManager.CreateChildView(m.GetId(), op)
				m.ViewManager.FocusView(v.GetId())
				return m, op.Init()
			case key.Matches(msg, m.keymap.Rebase.Mode):
				op := rebase.NewOperation(m.context, m.SelectedRevisions())
				v := m.ViewManager.CreateChildView(m.GetId(), op)
				m.ViewManager.FocusView(v.GetId())
				return m, op.Init()
			case key.Matches(msg, m.keymap.Duplicate.Mode):
				op := duplicate.NewOperation(m.context, m.SelectedRevisions())
				v := m.ViewManager.CreateChildView(m.GetId(), op)
				m.ViewManager.FocusView(v.GetId())
				return m, op.Init()
			case key.Matches(msg, m.keymap.SetParents):
				op := set_parents.NewOperation(m.context, m.Current())
				v := m.ViewManager.CreateChildView(m.GetId(), op)
				m.ViewManager.FocusView(v.GetId())
				return m, op.Init()
			case key.Matches(msg, m.keymap.FileSearch.Toggle):
				model := file_search.NewModel(m.context)
				v := m.ViewManager.CreateChildView(m.GetId(), model)
				m.ViewManager.AddModal(v, view.Left(len(model.GetId())+3), view.Bottom())
				m.ViewManager.StartEditing(v.GetId())
				return m, model.Init()
			case key.Matches(msg, m.keymap.ExecShell, m.keymap.ExecJJ):
				mode := common.ExecJJ
				if key.Matches(msg, m.keymap.ExecShell) {
					mode = common.ExecShell
				}
				model := exec_prompt.NewExecPrompt(m.context, mode)
				v := m.ViewManager.CreateChildView(m.GetId(), model)
				m.ViewManager.AddModal(v, view.Left(3), view.Bottom(1))
				m.ViewManager.StartEditing(v.GetId())
				return m, model.Init()
			}
		}
	}

	return m, cmd
}

func (m *Model) createModalView(model view.IViewModel) {
	v := m.ViewManager.CreateView(model)
	m.ViewManager.AddModal(v, view.CenterX(), view.CenterY())
	m.ViewManager.StartEditing(v.GetId())
}

func (m *Model) highlightChanges() tea.Msg {
	if m.err != nil || m.output == "" {
		return nil
	}

	changes := strings.Split(m.output, "\n")
	for _, change := range changes {
		if !strings.HasPrefix(change, " ") {
			continue
		}
		line := strings.Trim(change, "\n ")
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		if len(parts) > 0 {
			for i := range m.Items {
				row := m.Items[i]
				if row.Commit.GetChangeId() == parts[0] {
					row.IsAffected = true
					break
				}
			}
		}
	}
	return nil
}

func (m *Model) View() string {
	if len(m.Items) == 0 {
		if m.isLoading {
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "loading")
		}
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "(no matching revisions)")
	}

	selections := make(map[string]bool)
	checked := m.GetCheckedItems()
	for _, item := range checked {
		selections[item.Commit.GetChangeId()] = true
	}

	output := m.renderer.Render()
	output = m.textStyle.MaxWidth(m.Width).Render(output)
	return lipgloss.Place(m.Width, m.Height, 0, 0, output)
}

var _ operations.Operation = (*noop)(nil)

type noop struct{}

func (n noop) Render(*models.Commit, operations.RenderPosition) string {
	return ""
}

func New(c *appContext.MainContext, viewManager *view.ViewManager) view.IViewModel {
	keymap := config.Current.GetKeyMap()
	l := c.Revisions

	rl := &RevisionList{
		CheckableList: l.CheckableList,
		textStyle:     common.DefaultPalette.Get("revisions text"),
		selectedStyle: common.DefaultPalette.Get("revisions selected").Inline(true),
		dimmedStyle:   common.DefaultPalette.Get("revisions dimmed"),
		checkStyle:    common.DefaultPalette.Get("revisions success").Inline(true),
		Tracer:        parser.NewNoopTracer(),
		getOpFn: func() operations.Operation {
			for _, v := range viewManager.GetViews() {
				if !v.Visible {
					continue
				}
				if op, ok := v.Model.(operations.Operation); ok {
					return op
				}
			}
			return noop{}
		},
	}
	rl.renderer = list.NewRenderer[*models.RevisionItem](l.List, rl, view.NewSizeable(0, 0))

	m := Model{
		RevisionList: rl,
		context:      c,
		keymap:       keymap,
	}
	return &m
}
