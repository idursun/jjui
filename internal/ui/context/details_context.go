package context

import (
	"bufio"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	models2 "github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
)

type DetailsContext struct {
	CommandRunner
	*list.CheckableList[*models2.RevisionFileItem]
	Main *MainContext
}

func NewDetailsContext(ctx *MainContext) *DetailsContext {
	return &DetailsContext{
		CommandRunner: ctx.CommandRunner,
		CheckableList: list.NewCheckableList[*models2.RevisionFileItem](),
		Main:          ctx,
	}
}

func (m *DetailsContext) Load() tea.Cmd {
	current := m.Main.Revisions.Current()
	output, err := m.RunCommandImmediate(jj.Snapshot())
	if err == nil {
		output, err = m.RunCommandImmediate(jj.Status(current.Commit.GetChangeId()))
		if err == nil {
			return func() tea.Msg {
				summary := string(output)
				items := createListItems(summary)
				m.SetItems(items)
				m.Cursor = 0
				return ""
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

func createListItems(content string) []*models2.RevisionFileItem {
	items := make([]*models2.RevisionFileItem, 0)
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
		var status models2.Status
		switch file[0] {
		case 'A':
			status = models2.Added
		case 'D':
			status = models2.Deleted
		case 'M':
			status = models2.Modified
		case 'R':
			status = models2.Renamed
		}
		fileName := file[2:]

		actualFileName := fileName
		if status == models2.Renamed && strings.Contains(actualFileName, "{") {
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
		items = append(items, &models2.RevisionFileItem{
			Checkable: &models2.Checkable{Checked: false},
			Status:    status,
			Name:      fileName,
			FileName:  actualFileName,
			Conflict:  conflicts[index],
		})
		index++
	}

	return items
}
