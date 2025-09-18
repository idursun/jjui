package ui

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	customcommands "github.com/idursun/jjui/internal/ui/custom_commands"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/flash"
	"github.com/idursun/jjui/internal/ui/helppage"
	"github.com/idursun/jjui/internal/ui/oplog"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/internal/ui/status"
	"github.com/idursun/jjui/internal/ui/view"
)

var tabKey = key.NewBinding(
	key.WithKeys("tab"),
	key.WithHelp("tab", "next"),
)

type triggerAutoRefreshMsg struct{}

type Model struct {
	*view.Sizeable
	customCommands *customcommands.Model
	flash          *flash.Model
	context        *context.MainContext
	keymap         config.KeyMappings[key.Binding]
	layouts        []view.ILayoutConstraint
	viewManager    *view.ViewManager
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, v := range m.viewManager.GetViews() {
		if !v.Visible {
			continue
		}
		cmds = append(cmds, v.Model.Init())
	}
	cmds = append(cmds, tea.SetWindowTitle(fmt.Sprintf("jjui - %s", m.context.Location)))
	cmds = append(cmds, m.flash.Init())
	cmds = append(cmds, m.customCommands.Init())
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("UI Update: %T\n", msg)
	var cmds []tea.Cmd

	var cmd tea.Cmd
	m.flash, cmd = m.flash.Update(msg)
	cmds = append(cmds, cmd)

	update, cmd := m.internalUpdate(msg)
	cmds = append(cmds, cmd)
	for _, v := range m.viewManager.GetViewsNeedRefresh() {
		v.Model, cmd = v.Model.Update(common.RefreshMsg{})
		cmds = append(cmds, cmd)
	}
	return update, tea.Batch(cmds...)
}

func (m Model) internalUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewManager.SetHeight(msg.Height)
		m.viewManager.SetWidth(msg.Width)
		m.viewManager.Layout()
		m.SetWidth(msg.Width)
		m.SetHeight(msg.Height)
	case common.ShowDiffMsg:
		if diffView := m.viewManager.GetView(view.DiffViewId); diffView != nil {
			diffView.Visible = true
			m.viewManager.FocusView(view.DiffViewId)

			lb := view.NewLayoutBuilder()
			layout := lb.VerticalContainer(
				lb.Grow(view.DiffViewId, 1),
				lb.Fit(view.StatusViewId),
			)
			m.viewManager.SetLayout(layout)
			m.layouts = append(m.layouts, layout)
			return m, nil
		}
	case common.LoadDiffLayoutMsg:
		if diffView := m.viewManager.GetView(view.DiffViewId); diffView != nil {
			diffView.Visible = true
			m.viewManager.FocusView(view.DiffViewId)

			lb := view.NewLayoutBuilder()
			layout :=
				lb.VerticalContainer(
					lb.Grow(view.DiffViewId, 1),
					lb.Fit(view.StatusViewId),
				)
			m.layouts = append(m.layouts, layout)
			m.viewManager.SetLayout(layout)
			return m, diff.UpdateDiffCommand(msg.Args)
		}
	case common.LoadOplogLayoutMsg:
		if oplogView := m.viewManager.GetView(view.OpLogViewId); oplogView != nil {
			oplogView.Visible = true
			m.viewManager.FocusView(view.OpLogViewId)
			lb := view.NewLayoutBuilder()
			layout := lb.VerticalContainer(
				lb.HorizontalContainer(
					lb.Grow(oplogView.Id, 1),
					lb.Percentage(view.PreviewViewId, int(m.context.Preview.WindowPercentage)),
				),
				lb.Fit(view.StatusViewId),
			)
			m.viewManager.SetLayout(layout)
			m.layouts = append(m.layouts, layout)
			m.viewManager.SetLayout(layout)
			return m, oplogView.Model.Init()
		}
	case tea.KeyMsg:
		if m.viewManager.IsEditing() {
			var cmd tea.Cmd
			editingView := m.viewManager.GetEditingView()
			editingView.Model, cmd = editingView.Model.Update(msg)
			return m, cmd
		}
		switch {
		case key.Matches(msg, tabKey):
			focusedView := m.viewManager.GetFocusedView()
			if focusedView == nil {
				return m, nil
			}
			if focusedView.Id == view.PreviewViewId {
				m.viewManager.RestorePreviousFocus()
				return m, nil
			}
			if v := m.viewManager.GetView(view.PreviewViewId); v != nil && v.Visible {
				m.viewManager.FocusView(view.PreviewViewId)
				m.viewManager.StartEditing(view.PreviewViewId)
			}
			return m, nil
		case key.Matches(msg, m.keymap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.Cancel):
			switch {
			case m.viewManager.IsEditing():
				m.viewManager.StopEditing()
				m.viewManager.RestorePreviousFocus()
				return m, nil
			case m.flash.Any():
				m.flash.DeleteOldest()
				return m, nil

			case len(m.layouts) > 1:
				m.layouts = m.layouts[:len(m.layouts)-1]
				m.viewManager.SetLayout(m.layouts[len(m.layouts)-1])
				m.viewManager.RestorePreviousFocus()
				return m, nil
			}
		case key.Matches(msg, m.keymap.Revset):
			m.viewManager.FocusView(view.RevsetViewId)
			m.viewManager.StartEditing(view.RevsetViewId)
			return m, nil
		case key.Matches(msg, m.keymap.Help):
			model := helppage.New(m.context)
			v := m.viewManager.CreateView(model)
			m.viewManager.AddModal(v, view.CenterX(), view.CenterY())
			return m, model.Init()
		case key.Matches(msg, m.keymap.Preview.Mode):
			if previewView := m.viewManager.GetView(view.PreviewViewId); previewView != nil {
				var cmds []tea.Cmd
				previewView.Visible = !previewView.Visible
				if previewView.Visible {
					cmds = append(cmds, previewView.Model.Init())
				}
				m.viewManager.Layout()
				return m, tea.Batch(cmds...)
			}
		case key.Matches(msg, m.keymap.Suspend):
			return m, tea.Suspend
		case key.Matches(msg, m.keymap.CustomCommands):
			model := customcommands.NewModel(m.context)
			v := m.viewManager.CreateView(model)
			m.viewManager.AddModal(v, view.CenterX(), view.CenterY())
			m.viewManager.StartEditing(v.GetId())
			return m, model.Init()
		default:
			var cmds []tea.Cmd
			var cmd tea.Cmd
			if focusedView := m.viewManager.GetFocusedView(); focusedView != nil {
				if focusedView.KeyDelegation != nil {
					if delegatedView := m.viewManager.GetView(*focusedView.KeyDelegation); delegatedView != nil {
						delegatedView.Model, cmd = delegatedView.Model.Update(msg)
						cmds = append(cmds, cmd)
						return m, tea.Batch(cmds...)
					}
				}
				focusedView.Model, cmd = focusedView.Model.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}
		}
	}

	var cmds []tea.Cmd
	views := m.viewManager.GetViews()
	for _, v := range views {
		if v.Visible {
			var cmd tea.Cmd
			v.Model, cmd = v.Model.Update(msg)
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	rendered := m.viewManager.Render()
	if flashView := m.flash.View(); flashView != "" {
		fw, fh := lipgloss.Size(flashView)
		rendered = screen.Stacked(rendered, flashView, m.Width-fw-1, m.Height-fh-1)
	}
	return rendered
}

func New(c *context.MainContext) tea.Model {
	vm := view.NewViewManager()
	revisionsModel := revisions.New(c, vm)
	revisionsView := vm.CreateView(revisionsModel)
	previewView := vm.CreateView(preview.New(c, c.Preview))
	previewView.Visible = config.Current.Preview.ShowAtStart
	revsetView := vm.CreateView(revset.New(c))
	statusView := vm.CreateView(status.New(c))
	diffView := vm.CreateView(diff.New(c))
	diffView.Visible = false
	oplogView := vm.CreateView(oplog.New(c))
	oplogView.Visible = false

	vm.FocusView(revisionsView.Id)
	lb := view.NewLayoutBuilder()
	layout :=
		lb.VerticalContainer(
			lb.Fit(revsetView.Id),
			lb.HorizontalContainer(
				lb.Grow(revisionsView.Id, 1),
				lb.Percentage(previewView.Id, int(c.Preview.WindowPercentage)),
			),
			lb.Fit(statusView.Id),
		)
	layouts := []view.ILayoutConstraint{layout}
	vm.SetLayout(layout)

	m := Model{
		Sizeable:       view.NewSizeable(0, 0),
		viewManager:    vm,
		context:        c,
		keymap:         config.Current.GetKeyMap(),
		flash:          flash.New(c),
		layouts:        layouts,
		customCommands: customcommands.NewModel(c),
	}
	return m
}
