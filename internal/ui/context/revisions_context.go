package context

import (
	"fmt"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/operations"
)

type RevisionsContext struct {
	context          *MainContext
	Rows             []parser.Row
	Op               operations.Operation
	Checked          []parser.Row
	Cursor           int
	tag              atomic.Int64
	revisionToSelect string
	offScreenRows    []parser.Row
	streamer         *GraphStreamer
	hasMore          bool
	op               operations.Operation
}

func (c *RevisionsContext) LoadRows(revisionToSelect string) {
	ctx := c.context
	_ = c.tag.Add(1)

	if c.streamer != nil {
		c.streamer.Close()
		c.streamer = nil
	}

	c.Rows = make([]parser.Row, 0)
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
	c.requestMoreRows()
	c.Cursor = c.SelectRevision(revisionToSelect)
}

func (c *RevisionsContext) requestMoreRows() {
	if c.streamer != nil && c.hasMore {
		batch := c.streamer.RequestMore()
		c.Rows = append(c.Rows, batch.Rows...)
		c.hasMore = batch.HasMore
	}
}

func (c *RevisionsContext) SelectRevision(revision string) int {
	eqFold := func(other string) bool {
		return strings.EqualFold(other, revision)
	}

	if revision == "" {
		if c.Cursor >= 0 && c.Cursor < len(c.Rows) {
			return c.Cursor
		}
	}

	idx := slices.IndexFunc(c.Rows, func(row parser.Row) bool {
		if revision == "@" {
			return row.Commit.IsWorkingCopy
		}
		return eqFold(row.Commit.GetChangeId()) || eqFold(row.Commit.ChangeId) || eqFold(row.Commit.CommitId)
	})
	return idx
}

func (c *RevisionsContext) Next() {
	if len(c.Rows) == 0 {
		c.Cursor = -1
		return
	}
	if c.Cursor < len(c.Rows)-1 {
		c.Cursor++
	} else if c.hasMore {
		c.requestMoreRows()
	}
}

func (c *RevisionsContext) Prev() {
	if len(c.Rows) == 0 {
		c.Cursor = -1
		return
	}
	if c.Cursor > 0 {
		c.Cursor--
	}
}

func (c *RevisionsContext) JumpToParent(revisions jj.SelectedRevisions) {
	immediate, _ := c.context.RunCommandImmediate(jj.GetParent(revisions))
	parentIndex := c.context.Revisions.SelectRevision(string(immediate))
	if parentIndex != -1 {
		c.Cursor = parentIndex
	}
}
