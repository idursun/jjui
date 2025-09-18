package jj

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/idursun/jjui/internal/models"
)

type GlobalArguments struct {
	IgnoreWorkingCopy bool
	IgnoreImmutable   bool
	Color             string // "always", "never", "auto"
}

func (g GlobalArguments) GetGlobalArgs() CommandArgs {
	var args []string
	if g.IgnoreWorkingCopy {
		args = append(args, "--ignore-working-copy")
	}
	if g.IgnoreImmutable {
		args = append(args, "--ignore-immutable")
	}
	if g.Color != "" {
		args = append(args, "--color", g.Color)
	}
	return args
}

type ConfigListAllArgs struct{}

func (c ConfigListAllArgs) GetArgs() CommandArgs {
	return []string{"config", "list", "--color", "never", "--include-defaults", "--ignore-working-copy"}
}

type LogArgs struct {
	GlobalArguments
	Revset   string
	Limit    int
	Template string
	NoGraph  bool
}

func (l LogArgs) GetArgs() CommandArgs {
	args := []string{"log", "--quiet"}
	if l.Revset != "" {
		args = append(args, "-r", l.Revset)
	}
	if l.Limit > 0 {
		args = append(args, "--limit", strconv.Itoa(l.Limit))
	}
	if l.NoGraph {
		args = append(args, "--no-graph")
	}
	if l.Template != "" {
		args = append(args, "-T", l.Template)
	}
	args = append(args, l.GlobalArguments.GetGlobalArgs()...)
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
	GlobalArguments
	Revision models.RevisionItem
}

func (e EditArgs) GetArgs() CommandArgs {
	args := []string{"edit", "-r", e.Revision.Commit.GetChangeId()}
	args = append(args, e.GlobalArguments.GetGlobalArgs()...)
	return args
}

type DiffEditArgs struct {
	GlobalArguments
	Revision models.RevisionItem
}

func (d DiffEditArgs) GetArgs() CommandArgs {
	args := []string{"diffedit", "-r", d.Revision.Commit.GetChangeId()}
	args = append(args, d.GlobalArguments.GetGlobalArgs()...)
	return args
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
	GlobalArguments
	Revisions       SelectedRevisions
	RetainBookmarks bool
}

func (a AbandonArgs) GetArgs() CommandArgs {
	args := []string{"abandon"}
	args = append(args, a.Revisions.AsArgs()...)
	if a.RetainBookmarks {
		args = append(args, "--retain-bookmarks")
	}
	args = append(args, a.GlobalArguments.GetGlobalArgs()...)
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
	GlobalArguments
	From   SelectedRevisions
	To     models.RevisionItem
	Target Target
}

func (r RevertArgs) GetArgs() CommandArgs {
	args := []string{"revert"}
	args = append(args, r.From.AsPrefixedArgs("-r")...)
	args = append(args, targetToFlags[r.Target], r.To.Commit.GetChangeId())
	args = append(args, r.GlobalArguments.GetGlobalArgs()...)
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

type EvologArgs struct {
	Revision models.RevisionItem
}

func (e EvologArgs) GetArgs() CommandArgs {
	return []string{"evolog", "-r", e.Revision.Commit.GetChangeId(), "--color", "always", "--quiet", "--ignore-working-copy"}
}

func Args(args IGetArgs) CommandArgs {
	return args.GetArgs()
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

type OpLogArgs struct {
	GlobalArguments
	NoGraph  bool
	Limit    int
	Template string
}

func (o OpLogArgs) GetArgs() CommandArgs {
	args := []string{"op", "log", "--quiet"}
	if o.NoGraph {
		args = append(args, "--no-graph")
	}
	if o.Limit > 0 {
		args = append(args, "--limit", strconv.Itoa(o.Limit))
	}
	if o.Template != "" {
		args = append(args, "--template", o.Template)
	}
	args = append(args, o.GlobalArguments.GetGlobalArgs()...)
	return args
}

type OpShowArgs struct {
	Operation models.OperationLogItem
}

func (o OpShowArgs) GetArgs() CommandArgs {
	return []string{"op", "show", o.Operation.OperationId, "--color", "always", "--ignore-working-copy"}
}

type OpRestoreArgs struct {
	Operation models.OperationLogItem
}

func (o OpRestoreArgs) GetArgs() CommandArgs {
	return []string{"op", "restore", o.Operation.OperationId}
}

func GetParent(revisions SelectedRevisions) LogArgs {
	joined := strings.Join(revisions.GetIds(), "|")
	revset := fmt.Sprintf("heads(::fork_point(%s) & ~present(%s))", joined, joined)
	return LogArgs{
		Revset:   revset,
		Limit:    1,
		Template: "commit_id.shortest()",
		NoGraph:  true,
		GlobalArguments: GlobalArguments{
			IgnoreWorkingCopy: true,
			Color:             "never",
		},
	}
}

func GetParents(revision string) LogArgs {
	return LogArgs{
		Revset:   revision,
		Limit:    0,
		Template: "parents.map(|x| x.commit_id().shortest())",
		NoGraph:  true,
		GlobalArguments: GlobalArguments{
			IgnoreWorkingCopy: true,
			Color:             "never",
		},
	}
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

func GetIdsFromRevset(revset string) LogArgs {
	return LogArgs{
		Revset:   revset,
		Limit:    0,
		Template: "change_id.shortest() ++ '\n'",
		NoGraph:  true,
		GlobalArguments: GlobalArguments{
			IgnoreWorkingCopy: true,
			Color:             "never",
		},
	}
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
