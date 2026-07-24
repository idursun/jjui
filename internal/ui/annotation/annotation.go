package annotation

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type lineRange struct {
	Start int
	End   int
}

type Annotation struct {
	ID       int
	ChangeID string
	File     string
	OldLines lineRange
	NewLines lineRange
	Snippet  string
	Comment  string
}

type annotationStore struct {
	items   []Annotation
	nextID  int
	version uint64
}

func (s *annotationStore) Add(annotation Annotation) Annotation {
	s.nextID++
	annotation.ID = s.nextID
	s.items = append(s.items, annotation)
	s.version++
	return annotation
}

func (s *annotationStore) Find(id int) (Annotation, bool) {
	index := s.index(id)
	if index < 0 {
		return Annotation{}, false
	}
	return s.items[index], true
}

func (s *annotationStore) UpdateComment(id int, comment string) bool {
	index := s.index(id)
	if index < 0 {
		return false
	}
	s.items[index].Comment = comment
	s.version++
	return true
}

func (s *annotationStore) Remove(id int) bool {
	index := s.index(id)
	if index < 0 {
		return false
	}
	s.items = slices.Delete(s.items, index, index+1)
	s.version++
	return true
}

func (s *annotationStore) Clear() int {
	count := len(s.items)
	s.items = nil
	if count > 0 {
		s.version++
	}
	return count
}

func (s *annotationStore) All() []Annotation {
	return s.items
}

func (s *annotationStore) countRevision(changeID string) int {
	count := 0
	for _, annotation := range s.items {
		if annotation.ChangeID == changeID {
			count++
		}
	}
	return count
}

func (s *annotationStore) ForFile(changeID, file string) []Annotation {
	var annotations []Annotation
	for _, annotation := range s.items {
		if annotation.ChangeID == changeID && annotation.File == file {
			annotations = append(annotations, annotation)
		}
	}
	return annotations
}

func (s *annotationStore) index(id int) int {
	return slices.IndexFunc(s.items, func(annotation Annotation) bool {
		return annotation.ID == id
	})
}

func formatAnnotationsMarkdown(annotations []Annotation) string {
	var out strings.Builder
	for index, annotation := range annotations {
		if index > 0 {
			out.WriteString("\n\n")
		}
		revision := annotation.ChangeID
		if revision == "" {
			revision = "unknown"
		}

		locations := make([]string, 0, 2)
		if annotation.OldLines.Start != 0 || annotation.OldLines.End != 0 {
			locations = append(locations, "old "+formatLineRange(annotation.OldLines))
		}
		if annotation.NewLines.Start != 0 || annotation.NewLines.End != 0 {
			locations = append(locations, "new "+formatLineRange(annotation.NewLines))
		}
		if len(locations) == 0 {
			locations = append(locations, "line 0")
		}

		out.WriteString(fmt.Sprintf("### @%s (revision %s; %s)", annotation.File, revision, strings.Join(locations, ", ")))
		out.WriteString("\n\n")
		out.WriteString("```diff\n")
		out.WriteString(annotation.Snippet)
		out.WriteString("\n```\n\n")
		out.WriteString(annotation.Comment)
	}
	return out.String()
}

func formatLineRange(lines lineRange) string {
	if lines.Start == 0 && lines.End == 0 {
		return "0"
	}
	if lines.End == 0 {
		lines.End = lines.Start
	}
	if lines.Start == lines.End {
		return strconv.Itoa(lines.Start)
	}
	return fmt.Sprintf("%d-%d", lines.Start, lines.End)
}
