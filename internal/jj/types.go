package jj

type IGetArgs interface {
	GetArgs() CommandArgs
}

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

func Convert[T any](items []*T) []T {
	var result []T
	for _, item := range items {
		result = append(result, *item)
	}
	return result
}
