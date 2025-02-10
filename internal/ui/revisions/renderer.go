package revisions

import (
	"fmt"
	"strings"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"

	tea "github.com/charmbracelet/bubbletea"
)

type SegmentedRenderer struct {
	Palette       common.Palette
	IsHighlighted bool
	Overlay       tea.Model
	op            common.Operation
	After         string
}

func (s SegmentedRenderer) RenderBefore(commit *jj.Commit) string {
	if s.op == common.RebaseRevisionOperation || s.op == common.RebaseBranchOperation {
		if s.IsHighlighted {
			return common.DropStyle.Render("<< here >>")
		}
	}
	return ""
}

func (s SegmentedRenderer) RenderAfter(*jj.Commit) string {
	if s.IsHighlighted && s.Overlay != nil && s.op != common.EditDescriptionOperation && s.op != common.SetBookmarkOperation {
		return s.Overlay.View()
	}
	if s.After != "" {
		return s.After
	}
	return ""
}

func (s SegmentedRenderer) RenderGlyph(connection jj.ConnectionType, commit *jj.Commit) string {
	style := s.Palette.Normal
	squashDropMarker := ""
	if s.IsHighlighted {
		style = s.Palette.Selected
		if s.op == common.SquashOperation {
			squashDropMarker = common.DropStyle.Render(" << into >> ")
		}
	}
	return style.Render(string(connection) + squashDropMarker)
}

func (s SegmentedRenderer) RenderTermination(connection jj.ConnectionType) string {
	return s.Palette.CommitIdRestStyle.Render(string(connection))
}

func (s SegmentedRenderer) RenderChangeId(commit *jj.Commit) string {
	hidden := ""
	if commit.Hidden {
		hidden = s.Palette.Normal.Render(" hidden")
	}

	return fmt.Sprintf("%s%s %s", s.Palette.CommitShortStyle.Render(commit.ChangeIdShort), s.Palette.CommitIdRestStyle.Render(commit.ChangeId[len(commit.ChangeIdShort):]), hidden)
}

func (s SegmentedRenderer) RenderAuthor(commit *jj.Commit) string {
	if commit.IsRoot() {
		return s.Palette.Empty.Render("root()")
	}
	return s.Palette.AuthorStyle.Render(commit.Author)
}

func (s SegmentedRenderer) RenderDate(commit *jj.Commit) string {
	return s.Palette.TimestampStyle.Render(commit.Timestamp)
}

func (s SegmentedRenderer) RenderBookmarks(commit *jj.Commit) string {
	var w strings.Builder
	if s.op == common.SetBookmarkOperation && s.IsHighlighted {
		w.WriteString(s.Overlay.View())
	}
	w.WriteString(s.Palette.BookmarksStyle.Render(strings.Join(commit.Bookmarks, " ")))
	return w.String()
}

func (s SegmentedRenderer) RenderMarkers(commit *jj.Commit) string {
	if commit.Conflict {
		return s.Palette.ConflictStyle.Render("conflict")
	}
	return ""
}

func (s SegmentedRenderer) RenderDescription(commit *jj.Commit) string {
	var w strings.Builder
	if s.op == common.EditDescriptionOperation && s.IsHighlighted {
		w.WriteString(s.Overlay.View())
		return w.String()
	}
	if commit.Empty {
		w.WriteString(s.Palette.Empty.Render("(empty)"))
		w.WriteString(" ")
	}
	if commit.Description == "" {
		if commit.Empty {
			w.WriteString(s.Palette.Empty.Render("(no description set)"))
		} else {
			w.WriteString(s.Palette.NonEmpty.Render("(no description set)"))
		}
	} else {
		w.WriteString(s.Palette.Normal.Render(commit.Description))
	}
	return w.String()
}
