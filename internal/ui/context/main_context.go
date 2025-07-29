package context

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
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
	SelectedItem   SelectedItem
	Location       string
	CustomCommands map[string]CustomCommand
	Leader         LeaderMap
	JJConfig       *config.JJConfig
	DefaultRevset  string
	CurrentRevset  string
	Histories      *config.Histories
}

func NewAppContext(location string) *MainContext {
	m := &MainContext{
		CommandRunner: &MainCommandRunner{
			Location: location,
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

	return replacements
}
