package screen

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
							Text:  buffer.String(),
							Style: paramToStyle(currentParams),
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
				Text:  buffer.String(),
				Style: paramToStyle(currentParams),
			}
		}
	}()
	return ch
}

func paramToStyle(param string) lipgloss.Style {
	if param == "" {
		return lipgloss.NewStyle()
	}

	style := lipgloss.NewStyle()
	parts := strings.Split(param, ";")

	for i := 0; i < len(parts); i++ {
		code, err := strconv.Atoi(parts[i])
		if err != nil {
			continue
		}

		switch {
		case code == 0:
			// Reset
			style = lipgloss.NewStyle()
		case code == 1:
			// Bold
			style = style.Bold(true)
		case code == 2:
			// Dim
			style = style.Faint(true)
		case code == 3:
			// Italic
			style = style.Italic(true)
		case code == 4:
			// Underline
			style = style.Underline(true)
		case code == 5 || code == 6:
			// Blink (slow or rapid)
			style = style.Blink(true)
		case code == 7:
			// Reverse
			style = style.Reverse(true)
		case code == 8:
			// Hidden
			// Not directly supported in lipgloss, could use faint + background
			style = style.Faint(true)
		case code == 9:
			// Strikethrough
			style = style.Strikethrough(true)
		case code >= 30 && code <= 37:
			// Foreground color
			style = style.Foreground(lipgloss.ANSIColor(code - 30))
		case code == 38 && i+2 < len(parts):
			// Extended foreground color
			if parts[i+1] == "5" && i+2 < len(parts) { // 8-bit color
				colorIndex, err := strconv.Atoi(parts[i+2])
				if err == nil {
					style = style.Foreground(lipgloss.Color(strconv.Itoa(colorIndex)))
				}
				i += 2
			} else if parts[i+1] == "2" && i+4 < len(parts) { // 24-bit color
				r, errR := strconv.Atoi(parts[i+2])
				g, errG := strconv.Atoi(parts[i+3])
				b, errB := strconv.Atoi(parts[i+4])
				if errR == nil && errG == nil && errB == nil {
					hexColor := fmt.Sprintf("#%02x%02x%02x", r, g, b)
					style = style.Foreground(lipgloss.Color(hexColor))
				}
				i += 4
			}
		case code == 39:
			// Default foreground
			style = style.UnsetForeground()
		case code >= 40 && code <= 47:
			// Background color
			style = style.Background(lipgloss.ANSIColor(code - 40))
		case code == 48 && i+2 < len(parts):
			// Extended background color
			if parts[i+1] == "5" && i+2 < len(parts) { // 8-bit color
				colorIndex, err := strconv.Atoi(parts[i+2])
				if err == nil {
					style = style.Background(lipgloss.Color(strconv.Itoa(colorIndex)))
				}
				i += 2
			} else if parts[i+1] == "2" && i+4 < len(parts) { // 24-bit color
				r, errR := strconv.Atoi(parts[i+2])
				g, errG := strconv.Atoi(parts[i+3])
				b, errB := strconv.Atoi(parts[i+4])
				if errR == nil && errG == nil && errB == nil {
					hexColor := fmt.Sprintf("#%02x%02x%02x", r, g, b)
					style = style.Background(lipgloss.Color(hexColor))
				}
				i += 4
			}
		case code == 49:
			// Default background
			style = style.UnsetBackground()
		case code >= 90 && code <= 97:
			// Bright foreground color
			style = style.Foreground(lipgloss.ANSIColor(code - 90 + 8))
		case code >= 100 && code <= 107:
			// Bright background color
			style = style.Background(lipgloss.ANSIColor(code - 100 + 8))
		}
	}

	return style
}
