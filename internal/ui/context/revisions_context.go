package context

import (
	"bufio"
	"bytes"
	"fmt"
	"path"
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
	"github.com/idursun/jjui/internal/ui/operations"
)

type RevisionsContext struct {
	CommandRunner
	Parent           *MainContext
	Revisions        *list.CheckableList[*models.RevisionItem]
	Files            *list.CheckableList[*models.RevisionFileItem]
	Op               operations.Operation
	tag              atomic.Uint64
	revisionToSelect string
	offScreenRows    []*models.RevisionItem
	streamer         *GraphStreamer
	hasMore          bool
}

func NewRevisionsContext(commandRunner CommandRunner) *RevisionsContext {
	return &RevisionsContext{
		CommandRunner: commandRunner,
		Op:            operations.NewDefault(),
		Revisions:     list.NewCheckableList[*models.RevisionItem](),
		Files:         list.NewCheckableList[*models.RevisionFileItem](),
	}
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
		output, err := m.RunCommandImmediate(jj.Log(revset, config.Current.Limit))
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

func (m *RevisionsContext) CursorUp() tea.Cmd {
	if m.Revisions.Cursor > 0 {
		m.Revisions.Cursor--
	}
	return nil
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

func (m *RevisionsContext) SetOperation(op operations.Operation, continuations ...tea.Cmd) tea.Cmd {
	return tea.Sequence(func() tea.Msg {
		m.Op = op
		return nil
	}, tea.Batch(continuations...))
}

func (m *RevisionsContext) CloseOperation() tea.Cmd {
	m.Op = operations.NewDefault()
	m.Files.SetItems(nil)
	return m.UpdateSelection()
}

func (m *RevisionsContext) JumpToParent(revisions jj.SelectedRevisions) {
	immediate, _ := m.RunCommandImmediate(jj.GetParent(revisions))
	parentIndex := m.SelectRevision(string(immediate))
	if parentIndex != -1 {
		m.Revisions.Cursor = parentIndex
	}
}

// UpdateSelection FIXME: this can be made private
func (m *RevisionsContext) UpdateSelection() tea.Cmd {
	if len(m.Files.Items) > 0 {
		currentRevision := m.Revisions.Current()
		if current := m.Files.Current(); current != nil {
			return m.Parent.SetSelectedItem(SelectedFile{
				ChangeId: currentRevision.Commit.GetChangeId(),
				CommitId: currentRevision.Commit.CommitId,
				File:     current.FileName,
			})
		}
	}
	if current := m.Revisions.Current(); current != nil {
		return m.Parent.SetSelectedItem(SelectedRevision{
			ChangeId: current.Commit.GetChangeId(),
			CommitId: current.Commit.CommitId,
		})
	}
	return nil
}

func (m *RevisionsContext) LoadFiles() tea.Cmd {
	current := m.Revisions.Current()
	output, err := m.RunCommandImmediate(jj.Snapshot())
	if err == nil {
		output, err = m.RunCommandImmediate(jj.Status(current.Commit.GetChangeId()))
		if err == nil {
			return func() tea.Msg {
				summary := string(output)
				items := createListItems(summary)
				m.Files.SetItems(items)
				m.Files.Cursor = 0
				return m.UpdateSelection()
			}
		}
	}
	return func() tea.Msg {
		return common.CommandCompletedMsg{
			Output: string(output),
			Err:    err,
		}
	}
}

func createListItems(content string) []*models.RevisionFileItem {
	items := make([]*models.RevisionFileItem, 0)
	scanner := bufio.NewScanner(strings.NewReader(content))
	var conflicts []bool
	if scanner.Scan() {
		conflictsLine := strings.Split(scanner.Text(), " ")
		for _, c := range conflictsLine {
			conflicts = append(conflicts, c == "true")
		}
	} else {
		return items
	}

	index := 0
	for scanner.Scan() {
		file := strings.TrimSpace(scanner.Text())
		if file == "" {
			continue
		}
		var status models.Status
		switch file[0] {
		case 'A':
			status = models.Added
		case 'D':
			status = models.Deleted
		case 'M':
			status = models.Modified
		case 'R':
			status = models.Renamed
		}
		fileName := file[2:]

		actualFileName := fileName
		if status == models.Renamed && strings.Contains(actualFileName, "{") {
			for strings.Contains(actualFileName, "{") {
				start := strings.Index(actualFileName, "{")
				end := strings.Index(actualFileName, "}")
				if end == -1 {
					break
				}
				replacement := actualFileName[start+1 : end]
				parts := strings.Split(replacement, " => ")
				replacement = parts[1]
				actualFileName = path.Clean(actualFileName[:start] + replacement + actualFileName[end+1:])
			}
		}
		items = append(items, &models.RevisionFileItem{
			Checkable: &models.Checkable{Checked: false},
			Status:    status,
			Name:      fileName,
			FileName:  actualFileName,
			Conflict:  conflicts[index],
		})
		index++
	}

	return items
}

type AppendRowsBatchMsg struct {
	Rows    []*models.RevisionItem
	HasMore bool
}

type UpdateRevisionsMsg struct {
	Rows             []*models.RevisionItem
	SelectedRevision string
}
