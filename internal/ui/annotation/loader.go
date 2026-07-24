package annotation

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	appContext "github.com/idursun/jjui/internal/ui/context"
)

type revisionLoadedMsg struct {
	ChangeID    string
	RequestID   uint64
	Description string
	Files       []fileItem
	Err         error
}

type revisionTargetsLoadedMsg struct {
	ChangeID  string
	RequestID uint64
	Targets   []revisionTarget
	Err       error
}

type revisionTarget struct {
	ChangeID    string
	Description string
}

type fileLoadedMsg struct {
	ChangeID string
	Path     string
	Content  string
	Err      error
}

type annotationLoader struct {
	context *appContext.MainContext
}

func (l annotationLoader) LoadRevision(changeID string, fullFile bool, requestID uint64) tea.Cmd {
	if l.context == nil || changeID == "" {
		return nil
	}
	return func() tea.Msg {
		descriptionOutput, descriptionErr := l.context.RunCommandImmediate(jj.GetDescription(changeID))
		description := "(description unavailable)"
		if descriptionErr == nil {
			description = descriptionSummary(string(descriptionOutput))
		}
		diffOutput, err := l.context.RunCommandImmediate(jj.AnnotationDiff(changeID))
		if err != nil {
			return revisionLoadedMsg{
				ChangeID:    changeID,
				RequestID:   requestID,
				Description: description,
				Err:         err,
			}
		}
		files := patchFiles(parseGitPatch(string(diffOutput)))
		if fullFile && len(files) > 0 {
			file := &files[0]
			output, fileErr := l.context.RunCommandImmediate(
				jj.FileShow(fileRevision(changeID, file), file.Path),
			)
			file.ContentErr = fileErr
			file.ContentLoaded = fileErr == nil
			if fileErr == nil {
				file.Content = splitFileLines(string(output))
			}
		}
		return revisionLoadedMsg{
			ChangeID:    changeID,
			RequestID:   requestID,
			Description: description,
			Files:       files,
		}
	}
}

func (l annotationLoader) LoadFile(changeID string, file *fileItem) tea.Cmd {
	if l.context == nil || file == nil || file.ContentLoaded || file.ContentErr != nil {
		return nil
	}
	path := file.Path
	revision := fileRevision(changeID, file)
	return func() tea.Msg {
		output, err := l.context.RunCommandImmediate(jj.FileShow(revision, path))
		return fileLoadedMsg{
			ChangeID: changeID,
			Path:     path,
			Content:  string(output),
			Err:      err,
		}
	}
}

func (l annotationLoader) LoadRevisionTargets(
	changeID string,
	direction revisionDirection,
	requestID uint64,
) tea.Cmd {
	if l.context == nil || changeID == "" {
		return nil
	}
	revset := changeID + "-"
	if direction == revisionChild {
		revset = changeID + "+"
	}
	return func() tea.Msg {
		output, err := l.context.RunCommandImmediate(jj.GetRevisionSummariesFromRevset(revset))
		return revisionTargetsLoadedMsg{
			ChangeID:  changeID,
			RequestID: requestID,
			Targets:   parseRevisionTargets(string(output)),
			Err:       err,
		}
	}
}

func parseRevisionTargets(output string) []revisionTarget {
	var targets []revisionTarget
	for line := range strings.SplitSeq(strings.ReplaceAll(output, "\r", ""), "\n") {
		fields := strings.SplitN(line, "\t", 2)
		changeID := strings.TrimSpace(fields[0])
		if changeID == "" {
			continue
		}
		description := ""
		if len(fields) == 2 {
			description = fields[1]
		}
		targets = append(targets, revisionTarget{
			ChangeID:    changeID,
			Description: descriptionSummary(description),
		})
	}
	return targets
}

func descriptionSummary(description string) string {
	for line := range strings.SplitSeq(strings.ReplaceAll(description, "\r", ""), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return "(no description set)"
}

func fileRevision(changeID string, file *fileItem) string {
	if file.Patch != nil && file.Patch.NewPath == "" {
		return changeID + "-"
	}
	return changeID
}
