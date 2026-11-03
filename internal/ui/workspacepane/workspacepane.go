package workspacepane

import (
	"fmt"
	"maps"
	"os"
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
type RevealRevisionMsg struct{ Name, CommitID string }
type PaneClickedMsg struct{}
type ItemClickedMsg struct{ Name string }

// Message is routed to the retained panel even if another pane is now visible.
type Message interface{ workspacePaneMessage() }

type loadedMsg struct {
	location   string
	generation uint64
	workspaces []jj.Workspace
	paths      map[string]string
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
	selectedName string
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
	m := &Model{context: ctx, launchPath: ctx.Location, launchName: ctx.WorkspaceName, selectedName: ctx.WorkspaceName, paths: make(map[string]string)}
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

func (m *Model) Init() tea.Cmd { return nil }
func (m *Model) OnShow() tea.Cmd {
	m.selectedName = m.context.WorkspaceName
	return m.load()
}
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
	activeName, knownPaths := m.context.WorkspaceName, maps.Clone(m.paths)
	m.generation++
	generation := m.generation
	return func() tea.Msg {
		output, err := runner.RunCommandImmediate(jj.WorkspaceList())
		if err != nil {
			return loadedMsg{location: location, generation: generation, err: err}
		}
		workspaces := jj.ParseWorkspaces(string(output))
		paths := make(map[string]string, len(workspaces))
		for _, workspace := range workspaces {
			if workspace.Name == activeName {
				paths[workspace.Name] = location
			} else if path, err := resolveWorkspacePath(runner, location, workspace.Name); err == nil {
				paths[workspace.Name] = path
			} else if path := knownPaths[workspace.Name]; path != "" {
				paths[workspace.Name] = path
			}
		}
		return loadedMsg{location: location, generation: generation, workspaces: workspaces, paths: paths}
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
		if msg.paths != nil {
			m.paths = msg.paths
		}
		m.context.IdentifyWorkspace(msg.workspaces)
		if !slices.ContainsFunc(m.workspaces, func(w jj.Workspace) bool { return w.Name == m.selectedName }) {
			m.selectedName = m.context.WorkspaceName
		}
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
			if m.selectedName == m.context.WorkspaceName {
				m.selectedName = msg.name
			}
			delete(m.paths, m.context.WorkspaceName)
			m.context.WorkspaceName = msg.name
			m.paths[msg.name] = m.context.Location
			if m.context.Location == m.launchPath {
				m.launchName = msg.name
			}
		case "forget", "remove":
			delete(m.paths, msg.name)
			if msg.name == m.context.WorkspaceName {
				return func() tea.Msg { return ActivateMsg{Name: m.launchName, Path: m.launchPath} }
			}
		}
		return common.RefreshAndKeepSelections
	case common.RefreshMsg:
		return m.load()
	case ItemClickedMsg:
		if !m.busy && m.confirmation == nil {
			if slices.ContainsFunc(m.workspaces, func(w jj.Workspace) bool { return w.Name == msg.Name }) {
				m.selectedName = msg.Name
			}
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
	if name == "" || name == m.context.WorkspaceName {
		return nil
	}
	path, err := m.workspacePath(name)
	if err != nil {
		m.pending, m.pendingName = "activate", name
		return input.ShowWithTitle("Locate workspace "+name, "Directory: ", "")
	}
	return func() tea.Msg { return ActivateMsg{Name: name, Path: path} }
}

func (m *Model) workspacePath(name string) (string, error) {
	if name == m.context.WorkspaceName {
		return m.context.Location, nil
	}
	if path, ok := m.paths[name]; ok {
		return path, nil
	}
	return resolveWorkspacePath(m.context.CommandRunner, m.context.Location, name)
}

func resolveWorkspacePath(runner context.CommandRunner, location, name string) (string, error) {
	path, err := runner.RunCommandImmediate([]string{"workspace", "root", "--name", name, "--ignore-working-copy"})
	if err != nil || len(path) == 0 {
		if primary := primaryWorkspacePath(runner, location, name); primary != "" {
			return primary, nil
		}
		// Old jj versions and older workspaces may have no recorded root.
		if err != nil {
			return "", fmt.Errorf("cannot locate workspace %s: %w", name, err)
		}
		return "", fmt.Errorf("workspace %s has no recorded directory", name)
	}
	return string(path), nil
}

// Older repositories may not record the primary workspace's path in jj's
// workspace list. Secondary workspaces still point to its .jj/repo directory.
// Verify its workspace name rather than assuming the primary is called default.
func primaryWorkspacePath(runner context.CommandRunner, location, name string) string {
	metadata := filepath.Join(location, ".jj")
	pointer, err := os.ReadFile(filepath.Join(metadata, "repo"))
	if err != nil {
		return ""
	}
	repoPath := strings.TrimSpace(string(pointer))
	if !filepath.IsAbs(repoPath) {
		repoPath = filepath.Join(metadata, repoPath)
	}
	repoPath, err = filepath.EvalSymlinks(repoPath)
	if err != nil || filepath.Base(repoPath) != "repo" || filepath.Base(filepath.Dir(repoPath)) != ".jj" {
		return ""
	}
	primary := filepath.Dir(filepath.Dir(repoPath))
	if primary == location {
		return ""
	}
	output, err := runner.RunCommandImmediate(append([]string{"-R", primary}, jj.WorkspaceList()...))
	if err != nil {
		return ""
	}
	currentName := ""
	for _, workspace := range jj.ParseWorkspaces(string(output)) {
		if workspace.Current {
			if currentName != "" {
				return ""
			}
			currentName = workspace.Name
		}
	}
	if currentName != name {
		return ""
	}
	return primary
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
		if len(m.workspaces) == 0 {
			return nil, true
		}
		index := slices.IndexFunc(m.workspaces, func(w jj.Workspace) bool { return w.Name == m.selectedName })
		if index < 0 {
			index = 0
		}
		index = max(0, min(len(m.workspaces)-1, index+intent.Delta))
		m.selectedName = m.workspaces[index].Name
		return nil, true
	case intents.Apply:
		return m.activate(m.selectedName), true
	case intents.WorkspaceReveal:
		if m.selectedName == "" {
			return nil, true
		}
		out, err := m.context.RunCommandImmediate(jj.ResolveRevisionID(fmt.Sprintf("%q@", m.selectedName)))
		if err != nil {
			return message(err), true
		}
		_, commitID, ok := strings.Cut(strings.TrimSpace(string(out)), ";")
		if !ok || commitID == "" {
			return message(fmt.Errorf("workspace %s has no working-copy revision", m.selectedName)), true
		}
		name := m.selectedName
		return func() tea.Msg { return RevealRevisionMsg{Name: name, CommitID: commitID} }, true
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
	case intents.WorkspaceUpdateStale:
		if m.selectedName == "" {
			return nil, true
		}
		path, err := m.workspacePath(m.selectedName)
		if err != nil {
			return message(err), true
		}
		return m.run("update-stale", m.selectedName, []string{"-R", path, "workspace", "update-stale"}), true
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
	name := m.selectedName
	if name == "" {
		return nil
	}
	if m.launchName == "" {
		return message(fmt.Errorf("cannot identify the launch workspace; refresh before removing a workspace"))
	}
	if name == m.launchName || m.paths[name] == m.launchPath {
		return message(fmt.Errorf("the launch workspace %s is protected; select another workspace", m.launchName))
	}
	detail := "Working-copy files will remain on disk."
	if operation == "remove" {
		detail = "The entire workspace directory will be deleted, including ignored and untracked files."
	}
	args := []string{"-R", m.launchPath, "workspace", operation, "--", name}
	// Defer starting the operation until the confirmation is accepted.
	apply := func() tea.Msg {
		return removalConfirmedMsg{operation: operation, name: name, args: args}
	}
	activeMessage := fmt.Sprintf("The active workspace will remain %s.", m.context.WorkspaceName)
	if name == m.context.WorkspaceName {
		activeMessage = fmt.Sprintf("The active workspace will become %s.", m.launchName)
	}
	m.confirmation = confirmation.New([]string{
		fmt.Sprintf("%s workspace %s?", strings.ToUpper(operation[:1])+operation[1:], name),
		detail,
		activeMessage,
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
	cursor := slices.IndexFunc(m.workspaces, func(w jj.Workspace) bool { return w.Name == m.selectedName })
	m.list.Render(dl, content, len(m.workspaces), cursor, true,
		func(int) int { return 1 },
		func(dl *render.DisplayContext, index int, rect layout.Rectangle) {
			w := m.workspaces[index]
			style := text
			pathStyle := dimmed
			prefix := "  "
			if w.Name == m.context.WorkspaceName {
				prefix = "● "
			}
			if w.Name == m.selectedName && m.focused {
				style = theme.DefaultPalette.GetBlended("bookmarks", "", "", true)
				pathStyle = theme.DefaultPalette.GetBlended("bookmarks", "", "dimmed", true)
				dl.AddFill(rect, ' ', style, render.ZPreview)
			}
			path := m.paths[w.Name]
			if path == "" {
				path = "(path unavailable)"
			}
			dl.Text(rect.Min.X, rect.Min.Y, render.ZPreview).
				Styled(" "+prefix+w.Name, style).
				Styled("  "+path, pathStyle).
				Done()
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
