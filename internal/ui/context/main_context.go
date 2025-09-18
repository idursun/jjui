package context

import (
	"strings"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common/list"
)

type MainContext struct {
	CommandRunner
	OpLog          *list.List[*models.OperationLogItem]
	Revisions      *RevisionsContext
	Evolog         *list.List[*models.RevisionItem]
	Files          *DetailsContext
	Preview        *PreviewContext
	Location       string
	CustomCommands map[string]CustomCommand
	Leader         LeaderMap
	JJConfig       *config.JJConfig
	DefaultRevset  string
	CurrentRevset  string
	Histories      *config.Histories
}

func NewAppContext(commandRunner CommandRunner, location string) *MainContext {
	m := &MainContext{
		CommandRunner: commandRunner,
		Location:      location,
		Histories:     config.NewHistories(),
		OpLog:         list.NewList[*models.OperationLogItem](),
	}
	m.Revisions = NewRevisionsContext(m)
	m.Files = NewDetailsContext(m)
	m.Evolog = list.NewList[*models.RevisionItem]()
	m.Preview = NewPreviewContext(commandRunner)

	m.JJConfig = &config.JJConfig{}
	if output, err := m.RunCommandImmediate(jj.Args(jj.ConfigListAllArgs{})); err == nil {
		m.JJConfig, _ = config.DefaultConfig(output)
	}
	return m
}

// CreateReplacements context aware replacements for custom commands and exec input.
func (ctx *MainContext) CreateReplacements() map[string]string {
	replacements := make(map[string]string)
	replacements[jj.RevsetPlaceholder] = ctx.CurrentRevset

	if current := ctx.Revisions.Current(); current != nil {
		replacements[jj.ChangeIdPlaceholder] = current.Commit.ChangeId
		replacements[jj.CommitIdPlaceholder] = current.Commit.CommitId
	}
	if current := ctx.Files.Current(); current != nil {
		replacements[jj.FilePlaceholder] = current.FileName
	}
	if current := ctx.OpLog.Current(); current != nil {
		replacements[jj.OperationIdPlaceholder] = current.OperationId
	}
	if current := ctx.Evolog.Current(); current != nil {
		replacements[jj.CommitIdPlaceholder] = current.Commit.CommitId
	}

	var checkedRevisions []string
	for _, item := range ctx.Revisions.GetCheckedItems() {
		checkedRevisions = append(checkedRevisions, item.Commit.CommitId)
	}

	if len(checkedRevisions) == 0 {
		replacements[jj.CheckedCommitIdsPlaceholder] = "none()"
	} else {
		replacements[jj.CheckedCommitIdsPlaceholder] = strings.Join(checkedRevisions, "|")
	}

	var checkedFiles []string
	for _, item := range ctx.Files.GetCheckedItems() {
		checkedFiles = append(checkedFiles, item.FileName)
	}

	if len(checkedFiles) > 0 {
		replacements[jj.CheckedFilesPlaceholder] = strings.Join(checkedFiles, "\t")
	}

	return replacements
}
