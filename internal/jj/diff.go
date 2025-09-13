package jj

import "github.com/idursun/jjui/internal/models"

type DiffArgs struct {
	Revision models.RevisionItem
	Files    []models.RevisionFileItem
}

func (d DiffArgs) GetArgs() CommandArgs {
	args := []string{"diff", "-r", d.Revision.Commit.GetChangeId(), "--color", "always", "--ignore-working-copy"}
	for _, file := range d.Files {
		args = append(args, EscapeFileName(file.FileName))
	}
	return args
}
