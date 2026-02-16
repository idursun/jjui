package context

import (
	"reflect"
	"slices"
	"strings"

	"github.com/idursun/jjui/internal/askpass"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	lua "github.com/yuin/gopher-lua"

	tea "github.com/charmbracelet/bubbletea"
)

// SelectedItem type aliases to break circular dependencies
type SelectedItem = common.SelectedItem
type SelectedRevision = common.SelectedRevision
type SelectedCommit = common.SelectedCommit
type SelectedFile = common.SelectedFile
type SelectedOperation = common.SelectedOperation

type MainContext struct {
	CommandRunner
	SelectedItem  SelectedItem   // Single item where cursor is hover.
	CheckedItems  []SelectedItem // Items checked ✓ by the user.
	Location      string
	JJConfig      *config.JJConfig
	DefaultRevset string
	CurrentRevset string
	Histories     *config.Histories
	ScriptVM      *lua.LState
}

func NewAppContext(location string, aps *askpass.Server) *MainContext {
	m := &MainContext{
		CommandRunner: &MainCommandRunner{
			Location: location,
			Askpass:  aps,
		},
		Location:  location,
		Histories: config.NewHistories(),
	}

	m.JJConfig = &config.JJConfig{}
	if output, err := m.RunCommandImmediate(jj.ConfigListAll()); err == nil {
		m.JJConfig, _ = config.DefaultConfig(output)
	}
	return m
}

func (ctx *MainContext) ClearCheckedItems(ofType reflect.Type) {
	ctx.CheckedItems = slices.DeleteFunc(ctx.CheckedItems, func(i SelectedItem) bool {
		return ofType == nil || ofType == reflect.TypeOf(i)
	})
}

func (ctx *MainContext) AddCheckedItem(item SelectedItem) {
	exists := slices.ContainsFunc(ctx.CheckedItems, func(i SelectedItem) bool {
		return i.Equal(item)
	})
	if !exists {
		ctx.CheckedItems = append(ctx.CheckedItems, item)
	}
}

func (ctx *MainContext) RemoveCheckedItem(item SelectedItem) {
	ctx.CheckedItems = slices.DeleteFunc(ctx.CheckedItems, func(i SelectedItem) bool {
		return i.Equal(item)
	})
}

func (ctx *MainContext) SetSelectedItem(item SelectedItem) tea.Cmd {
	if item == nil {
		return nil
	}
	if item.Equal(ctx.SelectedItem) {
		return nil
	}
	ctx.SelectedItem = item
	return common.SelectionChanged(item)
}

// CreateReplacements creates context-aware replacements for exec input.
func (ctx *MainContext) CreateReplacements() map[string]string {
	selectedItem := ctx.SelectedItem
	replacements := make(map[string]string)
	replacements[jj.RevsetPlaceholder] = ctx.CurrentRevset

	switch selectedItem := selectedItem.(type) {
	case SelectedRevision:
		replacements[jj.ChangeIdPlaceholder] = selectedItem.ChangeId
		replacements[jj.CommitIdPlaceholder] = selectedItem.CommitId
	case SelectedFile:
		replacements[jj.ChangeIdPlaceholder] = selectedItem.ChangeId
		replacements[jj.CommitIdPlaceholder] = selectedItem.CommitId
		replacements[jj.FilePlaceholder] = selectedItem.File
	case SelectedOperation:
		replacements[jj.OperationIdPlaceholder] = selectedItem.OperationId
	}

	var checkedFiles []string
	var checkedRevisions []string
	for _, checked := range ctx.CheckedItems {
		switch c := checked.(type) {
		case SelectedRevision:
			checkedRevisions = append(checkedRevisions, c.CommitId)
		case SelectedFile:
			checkedFiles = append(checkedFiles, c.File)
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

func (ctx *MainContext) ToggleCheckedItem(item SelectedRevision) {
	for i, checked := range ctx.CheckedItems {
		if checked.Equal(item) {
			ctx.CheckedItems = slices.Delete(ctx.CheckedItems, i, i+1)
			return
		}
	}
	ctx.CheckedItems = append(ctx.CheckedItems, item)
}

func (ctx *MainContext) GetSelectedRevisions() map[string]bool {
	selectedRevisions := make(map[string]bool)
	for _, item := range ctx.CheckedItems {
		if rev, ok := item.(SelectedRevision); ok {
			selectedRevisions[rev.ChangeId] = true
		}
	}
	return selectedRevisions
}
