package context

import (
	"strings"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/common/models"
)

type ListId int

const (
	ListRevisions ListId = iota
	ListFiles
	ListOplog
	ListEvolog
)

type MainContext struct {
	CommandRunner
	ActiveList     ListId
	Revisions      *RevisionsContext
	Preview        *PreviewContext
	OpLog          *list.List[*models.OperationLogItem]
	Evolog         *list.List[*models.RevisionItem]
	Location       string
	CustomCommands map[string]CustomCommand
	Leader         LeaderMap
	JJConfig       *config.JJConfig
	DefaultRevset  string
	CurrentRevset  string
	Histories      *config.Histories
}

func NewAppContext(location string) *MainContext {
	commandRunner := &MainCommandRunner{
		Location: location,
	}
	m := &MainContext{
		CommandRunner: commandRunner,
		Location:      location,
		Histories:     config.NewHistories(),
		Revisions:     NewRevisionsContext(commandRunner),
		OpLog:         list.NewList[*models.OperationLogItem](),
		Evolog:        list.NewList[*models.RevisionItem](),
		Preview:       NewPreviewContext(commandRunner),
	}
	m.Revisions.Parent = m

	m.JJConfig = &config.JJConfig{}
	if output, err := m.RunCommandImmediate(jj.ConfigListAll()); err == nil {
		m.JJConfig, _ = config.DefaultConfig(output)
	}
	return m
}

// CreateReplacements context aware replacements for custom commands and exec input.
func (ctx *MainContext) CreateReplacements() map[string]string {
	replacements := make(map[string]string)
	replacements[jj.RevsetPlaceholder] = ctx.CurrentRevset

	if current := ctx.Revisions.Revisions.Current(); current != nil {
		replacements[jj.ChangeIdPlaceholder] = current.Commit.ChangeId
		replacements[jj.CommitIdPlaceholder] = current.Commit.CommitId
	}
	if current := ctx.Revisions.Files.Current(); current != nil {
		replacements[jj.FilePlaceholder] = current.FileName
	}
	if current := ctx.OpLog.Current(); current != nil {
		replacements[jj.OperationIdPlaceholder] = current.OperationId
	}

	var checkedRevisions []string
	for _, item := range ctx.Revisions.Revisions.GetCheckedItems() {
		checkedRevisions = append(checkedRevisions, item.Commit.CommitId)
	}

	if len(checkedRevisions) == 0 {
		replacements[jj.CheckedCommitIdsPlaceholder] = "none()"
	} else {
		replacements[jj.CheckedCommitIdsPlaceholder] = strings.Join(checkedRevisions, "|")
	}

	var checkedFiles []string
	for _, item := range ctx.Revisions.Files.GetCheckedItems() {
		checkedFiles = append(checkedFiles, item.FileName)
	}

	if len(checkedFiles) > 0 {
		replacements[jj.CheckedFilesPlaceholder] = strings.Join(checkedFiles, "\t")
	}

	return replacements
}
