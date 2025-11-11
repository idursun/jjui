package jj

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/idursun/jjui/internal/config"
)

const (
	ChangeIdPlaceholder    = "$change_id"
	CommitIdPlaceholder    = "$commit_id"
	FilePlaceholder        = "$file"
	OperationIdPlaceholder = "$operation_id"
	RevsetPlaceholder      = "$revset"

	// user checked file names, separated by `\t` tab.
	// tab is a lot less common than spaces on filenames,
	// and is also part of shell's IFS separator.
	// this allows programs like `ls -l ${checked_files[@]}`
	CheckedFilesPlaceholder = "$checked_files"

	// user checked commit ids, separated by `|`.
	// the reason is user can use checked commits as revsets
	// given to jj commands.
	CheckedCommitIdsPlaceholder = "$checked_commit_ids"
)

type CommandArgs []string

func ConfigListAll() CommandArgs {
	return []string{"config", "list", "--color", "never", "--include-defaults", "--ignore-working-copy"}
}

func Log(revset string, limit int) CommandArgs {
	args := []string{"log", "--color", "always", "--quiet"}
	if revset != "" {
		args = append(args, "-r", revset)
	}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	if config.Current.Revisions.Template != "" {
		prefix := "change_id.shortest(), commit_id.shortest(), divergent"
		// prefix := "format_short_id(change_id), format_short_id(commit_id), divergent"
		template := fmt.Sprintf("separate('',  %s, %s)", prefix, config.Current.Revisions.Template)
		args = append(args, "-T", template)
	}
	return args
}

func New(revisions SelectedRevisions) CommandArgs {
	args := []string{"new"}
	args = append(args, revisions.AsArgs()...)
	return args
}

func CommitWorkingCopy() CommandArgs {
	return []string{"commit"}
}

func Edit(changeId string, ignoreImmutable bool) CommandArgs {
	args := []string{"edit", "-r", changeId}
	if ignoreImmutable {
		args = append(args, "--ignore-immutable")
	}
	return args
}

func DiffEdit(changeId string) CommandArgs {
	return []string{"diffedit", "-r", changeId}
}

func Split(revision string, files []string, parallel bool) CommandArgs {
	args := []string{"split", "-r", revision}
	if parallel {
		args = append(args, "--parallel")
	}
	var escapedFiles []string
	for _, file := range files {
		escapedFiles = append(escapedFiles, EscapeFileName(file))
	}
	args = append(args, escapedFiles...)
	return args
}

func SquashFiles(from string, into string, files []string) CommandArgs {
	args := []string{"squash", "--from", from, "--into", into, "--use-destination-message"}
	var escapedFiles []string
	for _, file := range files {
		escapedFiles = append(escapedFiles, EscapeFileName(file))
	}
	args = append(args, escapedFiles...)
	return args
}

func Describe(revisions SelectedRevisions) CommandArgs {
	args := []string{"describe", "--edit"}
	args = append(args, revisions.AsArgs()...)
	return args
}

func SetDescription(revision string, description string) CommandArgs {
	return []string{"describe", "-r", revision, "-m", description}
}

func GetDescription(revision string) CommandArgs {
	return []string{"log", "-r", revision, "--template", "description", "--no-graph", "--ignore-working-copy", "--color", "never", "--quiet"}
}

func Abandon(revision SelectedRevisions, ignoreImmutable bool) CommandArgs {
	args := []string{"abandon", "--retain-bookmarks"}
	args = append(args, revision.AsArgs()...)
	if ignoreImmutable {
		args = append(args, "--ignore-immutable")
	}
	return args
}

func Diff(revision string, fileName string, extraArgs ...string) CommandArgs {
	args := []string{"diff", "-r", revision, "--color", "always", "--ignore-working-copy"}
	if fileName != "" {
		args = append(args, EscapeFileName(fileName))
	}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}
	return args
}

func Restore(revision string, files []string) CommandArgs {
	args := []string{"restore", "-c", revision}
	var escapedFiles []string
	for _, file := range files {
		escapedFiles = append(escapedFiles, EscapeFileName(file))
	}
	args = append(args, escapedFiles...)
	return args
}

func RestoreInteractive(revision string, file string) CommandArgs {
	args := []string{"restore", "-c", revision, "--interactive"}
	if file != "" {
		args = append(args, EscapeFileName(file))
	}
	return args
}

func RestoreEvolog(from string, into string) CommandArgs {
	args := []string{"restore", "--from", from, "--into", into, "--restore-descendants"}
	return args
}

func Undo() CommandArgs {
	return []string{"undo"}
}

func Redo() CommandArgs {
	return []string{"redo"}
}

func Snapshot() CommandArgs {
	return []string{"debug", "snapshot"}
}

func Status(revision string) CommandArgs {
	template := `separate(";", diff.files().map(|x| x.target().conflict())) ++ "\n"`
	return []string{"log", "-r", revision, "--summary", "--no-graph", "--color", "never", "--quiet", "--template", template, "--ignore-working-copy"}
}

func BookmarkSet(revision string, name string) CommandArgs {
	return []string{"bookmark", "set", "-r", revision, name}
}

func BookmarkMove(revision string, bookmark string, extraFlags ...string) CommandArgs {
	args := []string{"bookmark", "move", bookmark, "--to", revision}
	if extraFlags != nil {
		args = append(args, extraFlags...)
	}
	return args
}

func BookmarkDelete(name string) CommandArgs {
	return []string{"bookmark", "delete", name}
}

func BookmarkForget(name string) CommandArgs {
	return []string{"bookmark", "forget", name}
}

func BookmarkTrack(name string) CommandArgs {
	return []string{"bookmark", "track", name}
}

func BookmarkUntrack(name string) CommandArgs {
	return []string{"bookmark", "untrack", name}
}

func Squash(from SelectedRevisions, destination string, files []string, keepEmptied bool, interactive bool, ignoreImmutable bool) CommandArgs {
	args := []string{"squash"}
	args = append(args, from.AsPrefixedArgs("--from")...)
	args = append(args, "--into", destination)
	if keepEmptied {
		args = append(args, "--keep-emptied")
	}
	if interactive {
		args = append(args, "--interactive")
	}
	if ignoreImmutable {
		args = append(args, "--ignore-immutable")
	}
	if len(files) > 0 {
		var escapedFiles []string
		for _, file := range files {
			escapedFiles = append(escapedFiles, EscapeFileName(file))
		}
		args = append(args, escapedFiles...)
	}
	return args
}

func BookmarkList(revset string) CommandArgs {
	const template = `separate(";", name, if(remote, remote, "."), tracked, conflict, 'false', normal_target.commit_id().shortest(1)) ++ "\n"`
	return []string{"bookmark", "list", "-a", "-r", revset, "--template", template, "--color", "never", "--ignore-working-copy"}
}

func BookmarkListMovable(revision string) CommandArgs {
	revsetBefore := fmt.Sprintf("::%s", revision)
	revsetAfter := fmt.Sprintf("%s::", revision)
	revset := fmt.Sprintf("%s | %s", revsetBefore, revsetAfter)
	template := fmt.Sprintf(moveBookmarkTemplate, revsetAfter)
	return []string{"bookmark", "list", "-r", revset, "--template", template, "--color", "never", "--ignore-working-copy"}
}

func BookmarkListAll() CommandArgs {
	return []string{"bookmark", "list", "-a", "--template", allBookmarkTemplate, "--color", "never", "--ignore-working-copy"}
}

func GitFetch(flags ...string) CommandArgs {
	args := []string{"git", "fetch"}
	if flags != nil {
		args = append(args, flags...)
	}
	return args
}

func GitPush(flags ...string) CommandArgs {
	args := []string{"git", "push"}
	if flags != nil {
		args = append(args, flags...)
	}
	return args
}

func GitRemoteList() CommandArgs {
	return []string{"git", "remote", "list"}
}

func Show(revision string, extraArgs ...string) CommandArgs {
	args := []string{"show", "-r", revision, "--color", "always", "--ignore-working-copy"}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}
	return args
}

func Rebase(from SelectedRevisions, to string, source string, target string, skipEmptied bool, ignoreImmutable bool) CommandArgs {
	args := []string{"rebase"}
	args = append(args, from.AsPrefixedArgs(source)...)
	args = append(args, target, to)
	if ignoreImmutable {
		args = append(args, "--ignore-immutable")
	}
	if skipEmptied {
		args = append(args, "--skip-emptied")
	}
	return args
}

func RebaseInsert(from SelectedRevisions, insertAfter string, insertBefore string, skipEmptied bool, ignoreImmutable bool) CommandArgs {
	args := []string{"rebase"}
	args = append(args, from.AsArgs()...)
	args = append(args, "--insert-before", insertBefore)
	args = append(args, "--insert-after", insertAfter)
	if ignoreImmutable {
		args = append(args, "--ignore-immutable")
	}
	if skipEmptied {
		args = append(args, "--skip-emptied")
	}
	return args
}

func SetParents(to string, parentsToAdd []string, parentsToRemove []string) CommandArgs {
	var b strings.Builder
	b.WriteString("parents(")
	b.WriteString(to)
	b.WriteString(")")
	for _, remove := range parentsToRemove {
		b.WriteString(" ~ ")
		b.WriteString(remove)
	}
	args := []string{"rebase", "-s", to, "-d", b.String()}
	for _, add := range parentsToAdd {
		args = append(args, "-d", add)
	}
	return args
}

func Revert(from SelectedRevisions, to string, source string, target string) CommandArgs {
	args := []string{"revert"}
	args = append(args, from.AsPrefixedArgs(source)...)
	args = append(args, target, to)
	return args
}

func RevertInsert(from SelectedRevisions, insertAfter string, insertBefore string) CommandArgs {
	args := []string{"revert"}
	args = append(args, from.AsArgs()...)
	args = append(args, "--insert-before", insertBefore)
	args = append(args, "--insert-after", insertAfter)
	return args
}

func Duplicate(from SelectedRevisions, to string, target string) CommandArgs {
	args := []string{"duplicate"}
	args = append(args, from.AsPrefixedArgs("-r")...)
	args = append(args, target, to)
	return args
}

func Evolog(revision string) CommandArgs {
	return []string{"evolog", "-r", revision, "--color", "always", "--quiet", "--ignore-working-copy"}
}

func Args(args ...string) CommandArgs {
	return args
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

func Absorb(changeId string, files ...string) CommandArgs {
	args := []string{"absorb", "--from", changeId, "--color", "never"}
	var escapedFiles []string
	for _, file := range files {
		escapedFiles = append(escapedFiles, EscapeFileName(file))
	}
	args = append(args, escapedFiles...)
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

func GetFirstChild(revision *Commit) CommandArgs {
	args := []string{"log", "-r"}
	args = append(args, fmt.Sprintf("%s+", revision.CommitId))
	args = append(args, "-n", "1", "--color", "never", "--no-graph", "--quiet", "--ignore-working-copy", "--template", "commit_id.shortest()")
	return args
}

func FilesInRevision(revision *Commit) CommandArgs {
	args := []string{
		"file", "list", "-r", revision.CommitId,
		"--color", "never", "--no-pager", "--quiet", "--ignore-working-copy",
		"--template", "self.path() ++ \"\n\"",
	}
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
