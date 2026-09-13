package context

import (
	"os"
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
	if runner, ok := ctx.CommandRunner.(*MainCommandRunner); ok {
		runner.Location = path
	}
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
