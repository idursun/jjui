package jj

import (
	"strings"

	"github.com/idursun/jjui/internal/models"
)

type Source int

const (
	SourceRevision Source = iota
	SourceBranch
	SourceDescendants
)

type Target int

const (
	TargetDestination Target = iota
	TargetAfter
	TargetBefore
	TargetInsert
)

var (
	sourceToFlags = map[Source]string{
		SourceBranch:      "--branch",
		SourceRevision:    "--revisions",
		SourceDescendants: "--source",
	}
	targetToFlags = map[Target]string{
		TargetAfter:       "--insert-after",
		TargetBefore:      "--insert-before",
		TargetDestination: "--destination",
	}
)

type RebaseCommandArgs struct {
	From            SelectedRevisions
	To              models.RevisionItem
	Source          Source
	Target          Target
	IgnoreImmutable bool
}

func (r RebaseCommandArgs) GetArgs() CommandArgs {
	var args CommandArgs
	args = append(args, "rebase")
	args = append(args, r.From.AsPrefixedArgs(sourceToFlags[r.Source])...)
	args = append(args, targetToFlags[r.Target], r.To.Commit.GetChangeId())
	if r.IgnoreImmutable {
		args = append(args, "--ignore-immutable")
	}
	return args
}

type RebaseInsertArgs struct {
	From            SelectedRevisions
	InsertAfter     models.RevisionItem
	InsertBefore    models.RevisionItem
	IgnoreImmutable bool
}

func (r RebaseInsertArgs) GetArgs() CommandArgs {
	args := []string{"rebase"}
	args = append(args, r.From.AsArgs()...)
	args = append(args, "--insert-before", r.InsertBefore.Commit.GetChangeId())
	args = append(args, "--insert-after", r.InsertAfter.Commit.GetChangeId())
	if r.IgnoreImmutable {
		args = append(args, "--ignore-immutable")
	}
	return args
}

type RebaseSetParentsArgs struct {
	To              models.RevisionItem
	ParentsToAdd    []models.RevisionItem
	ParentsToRemove []models.RevisionItem
}

func (r RebaseSetParentsArgs) GetArgs() CommandArgs {
	var b strings.Builder
	b.WriteString("parents(")
	b.WriteString(r.To.Commit.GetChangeId())
	b.WriteString(") ")
	for _, remove := range r.ParentsToRemove {
		b.WriteString(" ~ ")
		b.WriteString(remove.Commit.GetChangeId())
	}
	for _, add := range r.ParentsToAdd {
		b.WriteString(" | ")
		b.WriteString(add.Commit.GetChangeId())
	}
	args := []string{"rebase", "-s", r.To.Commit.GetChangeId(), "-d", b.String()}
	return args
}
