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
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/flash"
	"github.com/idursun/jjui/internal/ui/helppage"
	"github.com/idursun/jjui/internal/ui/layouts"
	"github.com/idursun/jjui/internal/ui/view"
)

type Model struct {
	*view.Sizeable
	flash       *flash.Model
	context     *context.MainContext
	keyMap      config.KeyMappings[key.Binding]
	layouts     []tea.Model
	layoutIndex int
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, tea.SetWindowTitle(fmt.Sprintf("jjui - %s", m.context.Location)))
	layout := m.layouts[m.layoutIndex]
	cmds = append(cmds, layout.Init())
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
	return update, tea.Batch(cmds...)
}

func (m Model) internalUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	layout := m.layouts[m.layoutIndex]
	if accessor, ok := layout.(view.IViewManagerAccessor); ok && accessor != nil {
		vm := accessor.GetViewManager()
		if vm.IsEditing() {
			return m.updateCurrentLayout(msg)
		}
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetWidth(msg.Width)
		m.SetHeight(msg.Height)
		// update all layout's sizes
		for _, layout := range m.layouts {
			layout, _ = layout.Update(msg)
		}
	case common.ShowDiffMsg:
		v := layouts.NewDiffLayout(m.context, diff.New(m.context, diff.WithOutput(string(msg))))
		m.layouts = append(m.layouts, v)
		m.layoutIndex++
		m.updateCurrentLayout(tea.WindowSizeMsg{Width: m.Width, Height: m.Height})
		return m, v.Init()

	case common.LoadDiffLayoutMsg:
		v := layouts.NewDiffLayout(m.context, diff.New(m.context, diff.WithCommand(msg.Args)))
		m.layouts = append(m.layouts, v)
		m.layoutIndex++
		m.updateCurrentLayout(tea.WindowSizeMsg{Width: m.Width, Height: m.Height})
		return m, v.Init()

	case common.LoadOplogLayoutMsg:
		v := layouts.NewOplogLayout(m.context)
		m.layouts = append(m.layouts, v)
		m.layoutIndex++
		m.updateCurrentLayout(tea.WindowSizeMsg{Width: m.Width, Height: m.Height})
		return m, v.Init()

	case tea.KeyMsg:
		// TODO: if layout is editing then pass the message directly to layout
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.Cancel):
			switch {
			case m.flash.Any():
				m.flash.DeleteOldest()
				return m, nil
			case len(m.layouts) > 1:
				m.layouts = m.layouts[:len(m.layouts)-1]
				m.layoutIndex = len(m.layouts) - 1
				return m, nil
			}
		case key.Matches(msg, m.keyMap.Help):
			layout := m.layouts[m.layoutIndex]
			if accessor, ok := layout.(view.IViewManagerAccessor); ok && accessor != nil {
				vm := accessor.GetViewManager()
				model := helppage.New(m.context)
				v := vm.CreateView(model)
				// Center the help modal on screen
				vm.AddModal(v, view.CenterX(), view.CenterY())
				return m, model.Init()
			}
			return m, nil
		case key.Matches(msg, m.keyMap.Suspend):
			return m, tea.Suspend
		}
	}
	return m.updateCurrentLayout(msg)
}

func (m Model) updateCurrentLayout(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.layouts[m.layoutIndex], cmd = m.layouts[m.layoutIndex].Update(msg)
	return m, cmd
}

func (m Model) View() string {
	layout := m.layouts[m.layoutIndex]

	//var w strings.Builder
	//if l, ok := layout.(view.IViewManagerAccessor); ok {
	//	manager := l.GetViewManager()
	//	for _, view := range manager.GetViews() {
	//		w.WriteString(string(view.Id))
	//		if view.Visible {
	//			w.WriteString(" (visible)")
	//		}
	//		if manager.GetFocusedView() != nil && manager.GetFocusedView().Id == view.Id {
	//			w.WriteString(" (focused)")
	//		}
	//		if manager.IsThisEditing(view.Id) {
	//			w.WriteString(" (editing)")
	//		}
	//		w.WriteString("\n")
	//	}
	//}

	rendered := layout.View()
	if flashView := m.flash.View(); flashView != "" {
		fw, fh := lipgloss.Size(flashView)
		rendered = screen.Stacked(rendered, flashView, m.Width-fw-1, m.Height-fh-1)
	}
	return rendered
	//return screen.Stacked(rendered, w.String(), m.Width-100, m.Height-10)
}

func New(c *context.MainContext) tea.Model {
	layout := layouts.NewRevisionsLayout(c)

	m := Model{
		Sizeable: view.NewSizeable(0, 0),
		layouts:  []tea.Model{layout},
		context:  c,
		keyMap:   config.Current.GetKeyMap(),
		flash:    flash.New(c),
	}
	return m
}
