package test

import (
	"bufio"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/muesli/termenv"
)

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

type LogBuilder struct {
	w strings.Builder
}

func (l *LogBuilder) String() string {
	return l.w.String()
}

func (l *LogBuilder) Write(line string) {
	lipgloss.SetColorProfile(termenv.ANSI)
	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "short_id=") {
			text = strings.TrimPrefix(text, "short_id=")
			l.ShortId(text)
			continue
		}
		if strings.HasPrefix(text, "id=") {
			text = strings.TrimPrefix(text, "id=")
			l.Id(text[:1], text[1:])
			continue
		}
		if strings.HasPrefix(text, "author=") {
			l.Author(strings.TrimPrefix(text, "author="))
			continue
		}
		if strings.HasPrefix(text, "bookmarks=") {
			text = strings.TrimPrefix(text, "bookmarks=")
			values := strings.Split(text, ",")
			l.Bookmarks(strings.Join(values, " "))
			continue
		}
		l.Append(text)
	}
	l.w.WriteString("\n")
}

func (l *LogBuilder) Append(value string) {
	fmt.Fprintf(&l.w, "%s ", styles[normal].Render(value))
}

func (l *LogBuilder) ShortId(sid string) {
	fmt.Fprintf(&l.w, " %s ", styles[id].Render(sid))
}

func (l *LogBuilder) Id(short string, rest string) {
	fmt.Fprintf(&l.w, " %s%s ", styles[id].Render(short), styles[id].Render(rest))
}

func (l *LogBuilder) Author(value string) {
	fmt.Fprintf(&l.w, " %s ", styles[author].Render(value))
}

func (l *LogBuilder) Bookmarks(value string) {
	fmt.Fprintf(&l.w, " %s ", styles[bookmark].Render(value))
}
