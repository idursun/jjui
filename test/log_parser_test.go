package test

import (
	"bufio"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/jj"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	file, _ := os.Open("testdata/output.log")
	parser := jj.NewNoTemplateParser(file)
	rows := parser.Parse()
	assert.Len(t, rows, 11)
}

func TestParser_Parse_Disconnected(t *testing.T) {
	var lb logBuilder
	lb.write("*   id=abcde author=some@author id=xyrq")
	lb.write("│   some documentation")
	lb.write("~\n")
	lb.write("*   id=abcde author=some@author id=xyrq")
	lb.write("│   another commit")
	lb.write("~\n")
	parser := jj.NewNoTemplateParser(strings.NewReader(lb.String()))
	rows := parser.Parse()
	assert.Len(t, rows, 2)
}

func TestParser_Parse_Extend(t *testing.T) {
	var lb logBuilder
	lb.write("*   id=abcde author=some@author id=xyrq")
	lb.write("│   some documentation")

	parser := jj.NewNoTemplateParser(strings.NewReader(lb.String()))
	rows := parser.Parse()
	assert.Len(t, rows, 1)
	row := rows[0]

	extended := row.SegmentLines[1].Extend(row.Indent)
	assert.Len(t, extended.Segments, 1)
}

type part int

const (
	normal = iota
	id
	author
	bookmark
)

var styles = map[part]lipgloss.Style{
	normal:   lipgloss.NewStyle(),
	id:       lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	author:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	bookmark: lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
}

type logBuilder struct {
	w strings.Builder
}

func (l *logBuilder) String() string {
	return l.w.String()
}

func (l *logBuilder) write(line string) {
	lipgloss.SetColorProfile(termenv.ANSI)
	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "id=") {
			text = strings.TrimPrefix(text, "id=")
			l.id(text[:1], text[1:])
			continue
		}
		if strings.HasPrefix(text, "author=") {
			l.author(strings.TrimPrefix(text, "author="))
			continue
		}
		if strings.HasPrefix(text, "bookmarks=") {
			text = strings.TrimPrefix(text, "bookmarks=")
			values := strings.Split(text, ",")
			l.bookmarks(strings.Join(values, " "))
			continue
		}
		l.append(text)
	}
	l.w.WriteString("\n")
}

func (l *logBuilder) append(value string) {
	fmt.Fprintf(&l.w, "%s ", styles[normal].Render(value))
}

func (l *logBuilder) id(short string, rest string) {
	fmt.Fprintf(&l.w, " %s%s ", styles[id].Render(short), styles[id].Render(rest))
}

func (l *logBuilder) author(value string) {
	fmt.Fprintf(&l.w, " %s ", styles[author].Render(value))
}

func (l *logBuilder) bookmarks(value string) {
	fmt.Fprintf(&l.w, " %s ", styles[bookmark].Render(value))
}
