package jj

import (
	"fmt"

	"github.com/idursun/jjui/internal/models"
)

const (
	moveBookmarkTemplate = `separate(";", name, if(remote, "remote", "."), tracked, conflict, normal_target.contained_in("%s"), normal_target.commit_id().shortest(1)) ++ "\n"`
	allBookmarkTemplate  = `separate(";", name, if(remote, remote, "."), tracked, conflict, 'false', normal_target.commit_id().shortest(1)) ++ "\n"`
)

type BookmarkMoveArgs struct {
	Revision       models.RevisionItem
	Bookmark       string
	AllowBackwards bool
}

func (b BookmarkMoveArgs) GetArgs() CommandArgs {
	args := []string{"bookmark", "move", b.Bookmark, "--to", b.Revision.Commit.GetChangeId()}
	if b.AllowBackwards {
		args = append(args, "--allow-backwards")
	}
	return args
}

type BookmarkDeleteArgs struct {
	Bookmark string
}

func (b BookmarkDeleteArgs) GetArgs() CommandArgs {
	return []string{"bookmark", "delete", b.Bookmark}
}

type BookmarkForgetArgs struct {
	Bookmark string
}

func (b BookmarkForgetArgs) GetArgs() CommandArgs {
	return []string{"bookmark", "forget", b.Bookmark}
}

type BookmarkTrackArgs struct {
	Bookmark string
	Remote   string
}

func (b BookmarkTrackArgs) GetArgs() CommandArgs {
	name := fmt.Sprintf("%s@%s", b.Bookmark, b.Remote)
	return []string{"bookmark", "track", name}
}

type BookmarkUntrackArgs struct {
	Bookmark string
	Remote   string
}

func (b BookmarkUntrackArgs) GetArgs() CommandArgs {
	name := fmt.Sprintf("%s@%s", b.Bookmark, b.Remote)
	return []string{"bookmark", "untrack", name}
}

type BookmarkListArgs struct {
	Revision models.RevisionItem
}

func (b BookmarkListArgs) GetArgs() CommandArgs {
	const template = `separate(";", name, if(remote, remote, "."), tracked, conflict, 'false', normal_target.commit_id().shortest(1)) ++ "\n"`
	return []string{"bookmark", "list", "-a", "-r", b.Revision.Commit.GetChangeId(), "--template", template, "--color", "never", "--ignore-working-copy"}
}

type BookmarkListAllArgs struct {
}

func (b BookmarkListAllArgs) GetArgs() CommandArgs {
	return []string{"bookmark", "list", "-a", "--template", allBookmarkTemplate, "--color", "never", "--ignore-working-copy"}
}

type BookmarkSetArgs struct {
	Revision models.RevisionItem
	Bookmark string
}

func (b BookmarkSetArgs) GetArgs() CommandArgs {
	return []string{"bookmark", "set", "-r", b.Revision.Commit.GetChangeId(), b.Bookmark}
}

type BookmarkListMovableArgs struct {
	Revision models.RevisionItem
}

func (b BookmarkListMovableArgs) GetArgs() CommandArgs {
	revsetBefore := fmt.Sprintf("::%s", b.Revision.Commit.GetChangeId())
	revsetAfter := fmt.Sprintf("%s::", b.Revision.Commit.GetChangeId())
	revset := fmt.Sprintf("%s | %s", revsetBefore, revsetAfter)
	template := fmt.Sprintf(moveBookmarkTemplate, revsetAfter)
	return []string{"bookmark", "list", "-r", revset, "--template", template, "--color", "never", "--ignore-working-copy"}

}
