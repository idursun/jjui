package jj

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	ChangeIdPlaceholder    = "$change_id"
	FilePlaceholder        = "$file"
	OperationIdPlaceholder = "$operation_id"
	RevsetPlaceholder      = "$revset"
)

type CommandArgs []string

func ConfigListAll() CommandArgs {
	return []string{"config", "list", "--color", "never", "--include-defaults"}
}

func Log(revset string, limit int) CommandArgs {
	args := []string{"log", "--color", "always", "--quiet"}
	if revset != "" {
		args = append(args, "-r", revset)
	}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
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

func Edit(changeId string) CommandArgs {
	return []string{"edit", "-r", changeId}
}

func DiffEdit(changeId string) CommandArgs {
	return []string{"diffedit", "-r", changeId}
}

func Split(revision string, files []string) CommandArgs {
	args := []string{"split", "-r", revision}
	var escapedFiles []string
	for _, file := range files {
		escapedFiles = append(escapedFiles, escapeFileName(file))
	}
	args = append(args, escapedFiles...)
	return args
}

func Describe(revision string) CommandArgs {
	return []string{"describe", "-r", revision, "--edit"}
}

func SetDescription(revision string, description string) CommandArgs {
	return []string{"describe", "-r", revision, "-m", description}
}

func GetDescription(revision string) CommandArgs {
	return []string{"log", "-r", revision, "--template", "description", "--no-graph", "--ignore-working-copy", "--color", "never", "--quiet"}
}

func Abandon(revision SelectedRevisions) CommandArgs {
	args := []string{"abandon", "--retain-bookmarks"}
	args = append(args, revision.AsArgs()...)
	return args
}

func Diff(revision string, fileName string, extraArgs ...string) CommandArgs {
	args := []string{"diff", "-r", revision, "--color", "always"}
	if fileName != "" {
		args = append(args, escapeFileName(fileName))
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
		escapedFiles = append(escapedFiles, escapeFileName(file))
	}
	args = append(args, escapedFiles...)
	return args
}

func Undo() CommandArgs {
	return []string{"undo"}
}

func Snapshot() CommandArgs {
	return []string{"debug", "snapshot"}
}

func Status(revision string) CommandArgs {
	return []string{"log", "-r", revision, "--summary", "--no-graph", "--color", "never", "--quiet", "--template", "", "--ignore-working-copy"}
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

func Squash(from SelectedRevisions, destination string, keepEmptied bool, interactive bool) CommandArgs {
	args := []string{"squash"}
	args = append(args, from.AsPrefixedArgs("--from")...)
	args = append(args, "--into", destination)
	if keepEmptied {
		args = append(args, "--keep-emptied")
	}
	if interactive {
		args = append(args, "--interactive")
	}
	return args
}

func BookmarkList(revset string) CommandArgs {
	const template = `separate(";", name, if(remote, remote, "."), tracked, conflict, 'false', normal_target.commit_id().shortest(1)) ++ "\n"`
	return []string{"bookmark", "list", "-a", "-r", revset, "--template", template, "--color", "never"}
}

func BookmarkListMovable(revision string) CommandArgs {
	revsetBefore := fmt.Sprintf("::%s", revision)
	revsetAfter := fmt.Sprintf("%s::", revision)
	revset := fmt.Sprintf("%s | %s", revsetBefore, revsetAfter)
	template := fmt.Sprintf(moveBookmarkTemplate, revsetAfter)
	return []string{"bookmark", "list", "-r", revset, "--template", template, "--color", "never"}
}

func BookmarkListAll() CommandArgs {
	return []string{"bookmark", "list", "-a", "--template", allBookmarkTemplate, "--color", "never"}
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

func Show(revision string, extraArgs ...string) CommandArgs {
	args := []string{"show", "-r", revision, "--color", "always"}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}
	return args
}

func Rebase(from SelectedRevisions, to string, source string, target string) CommandArgs {
	args := []string{"rebase"}
	args = append(args, from.AsPrefixedArgs(source)...)
	args = append(args, target, to)
	return args
}

func RebaseInsert(from SelectedRevisions, insertAfter string, insertBefore string) CommandArgs {
	args := []string{"rebase"}
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
	return []string{"evolog", "-r", revision, "--color", "always", "--quiet"}
}

func Args(args ...string) CommandArgs {
	return args
}

func TemplatedArgs(templatedArgs []string, replacements map[string]string) CommandArgs {
	var args []string
	if fileReplacement, exists := replacements[FilePlaceholder]; exists {
		// Ensure that the file replacement is quoted
		replacements[FilePlaceholder] = escapeFileName(fileReplacement)
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
		escapedFiles = append(escapedFiles, escapeFileName(file))
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
	return []string{"op", "show", operationId, "--color", "always"}
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

func GetFirstChild(revision *Commit) CommandArgs {
	args := []string{"log", "-r"}
	args = append(args, fmt.Sprintf("%s+", revision.CommitId))
	args = append(args, "-n", "1", "--color", "never", "--no-graph", "--quiet", "--ignore-working-copy", "--template", "commit_id.shortest()")
	return args
}

func FilesInRevision(revision *Commit) CommandArgs {
	args := []string{"file", "list", "-r", revision.CommitId,
		"--color", "never", "--no-pager", "--quiet", "--ignore-working-copy",
		"--template", "self.path() ++ \"\n\""}
	return args
}

func GetIdsFromRevset(revset string) CommandArgs {
	return []string{"log", "-r", revset, "--color", "never", "--no-graph", "--quiet", "--ignore-working-copy", "--template", "change_id.shortest() ++ '\n'"}
}

func escapeFileName(fileName string) string {
	// Escape backslashes and quotes in the file name for shell compatibility
	if strings.Contains(fileName, "\\") {
		fileName = strings.ReplaceAll(fileName, "\\", "\\\\")
	}
	if strings.Contains(fileName, "\"") {
		fileName = strings.ReplaceAll(fileName, "\"", "\\\"")
	}
	return fmt.Sprintf("file:\"%s\"", fileName)
}
