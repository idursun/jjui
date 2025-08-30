package context

import (
	"fmt"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/helpers"
	"github.com/idursun/jjui/internal/ui/operations"
)

type RevisionsContext struct {
	*RevisionsList
	context       *MainContext
	Op            operations.Operation
	tag           atomic.Int64
	offScreenRows []parser.Row
	streamer      *GraphStreamer
	hasMore       bool
	op            operations.Operation
}

type RevisionsList struct {
	*helpers.List[parser.Row]
	helpers.ILoadable
}

func NewRevisionsContext() *RevisionsContext {
	return &RevisionsContext{
		RevisionsList: &RevisionsList{
			List: helpers.NewList[parser.Row](),
		},
		Op: operations.NewDefault(),
	}
}

func (c *RevisionsContext) LoadRows(revisionToSelect string) {
	ctx := c.context
	_ = c.tag.Add(1)

	if c.streamer != nil {
		c.streamer.Close()
		c.streamer = nil
	}

	c.List.Items = make([]parser.Row, 0)
	c.hasMore = false

	streamer, err := NewGraphStreamer(ctx, ctx.CurrentRevset)
	if err != nil {
		ctx.App.Send(common.UpdateRevisionsFailedMsg{
			Err:    err,
			Output: fmt.Sprintf("%v", err),
		})
	}
	c.streamer = streamer
	c.hasMore = true
	c.offScreenRows = nil
	//log.Println("Starting streaming revisions with tag:", tag)
	c.RequestMore()
	c.Cursor = c.SelectRevision(revisionToSelect)
}

func (c *RevisionsContext) RequestMore() {
	if c.streamer != nil && c.hasMore {
		batch := c.streamer.RequestMore()
		c.List.Items = append(c.List.Items, batch.Rows...)
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

	idx := slices.IndexFunc(c.List.Items, func(row parser.Row) bool {
		if revision == "@" {
			return row.Commit.IsWorkingCopy
		}
		return eqFold(row.Commit.GetChangeId()) || eqFold(row.Commit.ChangeId) || eqFold(row.Commit.CommitId)
	})
	return idx
}

func (c *RevisionsContext) JumpToParent(revisions jj.SelectedRevisions) {
	immediate, _ := c.context.RunCommandImmediate(jj.GetParent(revisions))
	parentIndex := c.context.Revisions.SelectRevision(string(immediate))
	if parentIndex != -1 {
		c.Cursor = parentIndex
	}
}
