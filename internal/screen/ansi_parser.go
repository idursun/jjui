package screen

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type Segment struct {
	Text   string
	Params []int
}

func (s Segment) String() string {
	if s.Text == "\n" {
		return s.Text
	}
	if len(s.Params) == 0 {
		return s.Text
	}

	params := make([]string, len(s.Params))
	for i, p := range s.Params {
		params[i] = strconv.Itoa(p)
	}
	return fmt.Sprintf(
		"\x1b[%sm%s\x1b[0m",
		strings.Join(params, ";"),
		s.Text,
	)
}

func (s Segment) WithBackground(bg int) string {
	newParams := make([]int, 0, len(s.Params)+1)
	for _, p := range s.Params {
		if (p >= 40 && p <= 49) || (p >= 100 && p <= 109) {
			continue
		}
		newParams = append(newParams, p)
	}
	newParams = append(newParams, bg)
	newSegment := Segment{
		Text:   s.Text,
		Params: newParams,
	}
	return newSegment.String()
}

func (s Segment) StyleEqual(other Segment) bool {
	if len(other.Params) != len(s.Params) {
		return false
	}
	for i, p := range s.Params {
		if p != other.Params[i] {
			return false
		}
	}
	return true
}

func Parse(raw []byte) []Segment {
	var segments []Segment
	var buffer bytes.Buffer
	var params []int
	pos := 0

	for pos < len(raw) {
		if raw[pos] == 0x1B && pos+1 < len(raw) && raw[pos+1] == '[' {
			// Save current buffer
			if buffer.Len() > 0 {
				segments = append(segments, Segment{
					Text:   buffer.String(),
					Params: params,
				})
				params = nil
				buffer.Reset()
			}

			// Extract full escape sequence
			end := bytes.IndexByte(raw[pos:], 'm')
			if end == -1 {
				pos++
				continue
			}
			end += pos

			// Parse parameters
			seq := raw[pos+2 : end]
			start := 0
			for i := 0; i <= len(seq); i++ {
				if i == len(seq) || seq[i] == ';' {
					if start < i {
						paramBytes := seq[start:i]
						if num, err := strconv.Atoi(string(paramBytes)); err == nil {
							params = append(params, num)
						}
					}
					start = i + 1
				}
			}

			// Handle reset
			if len(params) == 1 && params[0] == 0 {
				params = nil
			}

			pos = end + 1
		} else {
			buffer.WriteByte(raw[pos])
			pos++
		}
	}

	// Add remaining text
	if buffer.Len() > 0 {
		segments = append(segments, Segment{
			Text:   buffer.String(),
			Params: params,
		})
	}

	return segments
}
