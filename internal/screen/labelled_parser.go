package screen

import (
	"bufio"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
)

// ParseLabelledFromReader parses output produced with `--color=debug`, where jj
// wraps every templated span as `<<label1 label2::text>>`. The labels are
// attached to the resulting segments and the markers are removed.
func ParseLabelledFromReader(r io.Reader) <-chan *Segment {
	ch := make(chan *Segment)
	go func() {
		defer close(ch)
		p := labelledParser{
			style:    lipgloss.NewStyle(),
			interned: map[string][]string{},
			out:      ch,
		}
		reader := bufio.NewReader(r)
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				p.parseLine(line)
			}
			if err != nil {
				break
			}
		}
		p.flush()
	}()
	return ch
}

type labelledParser struct {
	style    lipgloss.Style
	labels   []string
	inSpan   bool
	buffer   strings.Builder
	interned map[string][]string
	out      chan<- *Segment
}

func (p *labelledParser) parseLine(line string) {
	for i := 0; i < len(line); {
		rest := line[i:]
		switch {
		case strings.HasPrefix(rest, "\x1b["):
			end := strings.IndexByte(rest, 'm')
			if end == -1 {
				p.buffer.WriteString(rest)
				return
			}
			p.flush()
			if params := rest[2:end]; params == "0" {
				p.style = lipgloss.NewStyle()
			} else {
				p.style = applyParamsToStyle(p.style, params)
			}
			i += end + 1
		case !p.inSpan && strings.HasPrefix(rest, "<<"):
			labels, n, ok := spanStart(rest)
			if !ok {
				p.buffer.WriteByte(line[i])
				i++
				continue
			}
			p.flush()
			if _, seen := p.interned[labels]; !seen {
				p.interned[labels] = strings.Fields(labels)
			}
			p.labels = p.interned[labels]
			p.inSpan = true
			i += n
		case p.inSpan && strings.HasPrefix(rest, ">>") && isSpanEnd(rest[2:]):
			// Empty spans are kept: their labels still say what the line is,
			// e.g. a blank line inside a description.
			if p.buffer.Len() == 0 {
				p.out <- &Segment{Style: p.style, Labels: p.labels}
			}
			p.flush()
			p.labels = nil
			p.inSpan = false
			i += 2
		default:
			p.buffer.WriteByte(line[i])
			i++
		}
	}
}

func (p *labelledParser) flush() {
	if p.buffer.Len() == 0 {
		return
	}
	p.out <- &Segment{Text: p.buffer.String(), Style: p.style, Labels: p.labels}
	p.buffer.Reset()
}

// spanStart reports whether s starts with `<<labels::` and returns the labels
// and the length of the marker.
func spanStart(s string) (string, int, bool) {
	end := strings.Index(s, "::")
	if end < 2 {
		return "", 0, false
	}
	labels := s[2:end]
	if !isLabelList(labels) {
		return "", 0, false
	}
	return labels, end + 2, true
}

func isLabelList(s string) bool {
	if s == "" || s[0] == ' ' || s[len(s)-1] == ' ' || strings.Contains(s, "  ") {
		return false
	}
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_', c == '-', c == '.', c == ' ':
		default:
			return false
		}
	}
	return true
}

// isSpanEnd decides whether a `>>` closes the current span. jj does not escape
// span text, so it is a close only if the rest of the line reaches the next
// span start or the end of the line without meeting another `>>`.
func isSpanEnd(rest string) bool {
	for j := 0; j < len(rest); j++ {
		switch {
		case rest[j] == '\n':
			return true
		case strings.HasPrefix(rest[j:], "<<"):
			_, _, ok := spanStart(rest[j:])
			return ok
		case strings.HasPrefix(rest[j:], ">>"):
			return false
		}
	}
	return true
}
