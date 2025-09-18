package jj

import (
	"strconv"

	"github.com/idursun/jjui/internal/models"
)

type WhitespaceOption int

const (
	WhitespaceDefault WhitespaceOption = iota
	WhitespaceIgnoreAll
	WhitespaceIgnoreChange
)

type OutputMode int

const (
	OutputModeNone OutputMode = iota
	OutputModeSummary
	OutputModeStat
	OutputModeTypes
	OutputModeNameOnly
)

type DiffDisplayType int

const (
	DiffDisplayNone DiffDisplayType = iota
	DiffDisplayGit
	DiffDisplayColorWords
)

type DiffFormattingOptions struct {
	WhiteSpace WhitespaceOption
	Output     OutputMode
	Display    DiffDisplayType
	Template   string
	Context    int
}

func (d DiffFormattingOptions) GetArgs() CommandArgs {
	var args []string
	switch d.WhiteSpace {
	case WhitespaceIgnoreAll:
		args = append(args, "--ignore-all-space")
	case WhitespaceIgnoreChange:
		args = append(args, "--ignore-space-change")
	}

	if d.Template != "" {
		args = append(args, "-T", d.Template)
	} else {
		switch d.Output {
		case OutputModeNone:
		case OutputModeSummary:
			args = append(args, "--summary")
		case OutputModeStat:
			args = append(args, "--stat")
		case OutputModeTypes:
			args = append(args, "--type")
		case OutputModeNameOnly:
			args = append(args, "--name-only")
		}
	}

	switch d.Display {
	case DiffDisplayGit:
		args = append(args, "--git")
	case DiffDisplayColorWords:
		args = append(args, "--color-words")
	}
	if d.Context > 0 {
		args = append(args, "--context", strconv.Itoa(d.Context))
	}
	return args
}

type IDiffSource interface {
	IGetArgs
	isDiffSource()
}

type diffRevisionsSource struct {
	revset IRevset
}

func (d diffRevisionsSource) isDiffSource() {}

func (d diffRevisionsSource) GetArgs() CommandArgs {
	return builder("-r", d.revset.GetArgs()...)
}

func NewDiffRevisionsSource(revset IRevset) IDiffSource {
	return diffRevisionsSource{
		revset: revset,
	}
}

type diffRangeSource struct {
	from IRevset
	to   IRevset
}

func NewDiffRangeArgs(from IRevset, to IRevset) IDiffSource {
	return diffRangeSource{
		from: from,
		to:   to,
	}
}

func (d diffRangeSource) isDiffSource() {}

func (d diffRangeSource) GetArgs() CommandArgs {
	var args []string
	args = append(args, builder("--from", d.from.GetArgs()...)...)
	args = append(args, builder("--to", d.to.GetArgs()...)...)
	return args
}

type DiffCommandArgs struct {
	// Source can be NewDiffRevisionsSource, NewDiffRangeArgs
	Source     IDiffSource
	Files      []models.RevisionFileItem
	Formatting DiffFormattingOptions
}

func (d DiffCommandArgs) GetArgs() CommandArgs {
	args := []string{"diff", "--color", "always", "--ignore-working-copy"}
	if d.Source != nil {
		args = append(args, d.Source.GetArgs()...)
	}
	for _, file := range d.Files {
		args = append(args, EscapeFileName(file.FileName))
	}
	args = append(args, d.Formatting.GetArgs()...)
	return args
}

func builder(prefix string, arguments ...string) []string {
	var ret []string
	for _, arg := range arguments {
		ret = append(ret, prefix, arg)
	}
	return ret
}
