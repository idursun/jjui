package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/idursun/jjui/internal/askpass"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	lua "github.com/yuin/gopher-lua"
)

// SelectedItem type aliases to break circular dependencies
type (
	SelectedItem      = common.SelectedItem
	SelectedRevision  = common.SelectedRevision
	SelectedCommit    = common.SelectedCommit
	SelectedFile      = common.SelectedFile
	SelectedOperation = common.SelectedOperation
)

type MainContext struct {
	CommandRunner
	stateProvider             common.ApplicationStateProvider
	Location                  string
	WorkspaceName             string
	repositoryPath            string
	WorkingDirectory          string
	JJConfig                  *config.JJConfig
	DefaultRevset             string
	CurrentRevset             string
	TerminalHasDarkBackground bool
	TerminalThemeDetected     bool
	TerminalBackground        string
	TerminalPalette           map[int]string
	ThemeBackgroundBlend      float64
	Histories                 *config.Histories
	ScriptVM                  *lua.LState
	TerminalWidth             int
	TerminalHeight            int
}

// SetStateProvider connects the context to the root's live state resolver.
// The provider is intentionally optional because the Lua VM is created before
// the UI model is constructed.
func (ctx *MainContext) SetStateProvider(provider common.ApplicationStateProvider) {
	ctx.stateProvider = provider
}

func (ctx *MainContext) QueryState(name string) (any, bool) {
	if ctx.stateProvider == nil {
		return nil, false
	}
	return ctx.stateProvider.QueryState(name)
}

func (ctx *MainContext) Selection() common.SelectionSnapshot {
	if ctx.stateProvider == nil {
		return common.SelectionSnapshot{}
	}
	return ctx.stateProvider.Selection()
}

func NewAppContext(location string, aps *askpass.Server) *MainContext {
	workingDirectory, _ := os.Getwd()
	m := &MainContext{
		CommandRunner: &MainCommandRunner{
			Location: location,
			Askpass:  aps,
		},
		Location:         location,
		WorkingDirectory: workingDirectory,
		Histories:        config.NewHistories(),
	}

	m.JJConfig = &config.JJConfig{}
	if output, err := m.RunCommandImmediate(jj.ConfigListAll()); err == nil {
		m.JJConfig, _ = config.DefaultConfig(output)
	}
	return m
}

// SetTerminalSize records the current terminal dimensions for width-sensitive
// subprocesses. Full-screen diffs capture `jj` output through a pipe, so
// tools like difftastic cannot probe the TTY width and fall back to 80
// columns unless DFT_WIDTH/COLUMNS are provided.
func (ctx *MainContext) SetTerminalSize(width, height int) {
	ctx.TerminalWidth = width
	ctx.TerminalHeight = height
}

// DiffEnv exposes the terminal size to width-sensitive diff tools.
// Returns nil when the terminal size is unknown, preserving old behavior.
func (ctx *MainContext) DiffEnv() []string {
	if ctx.TerminalWidth <= 0 {
		return nil
	}
	height := ctx.TerminalHeight
	if height <= 0 {
		height = 24
	}
	return []string{
		"DFT_WIDTH=" + strconv.Itoa(ctx.TerminalWidth),
		"COLUMNS=" + strconv.Itoa(ctx.TerminalWidth),
		"LINES=" + strconv.Itoa(height),
	}
}

// CreateReplacements creates context-aware replacements for exec input.
func (ctx *MainContext) CreateReplacements() map[string]string {
	snapshot := ctx.Selection()
	selectedItem := snapshot.Highlighted
	replacements := make(map[string]string)
	replacements[jj.RevsetPlaceholder] = ctx.CurrentRevset

	switch selectedItem := selectedItem.(type) {
	case SelectedRevision:
		replacements[jj.ChangeIdPlaceholder] = selectedItem.ChangeId
		replacements[jj.CommitIdPlaceholder] = selectedItem.CommitId
	case SelectedFile:
		replacements[jj.ChangeIdPlaceholder] = selectedItem.ChangeId
		replacements[jj.CommitIdPlaceholder] = selectedItem.CommitId
		replacements[jj.FilePlaceholder] = selectedItem.File.Path()
	case SelectedOperation:
		replacements[jj.OperationIdPlaceholder] = selectedItem.OperationId
	}

	var checkedFiles []string
	var checkedRevisions []string
	for _, checked := range snapshot.Checked {
		switch c := checked.(type) {
		case SelectedRevision:
			checkedRevisions = append(checkedRevisions, c.CommitId)
		case SelectedFile:
			checkedFiles = append(checkedFiles, c.File.Path())
		}
	}

	if len(checkedFiles) > 0 {
		replacements[jj.CheckedFilesPlaceholder] = strings.Join(checkedFiles, "\t")
	}

	if len(checkedRevisions) == 0 {
		replacements[jj.CheckedCommitIdsPlaceholder] = "none()"
	} else {
		replacements[jj.CheckedCommitIdsPlaceholder] = strings.Join(checkedRevisions, "|")
	}

	return replacements
}

func (ctx *MainContext) ChangeWorkspace(path string) {
	ctx.Location = path
	ctx.CommandRunner = ctx.RunnerAt(path)
	ctx.WorkspaceName = ""
	ctx.repositoryPath = ""
}

// RunnerAt leaves existing commands attached to their original directory.
func (ctx *MainContext) RunnerAt(path string) CommandRunner {
	if runner, ok := ctx.CommandRunner.(*MainCommandRunner); ok {
		return &MainCommandRunner{Location: path, Askpass: runner.Askpass}
	}
	return ctx.CommandRunner
}

// IdentifyWorkspace uses the current-working-copy hint only when it is unambiguous.
// Multiple workspaces can point at the same revision.
func (ctx *MainContext) IdentifyWorkspace(workspaces []jj.Workspace) {
	var candidates []string
	for _, workspace := range workspaces {
		if workspace.Path != "" && filepath.Clean(workspace.Path) == filepath.Clean(ctx.Location) {
			ctx.WorkspaceName = workspace.Name
			return
		}
		if workspace.Current {
			candidates = append(candidates, workspace.Name)
		}
	}
	if len(candidates) == 1 {
		ctx.WorkspaceName = candidates[0]
		return
	}
	for _, name := range candidates {
		out, err := ctx.RunCommandImmediate([]string{"workspace", "root", "--name", name, "--ignore-working-copy"})
		if err == nil && filepath.Clean(string(out)) == filepath.Clean(ctx.Location) {
			ctx.WorkspaceName = name
			return
		}
	}
}

// ActivateWorkspace validates the destination before changing any live context.
func (ctx *MainContext) ActivateWorkspace(name, path string) error {
	runner := ctx.RunnerAt(path)
	out, err := runner.RunCommandImmediate([]string{"-R", path, "root", "--ignore-working-copy"})
	if err != nil {
		return err
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return fmt.Errorf("workspace %s has no root directory", name)
	}
	// A manually located directory must belong to this repository, not just
	// contain a workspace with the same name in an unrelated repository.
	repositoryPath := ctx.repositoryPath
	if repositoryPath == "" {
		before, err := ctx.RunCommandImmediate([]string{"git", "root", "--ignore-working-copy"})
		if err != nil {
			return err
		}
		repositoryPath = filepath.Clean(string(before))
	}
	after, err := runner.RunCommandImmediate([]string{"-R", root, "git", "root", "--ignore-working-copy"})
	if err != nil {
		return err
	}
	if repositoryPath != filepath.Clean(string(after)) {
		return fmt.Errorf("%s belongs to a different repository", root)
	}
	// Check the name as well, including paths supplied for older jj versions
	// that cannot resolve another workspace's root by name.
	out, err = runner.RunCommandImmediate(append([]string{"-R", root}, jj.WorkspaceList()...))
	if err != nil {
		return err
	}
	matched := false
	currentCount := 0
	for _, workspace := range jj.ParseWorkspaces(string(out)) {
		if workspace.Current {
			currentCount++
		}
		if workspace.Name == name && workspace.Current {
			matched = true
		}
	}
	if !matched {
		return fmt.Errorf("%s is not workspace %s", root, name)
	}
	if currentCount > 1 {
		actualRoot, err := runner.RunCommandImmediate([]string{"-R", root, "workspace", "root", "--name", name, "--ignore-working-copy"})
		if err != nil {
			return fmt.Errorf("cannot verify workspace %s: %w", name, err)
		}
		if filepath.Clean(string(actualRoot)) != filepath.Clean(root) {
			return fmt.Errorf("%s is not workspace %s", root, name)
		}
	}
	out, err = runner.RunCommandImmediate(append([]string{"-R", root}, jj.ConfigListAll()...))
	if err != nil {
		return err
	}
	jjConfig, err := config.DefaultConfig(out)
	if err != nil {
		return err
	}
	ctx.ChangeWorkspace(root)
	ctx.WorkspaceName = name
	ctx.repositoryPath = repositoryPath
	ctx.WorkingDirectory = root
	ctx.JJConfig = jjConfig
	return nil
}

func (ctx *MainContext) GetSelectedRevisions() map[string]bool {
	selectedRevisions := make(map[string]bool)
	for _, item := range ctx.Selection().Checked {
		if rev, ok := item.(SelectedRevision); ok {
			selectedRevisions[rev.ChangeId] = true
		}
	}
	return selectedRevisions
}
