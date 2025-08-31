package context

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context/models"
	"github.com/idursun/jjui/internal/ui/helpers"
	"github.com/idursun/jjui/internal/ui/operations"
)

type RevisionsContext struct {
	*RevisionsList
	*RevsetContext
	CommandRunner
	UI
	DetailsContext *DetailsContext
	EvologContext  *EvologContext
	Op             operations.Operation
	tag            atomic.Int64
	streamer       *GraphStreamer
	hasMore        bool
}

type RevisionsList struct {
	*helpers.CheckableList[*models.RevisionItem]
	helpers.ILoadable
}

func NewRevisionsContext(commandRunner CommandRunner, ui UI, revsetCtx *RevsetContext) *RevisionsContext {
	return &RevisionsContext{
		RevisionsList: &RevisionsList{
			CheckableList: helpers.NewCheckableList[*models.RevisionItem](),
		},
		RevsetContext:  revsetCtx,
		Op:             operations.NewDefault(),
		CommandRunner:  commandRunner,
		UI:             ui,
		DetailsContext: NewDetailsContext(commandRunner),
		EvologContext:  NewEvologContext(commandRunner),
	}
}

func (c *RevisionsContext) LoadRows(revisionToSelect string) {
	_ = c.tag.Add(1)

	if c.streamer != nil {
		c.streamer.Close()
		c.streamer = nil
	}

	c.Items = make([]*models.RevisionItem, 0)
	c.hasMore = false

	streamer, err := NewGraphStreamer(c, c.CurrentRevset)
	if err != nil {
		c.UI.Send(common.UpdateRevisionsFailedMsg{
			Err:    err,
			Output: fmt.Sprintf("%v", err),
		})
	}
	c.streamer = streamer
	c.hasMore = true
	//log.Println("Starting streaming revisions with tag:", tag)
	c.RequestMore()
	c.Cursor = c.SelectRevision(revisionToSelect)
}

func (c *RevisionsContext) RequestMore() {
	if c.streamer != nil && c.hasMore {
		batch := c.streamer.RequestMore()
		for _, row := range batch.Rows {
			c.List.Items = append(c.List.Items, &models.RevisionItem{
				BaseItem: models.BaseItem{Kind: models.Revision},
				Row:      row,
			})
		}
		c.hasMore = batch.HasMore
	}
}

func (c *RevisionsContext) HasMore() bool {
	return c.hasMore
}

func (c *RevisionsContext) Next() {
	if c.Cursor+1 == len(c.Items) && c.HasMore() {
		c.RequestMore()
	}
	c.List.Next()
}

func (c *RevisionsContext) SelectRevision(revision string) int {
	eqFold := func(other string) bool {
		return strings.EqualFold(other, revision)
	}

	if revision == "" {
		if c.Cursor >= 0 && c.Cursor < len(c.List.Items) {
			return c.Cursor
		}
	}

	idx := slices.IndexFunc(c.List.Items, func(item *models.RevisionItem) bool {
		if revision == "@" {
			return item.Row.Commit.IsWorkingCopy
		}
		return eqFold(item.Row.Commit.GetChangeId()) || eqFold(item.Row.Commit.ChangeId) || eqFold(item.Row.Commit.CommitId)
	})
	return idx
}

func (c *RevisionsContext) JumpToParent(revisions jj.SelectedRevisions) {
	immediate, _ := c.RunCommandImmediate(jj.GetParent(revisions))
	parentIndex := c.SelectRevision(string(immediate))
	if parentIndex != -1 {
		c.Cursor = parentIndex
	}
}

func (c *RevisionsContext) AsRows() []parser.Row {
	rows := make([]parser.Row, len(c.Items))
	for i, item := range c.Items {
		rows[i] = item.Row
	}
	return rows
}

func (c *RevisionsContext) AddCheckedItem(sel SelectedFile) {

}

func (c *RevisionsContext) SetSelectedItem(first SelectedItem) tea.Cmd {
	return nil
}

func (c *RevisionsContext) ClearCheckedItems(typeFor reflect.Type) {

}

func (c *RevisionsContext) RemoveCheckedItem(file SelectedFile) {

}
