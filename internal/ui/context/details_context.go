package context

import (
	"bufio"
	"path"
	"strings"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/context/models"
	"github.com/idursun/jjui/internal/ui/helpers"
)

type DetailsContext struct {
	CommandRunner
	*helpers.CheckableList[*models.RevisionFile]
}

func NewDetailsContext(commandRunner CommandRunner) *DetailsContext {
	return &DetailsContext{
		CommandRunner: commandRunner,
		CheckableList: helpers.NewCheckableList[*models.RevisionFile](),
	}
}

func (d *DetailsContext) Load(item *models.RevisionItem) {
	revision := item.Commit.GetChangeId()
	output, err := d.RunCommandImmediate(jj.Snapshot())
	if err != nil {
		panic(err)
	}

	output, err = d.RunCommandImmediate(jj.Status(revision))
	if err != nil {
		panic(err)
	}

	content := string(output)
	d.Items = make([]*models.RevisionFile, 0)
	scanner := bufio.NewScanner(strings.NewReader(content))
	var conflicts []bool
	if scanner.Scan() {
		conflictsLine := strings.Split(scanner.Text(), " ")
		for _, c := range conflictsLine {
			conflicts = append(conflicts, c == "true")
		}
	} else {
		return
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
		d.Items = append(d.Items, &models.RevisionFile{
			Status:   status,
			Name:     fileName,
			FileName: actualFileName,
			Conflict: conflicts[index],
		})
		index++
	}
	d.SetCursor(0)
}
