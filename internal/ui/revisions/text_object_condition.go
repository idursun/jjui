package revisions

import (
	"strconv"
	"strings"

	"github.com/idursun/jjui/internal/jj"
)

// revsetCondition is the revset that matches what obj on commit stands for,
// or why there is none.
func revsetCondition(obj *textObject, commit *jj.Commit) (string, string) {
	switch obj.Kind {
	case textObjectNode, textObjectChangeId:
		return "::" + commit.GetChangeId(), ""
	case textObjectCommitId:
		return "::" + commit.CommitId, ""
	case textObjectBookmark:
		kind, value := previewObject(obj)
		if kind != textObjectBookmark {
			return "", "Conflicted bookmarks cannot be added to the revset"
		}
		return jj.BookmarkForkPoint(value) + ".." + value, ""
	case textObjectTag:
		return "::" + revsetSymbol(obj.Value), ""
	case textObjectWorkspace:
		return "::" + revsetSymbol(obj.Value) + "@", ""
	case textObjectAuthor:
		return "author(" + jj.ExactStringPattern(obj.Value) + ")", ""
	case textObjectDescription:
		line, _, _ := strings.Cut(obj.Value, "\n")
		return "description(substring:" + strconv.Quote(line) + ")", ""
	case textObjectConflict:
		return "conflicts()", ""
	case textObjectDivergent:
		return "divergent()", ""
	}
	return "", "This object cannot be added to the revset"
}
