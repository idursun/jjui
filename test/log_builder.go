package test

import (
	"fmt"
	"strings"
)

// LogBuilder writes log lines the way `jj log --color=debug` labels them.
type LogBuilder struct {
	w strings.Builder
}

func (l *LogBuilder) String() string {
	return l.w.String()
}

// Revision writes a revision line; "@" as node marks the working copy.
func (l *LogBuilder) Revision(gutter string, node string, changeId string, commitId string) {
	labels := "log commit mutable"
	if node == "@" {
		labels = "log commit working_copy mutable"
	}
	fmt.Fprintf(&l.w, "%s<<log commit node %s::%s>>  ", gutter, strings.TrimPrefix(labels, "log commit "), node)
	fmt.Fprintf(&l.w, "<<%s change_id shortest prefix::%s>><<%s:: >>", labels, changeId, labels)
	fmt.Fprintf(&l.w, "<<%s commit_id shortest prefix::%s>><<%s::>>\n", labels, commitId, labels)
}

// Description writes a description line below a revision.
func (l *LogBuilder) Description(gutter string, text string) {
	fmt.Fprintf(&l.w, "%s<<log commit mutable description first_line::%s>><<log commit mutable::>>\n", gutter, text)
}

// Graph writes a line that only draws the graph.
func (l *LogBuilder) Graph(gutter string) {
	l.w.WriteString(gutter + "\n")
}
