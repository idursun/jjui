package revisions

import (
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
)

type revisionListRenderer struct {
	*list.ListRenderer
	tracer     parser.LaneTracer
	selections map[string]bool
}

func newRevisionListRenderer(l list.IList, size *common.ViewNode) *revisionListRenderer {
	return &revisionListRenderer{
		ListRenderer: list.NewRenderer(l, size),
	}
}
