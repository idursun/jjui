package view

import tea "github.com/charmbracelet/bubbletea"

type ViewId string

const (
	RevisionsViewId ViewId = "revisions"
	RevsetViewId    ViewId = "revset"
	PreviewViewId   ViewId = "preview"
	StatusViewId    ViewId = "status"
	DetailsViewId   ViewId = "details"
	EvologViewId    ViewId = "evolog"
	OpLogViewId     ViewId = "oplog"
	DiffViewId      ViewId = "diff"
)

type IView interface {
	GetId() ViewId
}

type HasSize interface {
	GetSize() *Sizeable
}

var _ IView = (*BaseView)(nil)

type ViewOpts struct {
	*Sizeable
	Id            ViewId
	NeedsRefresh  bool
	KeyDelegation *ViewId
	Visible       bool
}

type BaseView struct {
	*ViewOpts
	Model tea.Model
}

func (b *BaseView) GetId() ViewId {
	return b.Id
}
