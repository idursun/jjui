package jj

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/idursun/jjui/internal/models"
)

func ConfigListAll() CommandArgs {
	return []string{"config", "list", "--color", "never", "--include-defaults", "--ignore-working-copy"}
}

type LogArgs struct {
	Revset   string
	Limit    int
	Template string
}

func (l LogArgs) GetArgs() CommandArgs {
	args := []string{"log", "--color", "always", "--quiet"}
	if l.Revset != "" {
		args = append(args, "-r", l.Revset)
	}
	if l.Limit > 0 {
		args = append(args, "--limit", strconv.Itoa(l.Limit))
	}
	if l.Template != "" {
		args = append(args, "-T", l.Template)

	}
	return args
}

type NewArgs struct {
	Revisions SelectedRevisions
}

func (n NewArgs) GetArgs() CommandArgs {
	args := []string{"new"}
	args = append(args, n.Revisions.AsArgs()...)
	return args
}

type CommitArgs struct{}

func (c CommitArgs) GetArgs() CommandArgs {
	return []string{"commit"}
}

type EditArgs struct {
	Revision        models.RevisionItem
	IgnoreImmutable bool
}

func (e EditArgs) GetArgs() CommandArgs {
	args := []string{"edit", "-r", e.Revision.Commit.GetChangeId()}
	if e.IgnoreImmutable {
		args = append(args, "--ignore-immutable")
	}
	return args
}

type DiffEditArgs struct {
	Revision models.RevisionItem
}

func (d DiffEditArgs) GetArgs() CommandArgs {
	return []string{"diffedit", "-r", d.Revision.Commit.GetChangeId()}
}

type SplitArgs struct {
	Revision models.RevisionItem
	Files    []models.RevisionFileItem
}

func (s SplitArgs) GetArgs() CommandArgs {
	args := []string{"split", "-r", s.Revision.Commit.GetChangeId()}
	var escapedFiles []string
	for _, file := range s.Files {
		escapedFiles = append(escapedFiles, EscapeFileName(file.FileName))
	}
	args = append(args, escapedFiles...)
	return args
}

type AbandonArgs struct {
	Revisions       SelectedRevisions
	IgnoreImmutable bool
	RetainBookmarks bool
}

func (a AbandonArgs) GetArgs() CommandArgs {
	args := []string{"abandon"}
	if a.RetainBookmarks {
		args = append(args, "--retain-bookmarks")
	}
	args = append(args, a.Revisions.AsArgs()...)
	if a.IgnoreImmutable {
		args = append(args, "--ignore-immutable")
	}
	return args
}

type RestoreArgs struct {
	Revision models.RevisionItem
	Files    []models.RevisionFileItem
}

func (r RestoreArgs) GetArgs() CommandArgs {
	args := []string{"restore", "-c", r.Revision.Commit.GetChangeId()}
	var escapedFiles []string
	for _, file := range r.Files {
		escapedFiles = append(escapedFiles, EscapeFileName(file.FileName))
	}
	args = append(args, escapedFiles...)
	return args
}

type RestoreEvologArgs struct {
	From               models.RevisionItem
	Into               models.RevisionItem
	Files              []models.RevisionFileItem
	RestoreDescendants bool
}

func (r RestoreEvologArgs) GetArgs() CommandArgs {
	args := []string{"restore", "--from", r.From.Commit.CommitId, "--into", r.Into.Commit.GetChangeId()}
	if r.RestoreDescendants {
		args = append(args, "--restore-descendants")
	}
	return args
}

type UndoArgs struct {
	Steps int
}

func (u UndoArgs) GetArgs() CommandArgs {
	return []string{"undo"}
}

type SnapshotArgs struct{}

func (s SnapshotArgs) GetArgs() CommandArgs {
	return []string{"debug", "snapshot"}
}

type StatusArgs struct {
	Revision models.RevisionItem
}

func (s StatusArgs) GetArgs() CommandArgs {
	template := `separate(";", diff.files().map(|x| x.target().conflict())) ++ "\n"`
	return []string{"log", "-r", s.Revision.Commit.GetChangeId(), "--summary", "--no-graph", "--color", "never", "--quiet", "--template", template, "--ignore-working-copy"}
}

type RevertArgs struct {
	From            SelectedRevisions
	To              models.RevisionItem
	Target          Target
	IgnoreImmutable bool
}

func (r RevertArgs) GetArgs() CommandArgs {
	args := []string{"revert"}
	args = append(args, r.From.AsPrefixedArgs("-r")...)
	args = append(args, targetToFlags[r.Target], r.To.Commit.GetChangeId())
	return args
}

type RevertInsertArgs struct {
	From         SelectedRevisions
	InsertAfter  models.RevisionItem
	InsertBefore models.RevisionItem
}

func (r RevertInsertArgs) GetArgs() CommandArgs {
	args := []string{"revert"}
	args = append(args, r.From.AsArgs()...)
	args = append(args, "--insert-before", r.InsertBefore.Commit.GetChangeId())
	args = append(args, "--insert-after", r.InsertAfter.Commit.GetChangeId())
	return args
}

type DuplicateArgs struct {
	From   SelectedRevisions
	Target Target
	To     models.RevisionItem
}

func (d DuplicateArgs) GetArgs() CommandArgs {
	args := []string{"duplicate"}
	args = append(args, d.From.AsPrefixedArgs("-r")...)
	args = append(args, targetToFlags[d.Target], d.To.Commit.GetChangeId())
	return args
}

func Evolog(revision string) CommandArgs {
	return []string{"evolog", "-r", revision, "--color", "always", "--quiet", "--ignore-working-copy"}
}

func Args(args IGetArgs) CommandArgs {
	return args.GetArgs()
}

func TemplatedArgs(templatedArgs []string, replacements map[string]string) CommandArgs {
	var args []string
	if fileReplacement, exists := replacements[FilePlaceholder]; exists {
		// Ensure that the file replacement is quoted
		replacements[FilePlaceholder] = EscapeFileName(fileReplacement)
	}
	for _, arg := range templatedArgs {
		for k, v := range replacements {
			arg = strings.ReplaceAll(arg, k, v)
		}
		args = append(args, arg)
	}
	return args
}

type AbsorbArgs struct {
	From  models.RevisionItem
	Into  models.RevisionItem
	Files []*models.RevisionFileItem
}

func (a AbsorbArgs) GetArgs() CommandArgs {
	args := []string{"absorb", "--from", a.From.Commit.GetChangeId(), "--color", "never"}
	for _, file := range a.Files {
		args = append(args, EscapeFileName(file.FileName))
	}
	return args
}

func OpLogId(snapshot bool) CommandArgs {
	args := []string{"op", "log", "--color", "never", "--quiet", "--no-graph", "--limit", "1", "--template", "id"}
	if !snapshot {
		args = append(args, "--ignore-working-copy")
	}
	return args
}

func OpLog(limit int) CommandArgs {
	args := []string{"op", "log", "--color", "always", "--quiet", "--ignore-working-copy"}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	return args
}

func OpShow(operationId string) CommandArgs {
	return []string{"op", "show", operationId, "--color", "always", "--ignore-working-copy"}
}

func OpRestore(operationId string) CommandArgs {
	return []string{"op", "restore", operationId}
}

func GetParent(revisions SelectedRevisions) CommandArgs {
	args := []string{"log", "-r"}
	joined := strings.Join(revisions.GetIds(), "|")
	args = append(args, fmt.Sprintf("heads(::fork_point(%s) & ~present(%s))", joined, joined))
	args = append(args, "-n", "1", "--color", "never", "--no-graph", "--quiet", "--ignore-working-copy", "--template", "commit_id.shortest()")
	return args
}

func GetParents(revision string) CommandArgs {
	args := []string{"log", "-r", revision}
	args = append(args, "--color", "never", "--no-graph", "--quiet", "--ignore-working-copy", "--template", "parents.map(|x| x.commit_id().shortest())")
	return args
}

func GetFirstChild(revision *models.Commit) CommandArgs {
	args := []string{"log", "-r"}
	args = append(args, fmt.Sprintf("%s+", revision.CommitId))
	args = append(args, "-n", "1", "--color", "never", "--no-graph", "--quiet", "--ignore-working-copy", "--template", "commit_id.shortest()")
	return args
}

func FilesInRevision(revision *models.Commit) CommandArgs {
	args := []string{"file", "list", "-r", revision.CommitId,
		"--color", "never", "--no-pager", "--quiet", "--ignore-working-copy",
		"--template", "self.path() ++ \"\n\""}
	return args
}

func GetIdsFromRevset(revset string) CommandArgs {
	return []string{"log", "-r", revset, "--color", "never", "--no-graph", "--quiet", "--ignore-working-copy", "--template", "change_id.shortest() ++ '\n'"}
}

func EscapeFileName(fileName string) string {
	// Escape backslashes and quotes in the file name for shell compatibility
	if strings.Contains(fileName, "\\") {
		fileName = strings.ReplaceAll(fileName, "\\", "\\\\")
	}
	if strings.Contains(fileName, "\"") {
		fileName = strings.ReplaceAll(fileName, "\"", "\\\"")
	}
	return fmt.Sprintf("file:\"%s\"", fileName)
}
