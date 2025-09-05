package layouts

import (
	"log"
	"time"

	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/leader"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/internal/ui/status"
)

var _ view.IViewManagerAccessor = (*Model)(nil)

type Model struct {
	state       common.State
	context     *context.MainContext
	viewManager *view.ViewManager
	keyMap      config.KeyMappings[key.Binding]
}

func (m *Model) GetViewManager() *view.ViewManager {
	return m.viewManager
}

type triggerAutoRefreshMsg struct{}

func (m *Model) Init() tea.Cmd {
	views := m.viewManager.GetViews()
	var cmds []tea.Cmd
	cmds = append(cmds, m.scheduleAutoRefresh())
	for _, v := range views {
		cmds = append(cmds, v.Model.Init())
	}
	return tea.Batch(cmds...)
}

var tabKey = key.NewBinding(
	key.WithKeys("tab"),
	key.WithHelp("tab", "next"),
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	nm, cmd := m.internalUpdate(msg)
	cmds = append(cmds, cmd)
	for _, v := range m.viewManager.GetViewsNeedsRefresh() {
		v.Model, cmd = v.Model.Update(common.RefreshMsg{})
		cmds = append(cmds, cmd)
	}
	return nm, tea.Batch(cmds...)
}

func (m *Model) internalUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("Revisions Layout Update: %T\n", msg)
	var cmd tea.Cmd
	if _, ok := msg.(common.CloseViewMsg); ok {
		if m.viewManager.IsEditing() {
			log.Println("revision_layout.Update: stopping edit")
			m.viewManager.StopEditing()
			return m, nil
		}
	}

	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewManager.SetHeight(msg.Height)
		m.viewManager.SetWidth(msg.Width)
		m.viewManager.Layout()
	case tea.KeyMsg:
		if m.viewManager.IsEditing() {
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
		case key.Matches(msg, m.keyMap.Cancel):
			switch {
			case m.state == common.Error:
				m.state = common.Ready
				return m, tea.Batch(cmds...)
			case m.viewManager.IsEditing():
				m.viewManager.StopEditing()
				m.viewManager.RestorePreviousFocus()
				return m, tea.Batch(cmds...)
			}
		case key.Matches(msg, m.keyMap.Revset):
			if revsetView := m.viewManager.GetView(view.RevsetViewId); revsetView != nil {
				m.viewManager.FocusView(revsetView.Id)
				m.viewManager.StartEditing(revsetView.Id)
				return m, nil
			}
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Mode):
			if previewView := m.viewManager.GetView(view.PreviewViewId); previewView != nil {
				previewView.Visible = !previewView.Visible
				if previewView.Visible {
					cmds = append(cmds, previewView.Model.Init())
				}
			}
			m.viewManager.Layout()
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Leader):
			//m.leader = leader.New(m.context)
			// We'll keep the traditional handling for the leader mode
			// since it doesn't fully implement tea.Model
			cmds = append(cmds, leader.InitCmd)
			return m, tea.Batch(cmds...)
		default:
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
	views := m.viewManager.GetViews()
	for _, v := range views {
		if v.Visible {
			v.Model, cmd = v.Model.Update(msg)
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	return m.viewManager.Render()
}

func (m *Model) scheduleAutoRefresh() tea.Cmd {
	interval := config.Current.UI.AutoRefreshInterval
	if interval > 0 {
		return tea.Tick(time.Duration(interval)*time.Second, func(time.Time) tea.Msg {
			return triggerAutoRefreshMsg{}
		})
	}
	return nil
}

func NewRevisionsLayout(c *context.MainContext) *Model {
	vm := view.NewViewManager()

	// Create the revisionsView
	revisionsModel := revisions.New(c, vm)
	revisionsView := vm.CreateView(revisionsModel)
	previewView := vm.CreateView(preview.New(c, c.Preview))
	previewView.Visible = config.Current.Preview.ShowAtStart
	revsetView := vm.CreateView(revset.New(c))
	statusView := vm.CreateView(status.New(c))

	vm.RegisterView(revisionsView)
	vm.RegisterView(previewView)
	vm.RegisterView(statusView)
	vm.RegisterView(revsetView)
	vm.FocusView(revisionsView.Id)
	lb := view.NewLayoutBuilder()
	vm.SetLayout(
		lb.VerticalContainer(
			lb.Fit(revsetView.Id),
			lb.HorizontalContainer(
				lb.Grow(revisionsView.Id, 1),
				lb.Percentage(previewView.Id, int(c.Preview.WindowPercentage)),
			),
			lb.Fit(statusView.Id),
		),
	)

	m := &Model{
		context:     c,
		viewManager: vm,
		keyMap:      config.Current.GetKeyMap(),
		state:       common.Loading,
	}
	return m
}
