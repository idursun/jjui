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
	Params string
}

func (s Segment) String() string {
	if s.Text == "\n" {
		return s.Text
	}
	if s.Params == "" {
		return s.Text
	}
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", s.Params, s.Text)
}

func (s Segment) WithBackground(bg string) string {
	var newParts []string
	parts := strings.Split(s.Params, ";")

	i := 0
	for i < len(parts) {
		part := parts[i]
		num, err := strconv.Atoi(part)
		if err != nil {
			i++
			continue
		}
		p := num

		isBg := false
		if (p >= 40 && p <= 49) || (p >= 100 && p <= 109) {
			isBg = true
		} else if p == 48 && i+1 < len(parts) {
			next, err := strconv.Atoi(parts[i+1])
			if err == nil {
				if next == 5 && i+2 < len(parts) {
					i += 3
					isBg = true
				} else if next == 2 && i+4 < len(parts) {
					i += 5
					isBg = true
				}
			}
		}

		if !isBg {
			newParts = append(newParts, part)
		}
		i++
	}

	for _, part := range strings.Split(bg, ";") {
		if _, err := strconv.Atoi(part); err != nil {
			panic(fmt.Sprintf("invalid background parameter %q", part))
		}
		newParts = append(newParts, part)
	}

	return Segment{
		Text:   s.Text,
		Params: strings.Join(newParts, ";"),
	}.String()
}

func (s Segment) StyleEqual(other Segment) bool {
	return s.Params == other.Params
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
		currentParams := ""
		reader := bufio.NewReader(r)

		for {
			b, err := reader.ReadByte()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
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

					var seq bytes.Buffer
					for {
						c, err := reader.ReadByte()
						if err != nil || c == 'm' {
							break
						}
						seq.WriteByte(c)
					}

					paramStr := seq.String()
					if paramStr == "0" {
						currentParams = ""
					} else {
						currentParams = paramStr
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

		if buffer.Len() > 0 {
			ch <- &Segment{
				Text:   buffer.String(),
				Params: currentParams,
			}
		}
	}()
	return ch
}
