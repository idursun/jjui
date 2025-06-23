package operations

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
)

type RenderPosition int

const (
	RenderPositionNil RenderPosition = iota
	RenderPositionAfter
	RenderPositionBefore
	RenderBeforeChangeId
	RenderBeforeCommitId
	RenderOverDescription
)

type Operation interface {
	Render(commit *jj.Commit, renderPosition RenderPosition) string
	Name() string
}

type OperationWithOverlay interface {
	Operation
	Update(msg tea.Msg) (OperationWithOverlay, tea.Cmd)
}

type TracksSelectedRevision interface {
	SetSelectedRevision(commit *jj.Commit)
}

type HandleKey interface {
	HandleKey(msg tea.KeyMsg) tea.Cmd
}
