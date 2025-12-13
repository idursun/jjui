package context

import (
	"reflect"
	"slices"
	"strings"

	"github.com/idursun/jjui/internal/askpass"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"

	tea "github.com/charmbracelet/bubbletea"
)

type SelectedItem interface {
	Equal(other SelectedItem) bool
}

type SelectedRevision struct {
	ChangeId string
	CommitId string
}

func (s SelectedRevision) Equal(other SelectedItem) bool {
	if o, ok := other.(SelectedRevision); ok {
		return s.ChangeId == o.ChangeId && s.CommitId == o.CommitId
	}
	return false
}

type SelectedFile struct {
	ChangeId string
	CommitId string
	File     string
}

func (s SelectedFile) Equal(other SelectedItem) bool {
	if o, ok := other.(SelectedFile); ok {
		return s.ChangeId == o.ChangeId && s.CommitId == o.CommitId && s.File == o.File
	}
	return false
}

type SelectedOperation struct {
	OperationId string
}

func (s SelectedOperation) Equal(other SelectedItem) bool {
	if o, ok := other.(SelectedOperation); ok {
		return s.OperationId == o.OperationId
	}
	return false
}

type MainContext struct {
	CommandRunner
	SelectedItem   SelectedItem   // Single item where cursor is hover.
	CheckedItems   []SelectedItem // Items checked ✓ by the user.
	Location       string
	CustomCommands map[string]CustomCommand
	Leader         LeaderMap
	JJConfig       *config.JJConfig
	DefaultRevset  string
	CurrentRevset  string
	Histories      *config.Histories
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
	return common.SelectionChanged
}

// CreateReplacements context aware replacements for custom commands and exec input.
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
