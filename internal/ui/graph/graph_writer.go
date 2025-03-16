package graph

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/jj"
)

type GraphWriter struct {
	buffer             bytes.Buffer
	lineCount          int
	connectionPos      int
	connections        []jj.ConnectionType
	connectionsWritten bool
	renderer           RowRenderer
	row                jj.GraphRow
	Width              int
}

func (w *GraphWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	w.lineCount += bytes.Count(p, []byte("\n"))
	return w.buffer.Write(p)
}

func (w *GraphWriter) LineCount() int {
	return w.lineCount
}

func (w *GraphWriter) String(start, end int) string {
	lines := strings.Split(w.buffer.String(), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	for end > len(lines) {
		lines = append(lines, "")
	}
	return strings.Join(lines[start:end], "\n")
}

func (w *GraphWriter) Reset() {
	w.buffer.Reset()
	w.lineCount = 0
}

func (w *GraphWriter) RenderRow(row jj.GraphRow, renderer RowRenderer, highlighted bool) {
	w.connectionPos = 0
	w.connectionsWritten = false
	w.row = row
	w.renderer = renderer
	w.connections = extendConnections(w.connections)
	renderer.BeginSection(RowSectionBefore)
	// will render by extending the previous connections
	written, _ := w.Write([]byte(renderer.RenderBefore(row.Commit)))
	if written > 0 {
		w.Write([]byte("\n"))
	}
	w.connectionsWritten = false
	w.connections = row.Connections[0]
	prefix := len(w.connections)*2 + 1
	renderer.BeginSection(RowSectionRevision)
	for _, segmentedLine := range row.SegmentLines {
		lw := strings.Builder{}
		for _, segment := range segmentedLine.Segments {
			if highlighted {
				fmt.Fprint(&lw, segment.WithBackground(40))
			} else {
				fmt.Fprint(&lw, segment.String())
			}
		}
		line := lw.String()
		fmt.Fprint(w, line)
		width := lipgloss.Width(line)
		gap := w.Width - prefix - width
		if gap > 0 {
			fmt.Fprint(w, renderer.RenderNormal(strings.Repeat(" ", gap)))
		}
		fmt.Fprint(w, "\n")
	}

	if row.Commit.IsRoot() {
		return
	}
	lastLineConnection := extendConnections(row.Connections[0])
	if len(row.Connections) > 1 && !slices.Contains(row.Connections[1], jj.TERMINATION) {
		w.connectionPos = 1
		lastLineConnection = row.Connections[1]
	}

	renderer.BeginSection(RowSectionAfter)
	w.connections = extendConnections(lastLineConnection)
	written, _ = w.Write([]byte(renderer.RenderAfter(row.Commit)))
	if written > 0 {
		w.Write([]byte("\n"))
	}

	w.connectionPos++
	for w.connectionPos < len(row.Connections) {
		w.connections = row.Connections[w.connectionPos]
		w.renderConnections()
		if slices.Contains(w.connections, jj.TERMINATION) {
			w.buffer.Write([]byte(w.renderer.RenderTermination(" (elided revisions)")))
		}
		w.buffer.Write([]byte("\n"))
		w.lineCount++
		w.connectionPos++
	}
}

func (w *GraphWriter) renderConnections() {
	return
	//if w.connections == nil {
	//	w.connectionsWritten = true
	//	return
	//}
	//maxPadding := 0
	//for _, c := range w.row.Connections {
	//	if len(c) > maxPadding {
	//		maxPadding = len(c)
	//	}
	//}
	//
	//for _, c := range w.connections {
	//	if c == jj.GLYPH || c == jj.GLYPH_IMMUTABLE || c == jj.GLYPH_WORKING_COPY || c == jj.GLYPH_CONFLICT {
	//		w.buffer.WriteString(w.renderer.RenderGlyph(c, w.row.Commit))
	//	} else if c == jj.TERMINATION {
	//		w.buffer.WriteString(w.renderer.RenderTermination(c))
	//	} else {
	//		w.buffer.WriteString(w.renderer.RenderConnection(c))
	//	}
	//}
	//if len(w.connections) < maxPadding {
	//	w.buffer.WriteString(strings.Repeat(jj.SPACE, maxPadding-len(w.connections)))
	//}
	//w.connectionsWritten = true
}

func extendConnections(connections []jj.ConnectionType) []jj.ConnectionType {
	if connections == nil {
		return nil
	}
	extended := make([]jj.ConnectionType, 0)
	for i, cur := range connections {
		if cur != jj.MERGE_LEFT && cur != jj.MERGE_BOTH && cur != jj.MERGE_RIGHT && cur != jj.HORIZONTAL && cur != jj.SPACE {
			extended = append(extended, jj.VERTICAL)
		} else if i != len(connections)-1 {
			extended = append(extended, jj.SPACE)
		}
	}
	return extended
}
