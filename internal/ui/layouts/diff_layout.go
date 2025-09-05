package layouts

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/status"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ tea.Model = (*DiffLayout)(nil)

type DiffLayout struct {
	viewManager *view.ViewManager
}

func (o *DiffLayout) Init() tea.Cmd {
	views := o.viewManager.GetViews()
	var cmds []tea.Cmd
	for _, v := range views {
		cmds = append(cmds, v.Model.Init())
	}
	return tea.Batch(cmds...)
}

func (o *DiffLayout) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if _, ok := msg.(common.CloseViewMsg); ok {
		if o.viewManager.IsEditing() {
			o.viewManager.StopEditing()
			return o, nil
		}
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		o.viewManager.SetWidth(msg.Width)
		o.viewManager.SetHeight(msg.Height)
		o.viewManager.Layout()
	case tea.KeyMsg:
		if focusedView := o.viewManager.GetFocusedView(); focusedView != nil {
			focusedView.Model, cmd = focusedView.Model.Update(msg)
			return o, cmd
		}
	}
	views := o.viewManager.GetViews()
	var cmds []tea.Cmd
	for _, v := range views {
		if v.Visible {
			v.Model, cmd = v.Model.Update(msg)
			cmds = append(cmds, cmd)
		}
	}
	return o, tea.Batch(cmds...)
}

func (o *DiffLayout) View() string {
	return o.viewManager.Render()
}

func NewDiffLayout(ctx *context.MainContext, diffViewModel view.IViewModel) tea.Model {
	vm := view.NewViewManager()
	diffView := vm.CreateView(diffViewModel)
	statusView := vm.CreateView(status.New(ctx))
	vm.RegisterView(diffView)
	vm.RegisterView(statusView)
	vm.FocusView(diffView.Id)
	lb := view.NewLayoutBuilder()
	vm.SetLayout(
		lb.VerticalContainer(
			lb.Grow(diffView.Id, 1),
			lb.Fit(statusView.Id),
		),
	)

	return &DiffLayout{
		viewManager: vm,
	}
}
