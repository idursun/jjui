package context

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
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
	UI
	Location       string
	CustomCommands map[string]CustomCommand
	Leader         LeaderMap
	JJConfig       *config.JJConfig
	Histories      *config.Histories
	Revisions      *RevisionsContext
	OpLog          *OplogContext
	Revset         *RevsetContext
	Preview        *PreviewContext
	App            *tea.Program
}

type UI interface {
	Update()
	Send(msg tea.Msg)
}

func NewAppContext(location string) *MainContext {
	commandRunner := &MainCommandRunner{
		Location: location,
	}
	m := &MainContext{
		CommandRunner: commandRunner,
		Location:      location,
		Histories:     config.NewHistories(),
	}
	m.Revset = NewRevsetContext()
	m.Revisions = NewRevisionsContext(commandRunner, m, m.Revset)
	m.OpLog = NewOpLogContext(commandRunner, m)
	m.Preview = NewPreviewContext(commandRunner, m, m.Revset)
	m.Revisions.List.AddHandler(func() {
		m.Preview.LoadRevision(m.Revisions.Current())
	})

	m.Revisions.DetailsContext.List.AddHandler(func() {
		m.Preview.LoadRevisionFile(m.Revisions.DetailsContext.Current())
	})

	m.Revisions.EvologContext.List.AddHandler(func() {
		m.Preview.LoadEvolog(m.Revisions.EvologContext.Current())
	})

	m.OpLog.List.AddHandler(func() {
		m.Preview.LoadOpLog(m.OpLog.Current())
	})

	m.JJConfig = &config.JJConfig{}
	if output, err := m.RunCommandImmediate(jj.ConfigListAll()); err == nil {
		m.JJConfig, _ = config.DefaultConfig(output)
	}
	return m
}

func (ctx *MainContext) Update() {
	ctx.Send("")
}

func (ctx *MainContext) Send(msg tea.Msg) {
	ctx.App.Send(msg)
}

// CreateReplacements context aware replacements for custom commands and exec input.
func (ctx *MainContext) CreateReplacements() map[string]string {
	selectedItem := ctx.Revisions.Current()
	replacements := make(map[string]string)
	replacements[jj.RevsetPlaceholder] = ctx.Revset.CurrentRevset
	replacements[jj.ChangeIdPlaceholder] = selectedItem.Commit.ChangeId
	replacements[jj.CommitIdPlaceholder] = selectedItem.Commit.CommitId

	//switch selectedItem := selectedItem.(type) {
	//case models:
	//case SelectedFile:
	//	replacements[jj.ChangeIdPlaceholder] = selectedItem.ChangeId
	//	replacements[jj.CommitIdPlaceholder] = selectedItem.CommitId
	//	replacements[jj.FilePlaceholder] = selectedItem.File
	//case SelectedOperation:
	//	replacements[jj.OperationIdPlaceholder] = selectedItem.OperationId
	//}

	//var checkedFiles []string
	//var checkedRevisions []string
	//for _, checked := range ctx.CheckedItems {
	//	switch c := checked.(type) {
	//	case SelectedRevision:
	//		checkedRevisions = append(checkedRevisions, c.CommitId)
	//	case SelectedFile:
	//		checkedFiles = append(checkedFiles, c.File)
	//	}
	//}
	//
	//if len(checkedFiles) > 0 {
	//	replacements[jj.CheckedFilesPlaceholder] = strings.Join(checkedFiles, "\t")
	//}
	//
	//if len(checkedRevisions) == 0 {
	//	replacements[jj.CheckedCommitIdsPlaceholder] = "none()"
	//} else {
	//	replacements[jj.CheckedCommitIdsPlaceholder] = strings.Join(checkedRevisions, "|")
	//}

	return replacements
}
