package jj

import "github.com/idursun/jjui/internal/models"

type GitPushArgs struct {
	Change   models.RevisionItem
	Bookmark string
	Remote   string
	AllowNew bool
	All      bool
	Deleted  bool
	Tracked  bool
}

func (g GitPushArgs) GetArgs() CommandArgs {
	args := []string{"git", "push"}
	if g.Bookmark != "" {
		args = append(args, "--bookmark", g.Bookmark)
	}
	if g.Remote != "" {
		args = append(args, "--remote", g.Remote)
	}
	if g.AllowNew {
		args = append(args, "--allow-new")
	}
	if g.All {
		args = append(args, "--all")
	}
	if g.Deleted {
		args = append(args, "--deleted")
	}
	if g.Tracked {
		args = append(args, "--tracked")
	}
	return args
}

func GitPush(args GitPushArgs) CommandArgs {
	return args.GetArgs()
}

type GitFetchArgs struct {
	AllRemotes bool
}

func (g GitFetchArgs) GetArgs() CommandArgs {
	args := []string{"git", "fetch"}
	if g.AllRemotes {
		args = append(args, "--all-remotes")
	}
	return args
}
