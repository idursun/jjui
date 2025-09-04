package context

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/common/models"
)

type RevisionsContext struct {
	Parent           *MainContext
	Revisions        *list.CheckableList[*models.RevisionItem]
	Files            *list.CheckableList[*models.RevisionFileItem]
	tag              atomic.Uint64
	revisionToSelect string
	offScreenRows    []*models.RevisionItem
	streamer         *GraphStreamer
	hasMore          bool
}

func (m *RevisionsContext) LoadStreaming(revset string, selectedRevision string) tea.Cmd {
	currentTag := m.tag.Add(1)

	if m.streamer != nil {
		m.streamer.Close()
		m.streamer = nil
	}

	m.hasMore = false

	streamer, err := NewGraphStreamer(m.Parent.CommandRunner, revset)
	if err != nil {
		return func() tea.Msg {
			return common.UpdateRevisionsFailedMsg{
				Err:    err,
				Output: fmt.Sprintf("%v", err),
			}
		}
	}
	m.streamer = streamer
	m.hasMore = true
	m.offScreenRows = nil
	m.revisionToSelect = selectedRevision

	// If the revision to select is not set, use the currently selected item
	if m.revisionToSelect == "" {
		if current := m.Revisions.Current(); current != nil {
			m.revisionToSelect = m.Revisions.Current().Commit.GetChangeId()
		}
	}

	for m.hasMore && (m.Revisions.Cursor == -1 || m.Revisions.Cursor >= len(m.offScreenRows)) {
		if currentTag != m.tag.Load() {
			// cancel loading if a new request has been made
			return nil
		}
		batch := m.streamer.RequestMore()
		m.offScreenRows = append(m.offScreenRows, batch.Items...)
		m.hasMore = batch.HasMore
	}

	if !m.hasMore {
		m.streamer.Close()
		m.streamer = nil
	}

	return func() tea.Msg {
		return AppendRowsBatchMsg{
			Rows:    m.offScreenRows,
			HasMore: m.hasMore,
		}
	}
}

func (m *RevisionsContext) RequestMoreRows() tea.Cmd {
	currentTag := m.tag.Add(1)
	return func() tea.Msg {
		if m.streamer == nil || !m.hasMore || currentTag != m.tag.Load() {
			return nil
		}
		batch := m.streamer.RequestMore()
		return AppendRowsBatchMsg{batch.Items, batch.HasMore}
	}
}

func (m *RevisionsContext) Load(revset string, selectedRevision string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.Parent.RunCommandImmediate(jj.Log(revset, config.Current.Limit))
		if err != nil {
			return common.UpdateRevisionsFailedMsg{
				Err:    err,
				Output: string(output),
			}
		}
		rows := parser.ParseRows(bytes.NewReader(output))
		return UpdateRevisionsMsg{rows, selectedRevision}
	}
}

func (m *RevisionsContext) CursorDown() tea.Cmd {
	if m.Revisions.Cursor < len(m.Revisions.Items)-1 {
		m.Revisions.Cursor++
	} else if m.hasMore {
		return m.RequestMoreRows()
	}
	return nil
}

func (m *RevisionsContext) SelectRevision(revision string) int {
	eqFold := func(other string) bool {
		return strings.EqualFold(other, revision)
	}

	idx := slices.IndexFunc(m.Revisions.Items, func(row *models.RevisionItem) bool {
		if revision == "@" {
			return row.Commit.IsWorkingCopy
		}
		return eqFold(row.Commit.GetChangeId()) || eqFold(row.Commit.ChangeId) || eqFold(row.Commit.CommitId)
	})
	return idx
}

func (m *RevisionsContext) JumpToParent(revisions jj.SelectedRevisions) {
	immediate, _ := m.Parent.RunCommandImmediate(jj.GetParent(revisions))
	parentIndex := m.SelectRevision(string(immediate))
	if parentIndex != -1 {
		m.Revisions.Cursor = parentIndex
	}
}

type AppendRowsBatchMsg struct {
	Rows    []*models.RevisionItem
	HasMore bool
}

type UpdateRevisionsMsg struct {
	Rows             []*models.RevisionItem
	SelectedRevision string
}
