package screen

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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

func (s Segment) WithBackground(bg string) string {
	newParams := make([]int, 0, len(s.Params)+5)
	i := 0
	for i < len(s.Params) {
		p := s.Params[i]
		if (p >= 40 && p <= 49) || (p >= 100 && p <= 109) {
			if p == 48 {
				if i+1 < len(s.Params) {
					next := s.Params[i+1]
					if next == 5 {
						if i+2 < len(s.Params) {
							i += 3
						} else {
							i = len(s.Params)
						}
						continue
					} else if next == 2 {
						if i+4 < len(s.Params) {
							i += 5
						} else {
							i = len(s.Params)
						}
						continue
					}
				}
			}
			i++
			continue
		}
		newParams = append(newParams, p)
		i++
	}

	parts := strings.Split(bg, ";")
	bgParams := make([]int, 0, len(parts))
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			panic(fmt.Sprintf("invalid background parameter %q", part))
		}
		bgParams = append(bgParams, num)
	}
	newParams = append(newParams, bgParams...)

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
	for segment := range ParseFromReader(bytes.NewReader(raw)) {
		segments = append(segments, *segment)
	}
	return segments
}

func ParseFromReader(r io.Reader) <-chan *Segment {
	ch := make(chan *Segment)
	go func() {
		defer close(ch)
		var buffer bytes.Buffer
		var currentParams []int
		reader := bufio.NewReader(r)

		for {
			b, err := reader.ReadByte()
			if err == io.EOF {
				break
			}
			if err != nil {
				break // Handle error as needed
			}

			if b == 0x1B {
				peekBytes, err := reader.Peek(1)
				if err != nil {
					buffer.WriteByte(b)
					break
				}

				if len(peekBytes) >= 1 && peekBytes[0] == '[' {
					_, _ = reader.Discard(1)
					if buffer.Len() > 0 {
						ch <- &Segment{
							Text:   buffer.String(),
							Params: currentParams,
						}
						buffer.Reset()
					}

					// Read until 'm'
					var seq bytes.Buffer
					for {
						c, err := reader.ReadByte()
						if err != nil || c == 'm' {
							break
						}
						seq.WriteByte(c)
					}

					// Parse parameters
					var newParams []int
					s := seq.String()
					start := 0
					for i := 0; i <= len(s); i++ {
						if i == len(s) || s[i] == ';' {
							if start < i {
								num, _ := strconv.Atoi(s[start:i])
								newParams = append(newParams, num)
							}
							start = i + 1
						}
					}

					// Update parameters
					if len(newParams) == 1 && newParams[0] == 0 {
						currentParams = nil
					} else {
						currentParams = newParams
					}
				} else {
					buffer.WriteByte(b)
					if len(peekBytes) >= 1 {
						nextByte, _ := reader.ReadByte()
						buffer.WriteByte(nextByte)
					}
				}
			} else {
				buffer.WriteByte(b)
			}
		}

		// Add final segment
		if buffer.Len() > 0 {
			ch <- &Segment{
				Text:   buffer.String(),
				Params: currentParams,
			}
		}
	}()
	return ch
}
