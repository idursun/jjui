package graph

import "github.com/idursun/jjui/internal/jj"

type RowRenderer interface {
	RenderBefore(commit *jj.Commit) string
	RenderAfter(commit *jj.Commit) string
	RenderBeforeChangeId() string
	RenderBeforeCommitId() string
}
