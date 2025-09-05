package layouts

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/oplog"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/status"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ tea.Model = (*OplogLayout)(nil)

type OplogLayout struct {
	viewManager *view.ViewManager
	context     *context.MainContext
	keyMap      config.KeyMappings[key.Binding]
}

func (o *OplogLayout) Init() tea.Cmd {
	views := o.viewManager.GetViews()
	var cmds []tea.Cmd
	for _, v := range views {
		cmds = append(cmds, v.Model.Init())
	}
	return tea.Batch(cmds...)
}
func (o *OplogLayout) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	nm, cmd := o.internalUpdate(msg)
	cmds = append(cmds, cmd)
	for _, v := range o.viewManager.GetViewsNeedsRefresh() {
		v.Model, cmd = v.Model.Update(common.RefreshMsg{})
		cmds = append(cmds, cmd)
	}
	return nm, tea.Batch(cmds...)
}

func (o *OplogLayout) internalUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		switch {
		default:
			if focusedView := o.viewManager.GetFocusedView(); focusedView != nil {
				focusedView.Model, cmd = focusedView.Model.Update(msg)
				return o, cmd
			}
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

func (o *OplogLayout) View() string {
	return o.viewManager.Render()
}

func NewOplogLayout(ctx *context.MainContext) tea.Model {
	vm := view.NewViewManager()
	oplogView := vm.CreateView(oplog.New(ctx))
	statusView := vm.CreateView(status.New(ctx))
	previewView := vm.CreateView(preview.New(ctx, ctx.Preview))
	previewView.Visible = config.Current.Preview.ShowAtStart
	vm.RegisterView(oplogView)
	vm.RegisterView(statusView)
	vm.RegisterView(previewView)
	vm.FocusView(oplogView.Id)
	lb := view.NewLayoutBuilder()
	vm.SetLayout(
		lb.VerticalContainer(
			lb.HorizontalContainer(
				lb.Grow(oplogView.Id, 1),
				lb.Percentage(previewView.Id, int(ctx.Preview.WindowPercentage)),
			),
			lb.Fit(statusView.Id),
		),
	)

	return &OplogLayout{
		viewManager: vm,
		context:     ctx,
		keyMap:      config.Current.GetKeyMap(),
	}
}
