package workspacepane

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
)

// ActivateMsg requests an execution-context change, not revision navigation.
type ActivateMsg struct{ Name, Path string }
type ReturnToRevisionsMsg struct{}
type PaneClickedMsg struct{}
type ItemClickedMsg struct{ Name string }

// Message is routed to the retained panel even if another pane is now visible.
type Message interface{ workspacePaneMessage() }

type loadedMsg struct {
	location   string
	generation uint64
	workspaces []jj.Workspace
	err        error
}

type completedMsg struct {
	operation string
	name      string
	err       error
}

func (loadedMsg) workspacePaneMessage()           {}
func (completedMsg) workspacePaneMessage()        {}
func (removalConfirmedMsg) workspacePaneMessage() {}

type Model struct {
	context      *context.MainContext
	launchPath   string
	launchName   string
	workspaces   []jj.Workspace
	focused      bool
	busy         bool
	pending      string
	pendingName  string
	addPath      string
	confirmation *confirmation.Model
	list         *render.ListRenderer
	paths        map[string]string
	generation   uint64
}

func New(ctx *context.MainContext) *Model {
	m := &Model{context: ctx, launchPath: ctx.Location, launchName: ctx.WorkspaceName, paths: make(map[string]string)}
	if ctx.WorkspaceName != "" {
		m.paths[ctx.WorkspaceName] = ctx.Location
	}
	m.list = render.NewListRenderer(func(msg tea.MouseMsg) tea.Msg {
		delta, horizontal := render.WheelDelta(msg)
		if horizontal {
			return nil
		}
		return intents.WorkspaceNavigate{Delta: delta}
	})
	m.list.Z = render.ZPreview
	return m
}

func (m *Model) Init() tea.Cmd           { return nil }
func (m *Model) OnShow() tea.Cmd         { return m.load() }
func (m *Model) OnHide()                 { m.focused = false; m.confirmation = nil }
func (m *Model) SetFocused(focused bool) { m.focused = focused }
func (m *Model) IsEditing() bool         { return m.confirmation != nil || m.pending != "" }

func (m *Model) Scopes() []common.Scope {
	if !m.focused {
		return nil
	}
	name := actions.ScopeWorkspacePane
	leak := common.LeakGlobal
	if m.confirmation != nil {
		name = actions.ScopeWorkspacePaneConfirmation
		leak = common.LeakNone
	}
	return []common.Scope{{Name: keybindings.ScopeName(name), Leak: leak, Handler: m}}
}

func message(err error) tea.Cmd {
	return intents.Invoke(intents.AddMessage{Text: err.Error(), Err: err})
}

func (m *Model) load() tea.Cmd {
	runner, location := m.context.CommandRunner, m.context.Location
	m.generation++
	generation := m.generation
	return func() tea.Msg {
		output, err := runner.RunCommandImmediate(jj.WorkspaceList())
		return loadedMsg{location: location, generation: generation, workspaces: jj.ParseWorkspaces(string(output)), err: err}
	}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case removalConfirmedMsg:
		m.confirmation = nil
		return m.run(msg.operation, msg.name, msg.args)
	case loadedMsg:
		if msg.location != m.context.Location || msg.generation != m.generation {
			return nil
		}
		if msg.err != nil {
			return message(msg.err)
		}
		m.workspaces = msg.workspaces
		m.context.IdentifyWorkspace(msg.workspaces)
		if m.context.WorkspaceName != "" {
			m.paths[m.context.WorkspaceName] = m.context.Location
		}
		if m.context.Location == m.launchPath {
			m.launchName = m.context.WorkspaceName
		}
	case completedMsg:
		m.busy = false
		m.generation++
		if msg.err != nil {
			return message(msg.err)
		}
		switch msg.operation {
		case "rename":
			delete(m.paths, m.context.WorkspaceName)
			m.context.WorkspaceName = msg.name
			m.paths[msg.name] = m.context.Location
			if m.context.Location == m.launchPath {
				m.launchName = msg.name
			}
		case "forget", "remove":
			delete(m.paths, msg.name)
			return func() tea.Msg { return ActivateMsg{Name: m.launchName, Path: m.launchPath} }
		}
		return common.RefreshAndKeepSelections
	case common.RefreshMsg:
		return m.load()
	case ItemClickedMsg:
		if !m.busy && m.confirmation == nil {
			return m.activate(msg.Name)
		}
	case input.CancelledMsg:
		m.pending, m.pendingName, m.addPath = "", "", ""
	case input.SelectedMsg:
		pending := m.pending
		m.pending = ""
		value := strings.TrimSpace(msg.Value)
		if value == "" {
			return nil
		}
		switch pending {
		case "activate":
			name := m.pendingName
			m.pendingName = ""
			path := m.absolutePath(value)
			return func() tea.Msg { return ActivateMsg{Name: name, Path: path} }
		case "add-path":
			m.addPath = m.absolutePath(value)
			m.pending = "add-name"
			return input.ShowWithTitle("Add workspace", "Name: ", filepath.Base(m.addPath))
		case "add-name":
			args := []string{"workspace", "add", "--name", value}
			if revision, ok := m.context.Selection().Highlighted.(common.SelectedRevision); ok {
				args = append(args, "-r", revision.CommitId)
			}
			args = append(args, "--", m.addPath)
			m.addPath = ""
			return m.run("add", value, args)
		case "rename":
			return m.run("rename", value, []string{"workspace", "rename", "--", value})
		}
	case confirmation.CloseMsg:
		m.confirmation = nil
	case confirmation.SelectOptionMsg:
		if m.confirmation != nil {
			return m.confirmation.Update(msg)
		}
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
	}
	return nil
}

func (m *Model) absolutePath(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(m.context.Location, path)
	}
	return filepath.Clean(path)
}

func (m *Model) activate(name string) tea.Cmd {
	if name == m.context.WorkspaceName {
		return nil
	}
	if path, ok := m.paths[name]; ok {
		return func() tea.Msg { return ActivateMsg{Name: name, Path: path} }
	}
	path, err := m.context.RunCommandImmediate([]string{"workspace", "root", "--name", name, "--ignore-working-copy"})
	if err != nil || len(path) == 0 {
		// Old jj versions and older workspaces may have no recorded root.
		m.pending, m.pendingName = "activate", name
		return input.ShowWithTitle("Locate workspace "+name, "Directory: ", "")
	}
	return func() tea.Msg { return ActivateMsg{Name: name, Path: string(path)} }
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	if !m.focused {
		return nil, false
	}
	if m.confirmation != nil {
		switch intent.(type) {
		case intents.Apply, intents.Cancel, intents.OptionSelect:
			return m.confirmation.Update(intent), true
		}
		return nil, false
	}
	if m.busy {
		return nil, true
	}
	switch intent := intent.(type) {
	case intents.WorkspaceNavigate:
		index := slices.IndexFunc(m.workspaces, func(w jj.Workspace) bool { return w.Name == m.context.WorkspaceName })
		if index < 0 || len(m.workspaces) == 0 {
			return nil, true
		}
		index = max(0, min(len(m.workspaces)-1, index+intent.Delta))
		return m.activate(m.workspaces[index].Name), true
	case intents.Apply:
		return func() tea.Msg { return ReturnToRevisionsMsg{} }, true
	case intents.Cancel:
		return common.Close, true
	case intents.WorkspaceAdd:
		m.pending = "add-path"
		return input.ShowWithTitle("Add workspace", "Directory: ", ""), true
	case intents.WorkspaceRename:
		m.pending = "rename"
		return input.ShowWithTitle("Rename active workspace", "Name: ", m.context.WorkspaceName), true
	case intents.WorkspaceForget:
		return m.confirmRemoval("forget"), true
	case intents.WorkspaceRemove:
		return m.confirmRemoval("remove"), true
	case intents.WorkspaceRoot:
		return intents.Invoke(intents.AddMessage{Text: m.context.Location}), true
	case intents.WorkspaceUpdateStale:
		return m.run("update-stale", "", []string{"workspace", "update-stale"}), true
	case intents.WorkspaceRefresh:
		return m.load(), true
	}
	return nil, false
}

func (m *Model) run(operation, name string, args []string) tea.Cmd {
	m.busy = true
	runner := m.context.CommandRunner
	if operation == "forget" || operation == "remove" {
		runner = m.context.RunnerAt(m.launchPath)
	}
	return func() tea.Msg {
		_, err := runner.RunCommandImmediate(args)
		return completedMsg{operation: operation, name: name, err: err}
	}
}

func (m *Model) confirmRemoval(operation string) tea.Cmd {
	if m.context.Location == m.launchPath || m.context.WorkspaceName == "" || m.launchName == "" {
		return message(fmt.Errorf("the launch workspace is protected; activate another workspace first"))
	}
	name := m.context.WorkspaceName
	detail := "Working-copy files will remain on disk."
	if operation == "remove" {
		detail = "The entire workspace directory will be deleted, including ignored and untracked files."
	}
	args := []string{"-R", m.launchPath, "workspace", operation, "--", name}
	// Defer starting the operation until the confirmation is accepted.
	apply := func() tea.Msg {
		return removalConfirmedMsg{operation: operation, name: name, args: args}
	}
	m.confirmation = confirmation.New([]string{
		fmt.Sprintf("%s workspace %s?", strings.ToUpper(operation[:1])+operation[1:], name),
		detail,
		fmt.Sprintf("The active workspace will become %s.", m.launchName),
	},
		confirmation.WithStyleScope("workspace_pane"),
		confirmation.WithZIndex(render.ZDialogs),
		confirmation.WithOption("Cancel", confirmation.Close, key.NewBinding(key.WithKeys("esc"))),
		confirmation.WithOption("Confirm", apply, key.NewBinding(key.WithKeys("enter"))),
	)
	return nil
}

type removalConfirmedMsg struct {
	operation, name string
	args            []string
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}
	text := theme.DefaultPalette.Get("bookmarks", "", "text", false)
	title := theme.DefaultPalette.Get("bookmarks", "", "title", false)
	dimmed := theme.DefaultPalette.Get("bookmarks", "", "dimmed", false)
	dl.AddFill(box.R, ' ', text, render.ZPreview)
	dl.AddInteraction(box.R, PaneClickedMsg{}, render.InteractionClick, render.ZPreview-1)
	top, content := box.CutTop(2)
	dl.Text(top.R.Min.X, top.R.Min.Y, render.ZPreview).Styled(" Workspaces ", title).Done()
	active := slices.IndexFunc(m.workspaces, func(w jj.Workspace) bool { return w.Name == m.context.WorkspaceName })
	m.list.Render(dl, content, len(m.workspaces), active, true,
		func(int) int { return 1 },
		func(dl *render.DisplayContext, index int, rect layout.Rectangle) {
			w := m.workspaces[index]
			style := text
			prefix := "  "
			if w.Name == m.context.WorkspaceName {
				prefix = "● "
				style = theme.DefaultPalette.GetBlended("bookmarks", "", "", true)
				dl.AddFill(rect, ' ', style, render.ZPreview)
			}
			dl.Text(rect.Min.X, rect.Min.Y, render.ZPreview).Styled(" "+prefix+w.Name, style).Done()
		},
		func(index int, _ tea.Mouse) tea.Msg { return ItemClickedMsg{Name: m.workspaces[index].Name} },
	)
	m.list.RegisterScroll(dl, content)
	if m.busy {
		dl.Text(top.R.Min.X, top.R.Min.Y+1, render.ZPreview).Styled(" Working…", dimmed).Done()
	}
}

func (m *Model) RenderOverlay(dl *render.DisplayContext, box layout.Box) {
	if m.confirmation == nil {
		return
	}
	m.confirmation.Styles.Border = theme.DefaultPalette.GetBorder("confirmation", "", "border", false, lipgloss.NormalBorder()).Padding(1)
	w, h := lipgloss.Size(m.confirmation.View())
	frame := layout.Rect(box.R.Min.X+max(0, (box.R.Dx()-w)/2), box.R.Min.Y+max(0, (box.R.Dy()-h)/2), w, h)
	dl.AddBackdrop(box.R, render.ZDialogs-1)
	m.confirmation.ViewRect(dl, layout.Box{R: frame})
}
