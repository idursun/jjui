package graph

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/operations"
)

type DefaultRowRenderer struct {
	section             RowSection
	Palette             common.Palette
	HighlightBackground lipgloss.AdaptiveColor
	IsHighlighted       bool
	IsSelected          bool
	IsAffected          bool
	Op                  operations.Operation
}

func (s *DefaultRowRenderer) BeginSection(section RowSection) {
	s.section = section
}

func (s *DefaultRowRenderer) RenderNormal(text string) string {
	normal := s.Palette.Normal
	if s.IsHighlighted {
		normal = normal.Background(s.HighlightBackground)
	}
	return normal.Render(text)
}

func (s *DefaultRowRenderer) RenderConnection(connectionType jj.ConnectionType) string {
	normal := s.Palette.Normal
	if s.IsHighlighted && s.section == RowSectionRevision {
		normal = normal.Background(s.HighlightBackground)
	}
	return normal.Render(string(connectionType))
}

func (s *DefaultRowRenderer) RenderBefore(*jj.Commit) string {
	if s.IsHighlighted && s.Op.RenderPosition() == operations.RenderPositionBefore {
		return s.Op.Render()
	}
	return ""
}

func (s *DefaultRowRenderer) RenderAfter(*jj.Commit) string {
	if s.IsHighlighted && s.Op.RenderPosition() == operations.RenderPositionAfter {
		return s.Op.Render()
	}
	return ""
}

func (s *DefaultRowRenderer) RenderGlyph(connection jj.ConnectionType, commit *jj.Commit) string {
	opMarker := ""
	if s.IsHighlighted {
		if s.Op.RenderPosition() == operations.RenderPositionGlyph {
			opMarker = s.Op.Render()
		}
	}
	selectedMarker := ""
	if s.IsSelected {
		selectedMarker = s.Palette.Added.Render("✓")
	}
	return opMarker + selectedMarker
}

func (s *DefaultRowRenderer) RenderTermination(connection jj.ConnectionType) string {
	return s.Palette.Elided.Render(string(connection))
}

func (s *DefaultRowRenderer) RenderChangeId(commit *jj.Commit) string {
	normalStyle := s.Palette.Normal
	changeIdStyle := s.Palette.ChangeId
	restStyle := s.Palette.Rest
	if s.IsHighlighted {
		normalStyle = normalStyle.Background(s.HighlightBackground)
		changeIdStyle = changeIdStyle.Background(s.HighlightBackground)
		restStyle = restStyle.Background(s.HighlightBackground)
	}
	changeId := changeIdStyle.Render("", commit.ChangeIdShort) + restStyle.Render(commit.ChangeId[len(commit.ChangeIdShort):])
	hidden := ""
	if commit.Hidden {
		hidden = normalStyle.Render(" hidden")
		return lipgloss.JoinHorizontal(0, changeId, hidden)
	}
	return changeId
}

func (s *DefaultRowRenderer) RenderCommitId(commit *jj.Commit) string {
	commitIdStyle := s.Palette.CommitId
	restStyle := s.Palette.Rest
	if s.IsHighlighted {
		commitIdStyle = commitIdStyle.Background(s.HighlightBackground)
		restStyle = restStyle.Background(s.HighlightBackground)
	}
	commitId := commitIdStyle.Render("", commit.CommitIdShort) + restStyle.Render(commit.CommitId[len(commit.ChangeIdShort):])
	if s.IsAffected {
		commitId += restStyle.Render(" (affected by last operation)")
	}
	return commitId
}

func (s *DefaultRowRenderer) RenderAuthor(commit *jj.Commit) string {
	placeholderStyle := s.Palette.EmptyPlaceholder
	authorStyle := s.Palette.Author
	if s.IsHighlighted {
		placeholderStyle = placeholderStyle.Background(s.HighlightBackground)
		authorStyle = authorStyle.Background(s.HighlightBackground)
	}
	if commit.IsRoot() {
		return placeholderStyle.Render(" root()")
	}
	return authorStyle.Render("", commit.Author)
}

func (s *DefaultRowRenderer) RenderDate(commit *jj.Commit) string {
	if commit.IsRoot() {
		return ""
	}
	timestamp := s.Palette.Timestamp
	if s.IsHighlighted {
		timestamp = timestamp.Background(s.HighlightBackground)
	}
	return timestamp.Render("", commit.Timestamp)
}

func (s *DefaultRowRenderer) RenderBookmarks(commit *jj.Commit) string {
	bookmarksStyle := s.Palette.Bookmarks
	if s.IsHighlighted {
		bookmarksStyle = bookmarksStyle.Background(s.HighlightBackground)
	}
	var w strings.Builder
	if s.IsHighlighted && s.Op.RenderPosition() == operations.RenderPositionBookmark {
		w.WriteString(" ")
		w.WriteString(s.Op.Render())
	}
	if len(commit.Bookmarks) > 0 {
		var bookmarks []string
		bookmarks = append(bookmarks, "")
		bookmarks = append(bookmarks, commit.Bookmarks...)
		w.WriteString(bookmarksStyle.Render(bookmarks...))
	}
	return w.String()
}

func (s *DefaultRowRenderer) RenderMarkers(commit *jj.Commit) string {
	conflictStyle := s.Palette.Conflict
	if s.IsHighlighted {
		conflictStyle = conflictStyle.Background(s.HighlightBackground)
	}
	if commit.Conflict {
		return conflictStyle.Render(" conflict")
	}
	return ""
}

func (s *DefaultRowRenderer) RenderDescription(commit *jj.Commit) string {
	emptyPlaceholderStyle := s.Palette.EmptyPlaceholder
	placeholderStyle := s.Palette.Placeholder
	normalStyle := s.Palette.Normal
	if s.IsHighlighted {
		emptyPlaceholderStyle = emptyPlaceholderStyle.Background(s.HighlightBackground)
		placeholderStyle = placeholderStyle.Background(s.HighlightBackground)
		normalStyle = normalStyle.Background(s.HighlightBackground)
	}
	if s.IsHighlighted && s.Op.RenderPosition() == operations.RenderPositionDescription {
		return s.Op.Render()
	}

	if commit.Empty && commit.Description == "" {
		return emptyPlaceholderStyle.Render(" (empty) (no description set)")
	}
	if commit.Empty {
		return lipgloss.JoinHorizontal(0, emptyPlaceholderStyle.Render(" (empty)"), normalStyle.Render("", commit.Description))
	} else if commit.Description == "" {
		return placeholderStyle.Render(" (no description set)")
	}
	return normalStyle.Render("", commit.Description)
}
