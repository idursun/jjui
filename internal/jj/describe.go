package jj

import "github.com/idursun/jjui/internal/models"

type DescribeArgs struct {
	Revisions SelectedRevisions
}

func (d DescribeArgs) GetArgs() CommandArgs {
	args := []string{"describe", "--edit"}
	args = append(args, d.Revisions.AsArgs()...)
	return args
}

type SetDescriptionArgs struct {
	Revision models.RevisionItem
	Message  string
}

func (s SetDescriptionArgs) GetArgs() CommandArgs {
	return []string{"describe", "-r", s.Revision.Commit.GetChangeId(), "-m", s.Message}
}

type GetDescriptionArgs struct {
	Revision models.RevisionItem
}

func (g GetDescriptionArgs) GetArgs() CommandArgs {
	return []string{"log", "-r", g.Revision.Commit.GetChangeId(), "--template", "description", "--no-graph", "--ignore-working-copy", "--color", "never", "--quiet"}
}
