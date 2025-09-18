package jj

import "github.com/idursun/jjui/internal/models"

type SquashRevisionArgs struct {
	From        SelectedRevisions
	Into        models.RevisionItem
	Interactive bool
	KeepEmptied bool
}

type SquashFilesArgs struct {
	From        models.RevisionItem
	Into        models.RevisionItem
	Files       []models.RevisionFileItem
	Interactive bool
	KeepEmptied bool
}

func (s SquashFilesArgs) GetArgs() CommandArgs {
	args := []string{"squash", "--from", s.From.Commit.GetChangeId(), "--into", s.Into.Commit.GetChangeId(), "--use-destination-message"}
	if s.KeepEmptied {
		args = append(args, "--keep-emptied")
	}
	if s.Interactive {
		args = append(args, "--interactive")
	}
	var escapedFiles []string
	for _, file := range s.Files {
		escapedFiles = append(escapedFiles, EscapeFileName(file.FileName))
	}
	args = append(args, escapedFiles...)
	return args
}

func (s SquashRevisionArgs) GetArgs() CommandArgs {
	args := []string{"squash"}
	args = append(args, s.From.AsPrefixedArgs("--from")...)
	args = append(args, "--into", s.Into.Commit.GetChangeId())
	if s.KeepEmptied {
		args = append(args, "--keep-emptied")
	}
	if s.Interactive {
		args = append(args, "--interactive")
	}
	return args
}

func Squash(args IGetArgs) CommandArgs {
	return args.GetArgs()
}
