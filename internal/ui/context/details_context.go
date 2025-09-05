package context

import (
	"bufio"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/common/models"
)

type DetailsContext struct {
	*baseView
	CommandRunner
	*list.CheckableList[*models.RevisionFileItem]
	Main *MainContext
}

func NewDetailsContext(ctx *MainContext) *DetailsContext {
	return &DetailsContext{
		baseView:      &baseView{Visible: true, Focused: false},
		CommandRunner: ctx.CommandRunner,
		CheckableList: list.NewCheckableList[*models.RevisionFileItem](),
		Main:          ctx,
	}
}

func (m *DetailsContext) Load() tea.Cmd {
	current := m.Main.Revisions.Revisions.Current()
	output, err := m.RunCommandImmediate(jj.Snapshot())
	if err == nil {
		output, err = m.RunCommandImmediate(jj.Status(current.Commit.GetChangeId()))
		if err == nil {
			return func() tea.Msg {
				summary := string(output)
				items := createListItems(summary)
				m.SetItems(items)
				m.Cursor = 0
				return nil
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
