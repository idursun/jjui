package jj

import (
	"fmt"
	"strconv"
)

type CommandArgs []string

func ConfigGet(key string) CommandArgs {
	return []string{"config", "get", key}
}

func Log(revset string) CommandArgs {
	args := []string{"log", "--color", "always", "--quiet"}
	if revset != "" {
		args = append(args, "-r", revset)
	}
	return args
}

func New(revisions ...string) CommandArgs {
	args := []string{"new"}
	for _, revision := range revisions {
		args = append(args, "-r", revision)
	}
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
	args = append(args, files...)
	return args
}

func Describe(revision string) CommandArgs {
	return []string{"describe", "-r", revision, "--edit"}
}

func Abandon(revision ...string) CommandArgs {
	args := []string{"abandon"}
	for _, rev := range revision {
		args = append(args, "-r", rev)
	}
	return args
}

func Diff(revision string, fileName string, extraArgs ...string) CommandArgs {
	args := []string{"diff", "-r", revision, "--color", "always"}
	if fileName != "" {
		args = append(args, fileName)
	}
	if extraArgs != nil {
		args = append(args, extraArgs...)
	}
	return args
}

func Restore(revision string, files []string) CommandArgs {
	args := []string{"restore", "-c", revision}
	args = append(args, files...)
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

func Squash(from string, destination string) CommandArgs {
	return []string{"squash", "--from", from, "--into", destination}
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

func Rebase(from string, to string, source string, target string) CommandArgs {
	return []string{"rebase", source, from, target, to}
}

func RebaseInsert(from string, insertAfter string, insertBefore string) CommandArgs {
	return []string{"rebase", "--revisions", from, "--insert-before", insertBefore, "--insert-after", insertAfter}
}

func Evolog(revision string) CommandArgs {
	return []string{"evolog", "-r", revision, "--color", "always", "--quiet"}
}

func Args(args ...string) CommandArgs {
	return args
}

func Absorb(changeId string) CommandArgs {
	return []string{"absorb", "--from", changeId}
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

func GetParent(revision string) CommandArgs {
	return []string{"log", "-r", fmt.Sprintf("%s-", revision), "--color", "never", "--no-graph", "--quiet", "--ignore-working-copy", "--template", "commit_id.shortest()"}
}
