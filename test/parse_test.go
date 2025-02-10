package test

import (
	"os"
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/stretchr/testify/assert"
)

func Test_Parse_Line(t *testing.T) {
	tests := []struct {
		line     string
		changeId string
	}{
		{"│ │ │ ○ │ │ │ │ │   │ │ │  yskmz;yskmzrpp", "yskmz"},
	}
	for _, test := range tests {
		t.Run(test.line, func(t *testing.T) {
			p := jj.NewParser(strings.NewReader(test.line))
			rows := p.Parse()
			assert.Equal(t, 1, len(rows))
			assert.Equal(t, test.changeId, rows[0].Commit.ChangeIdShort)
		})
	}
}

func Test_Parse_File(t *testing.T) {
	tests := []struct {
		logFile     string
		highlighted string
	}{
		{"testdata/many-levels.log", ""},
		{"testdata/conflicted.log", ""},
		{"testdata/merges-with-elided-revisions.log", ""},
		{"testdata/before-rendering.log", "up"},
	}
	for _, test := range tests {
		t.Run(test.logFile, func(t *testing.T) {
			file, err := os.Open(test.logFile)
			if err != nil {
				t.Fatalf("could not open file: %v", err)
			}

			p := jj.NewParser(file)
			rows := p.Parse()
			var w jj.GraphWriter
			for _, row := range rows {
				w.RenderRow(row, TestRenderer{highlighted: row.Commit.ChangeIdShort == test.highlighted})
			}
			actual := w.String(0, w.LineCount())
			renderedFileName := strings.Replace(test.logFile, ".log", ".expected", 1)
			content, err := os.ReadFile(renderedFileName)
			if err != nil {
				t.Fatalf("could not read file: %v", err)
			}

			_ = file.Close()
			assert.Equal(t, string(content), actual)
		})
	}
}

type TestRenderer struct {
	highlighted bool
}

func (t TestRenderer) RenderMarkers(*jj.Commit) string {
	return ""
}

func (t TestRenderer) RenderBefore(*jj.Commit) string {
	if t.highlighted {
		return "<here>"
	}
	return ""
}

func (t TestRenderer) RenderAfter(*jj.Commit) string {
	return ""
}

func (t TestRenderer) RenderGlyph(connection jj.ConnectionType, _ *jj.Commit) string {
	return string(connection)
}

func (t TestRenderer) RenderTermination(connection jj.ConnectionType) string {
	return string(connection)
}

func (t TestRenderer) RenderChangeId(commit *jj.Commit) string {
	return commit.ChangeId
}

func (t TestRenderer) RenderAuthor(commit *jj.Commit) string {
	if commit.IsRoot() {
		return "root()"
	}
	return ""
}

func (t TestRenderer) RenderDate(*jj.Commit) string {
	return ""
}

func (t TestRenderer) RenderBookmarks(*jj.Commit) string {
	return ""
}

func (t TestRenderer) RenderDescription(commit *jj.Commit) string {
	if commit.IsRoot() {
		return ""
	}
	var w strings.Builder
	if commit.Empty {
		w.WriteString("(empty) ")
	}
	if commit.Description == "" {
		w.WriteString("(no description set)")
	} else {
		w.WriteString(commit.Description)
	}
	return w.String()
}
