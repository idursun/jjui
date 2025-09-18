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
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
)

type RevisionsContext struct {
	CommandRunner
	*list.CheckableList[*models.RevisionItem]
	tag              atomic.Uint64
	revisionToSelect string
	offScreenRows    []*models.RevisionItem
	streamer         *GraphStreamer
	hasMore          bool
}

func NewRevisionsContext(ctx *MainContext) *RevisionsContext {
	return &RevisionsContext{
		CommandRunner: ctx.CommandRunner,
		CheckableList: list.NewCheckableList[*models.RevisionItem](),
	}
}

func (m *RevisionsContext) LoadStreaming(revset string, selectedRevision string) tea.Cmd {
	currentTag := m.tag.Add(1)

	if m.streamer != nil {
		m.streamer.Close()
		m.streamer = nil
	}

	m.hasMore = false

	streamer, err := NewGraphStreamer(m.CommandRunner, revset)
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
		if current := m.Current(); current != nil {
			m.revisionToSelect = m.Current().Commit.GetChangeId()
		}
	}

	for m.hasMore && (m.Cursor == -1 || m.Cursor >= len(m.offScreenRows)) {
		if currentTag != m.tag.Load() {
			// cancel loading if a new request has been made
			return nil
		}
		batch := m.streamer.RequestMore()
		m.offScreenRows = append(m.offScreenRows, batch.Items...)
		m.hasMore = batch.HasMore

		// if it is initial load and there are no items, break to avoid loading all items at once
		if len(m.List.Items) == 0 && m.Cursor == -1 && len(m.offScreenRows) > 0 {
			break
		}
	}

	if !m.hasMore {
		m.streamer.Close()
		m.streamer = nil
	}

	if m.tag.Load() == currentTag {
		m.SetItems(m.offScreenRows)
		m.offScreenRows = nil
		if m.revisionToSelect != "" {
			idx := m.FindRevision(m.revisionToSelect)
			if idx != -1 {
				m.SetCursor(idx)
			}
			m.revisionToSelect = ""
		}

		if m.Cursor == -1 && len(m.Items) > 0 || m.Cursor >= len(m.Items) {
			m.Cursor = 0
		}
	}

	return func() tea.Msg {
		return UpdateRevisionsMsg{}
	}
}

func (m *RevisionsContext) requestMoreRows() tea.Cmd {
	currentTag := m.tag.Load()
	return func() tea.Msg {
		if m.streamer == nil || !m.hasMore || currentTag != m.tag.Load() {
			return nil
		}
		batch := m.streamer.RequestMore()
		m.AppendItems(batch.Items...)

		return UpdateRevisionsMsg{}
	}
}

func (m *RevisionsContext) Load(revset string, selectedRevision string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.RunCommandImmediate(jj.Args(jj.LogArgs{
			Revset:          revset,
			Limit:           config.Current.Limit,
			Template:        config.Current.Revisions.Template,
			GlobalArguments: jj.GlobalArguments{Color: "always"},
		}))
		if err != nil {
			return common.UpdateRevisionsFailedMsg{
				Err:    err,
				Output: string(output),
			}
		}
		rows := parser.ParseRows(bytes.NewReader(output))
		m.SetItems(rows)
		idx := m.FindRevision(selectedRevision)
		if idx != -1 {
			m.SetCursor(idx)
		}
		return UpdateRevisionsMsg{}
	}
}

func (m *RevisionsContext) CursorUp() tea.Cmd {
	if m.Cursor > 0 {
		m.Cursor--
	}
	return nil
}

func (m *RevisionsContext) CursorDown() tea.Cmd {
	if m.Cursor < len(m.Items)-1 {
		m.Cursor++
	} else if m.hasMore {
		return m.requestMoreRows()
	}
	return nil
}

func (m *RevisionsContext) FindRevision(revision string) int {
	eqFold := func(other string) bool {
		return strings.EqualFold(other, revision)
	}

	idx := slices.IndexFunc(m.Items, func(row *models.RevisionItem) bool {
		if revision == "@" {
			return row.Commit.IsWorkingCopy
		}
		return eqFold(row.Commit.GetChangeId()) || eqFold(row.Commit.ChangeId) || eqFold(row.Commit.CommitId)
	})
	return idx
}

func (m *RevisionsContext) JumpToParent(revisions jj.SelectedRevisions) {
	immediate, _ := m.RunCommandImmediate(jj.GetParent(revisions).GetArgs())
	parentIndex := m.FindRevision(string(immediate))
	if parentIndex != -1 {
		m.Cursor = parentIndex
	}
}

func (m *RevisionsContext) GetCommitIds() []string {
	var commitIds []string
	for _, row := range m.Items {
		commitIds = append(commitIds, row.Commit.CommitId)
	}
	return commitIds
}

type AppendRowsBatchMsg struct {
	Rows    []*models.RevisionItem
	HasMore bool
}

type UpdateRevisionsMsg struct {
	SelectedRevision string
}
