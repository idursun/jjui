package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"

	ghostty "go.mitchellh.com/libghostty"
)

// encodeKey turns a logical key event into the bytes an application would
// receive from a terminal. Those bytes belong on the child process's PTY.
func encodeKey(term *ghostty.Terminal, key ghostty.Key, text string, mods ghostty.Mods) ([]byte, error) {
	encoder, err := ghostty.NewKeyEncoder()
	if err != nil {
		return nil, err
	}
	defer encoder.Close()

	// The terminal can change keyboard modes (for example, application cursor
	// mode), so keep the encoder synchronized before encoding each event.
	encoder.SetOptFromTerminal(term)

	event, err := ghostty.NewKeyEvent()
	if err != nil {
		return nil, err
	}
	defer event.Close()

	event.SetAction(ghostty.KeyActionPress)
	event.SetKey(key)
	event.SetMods(mods)
	event.SetUTF8(text)

	return encoder.Encode(event)
}

// sendKey writes one encoded key event to the application side of a PTY.
// In production, w would be the PTY master connected to the child process.
func sendKey(w io.Writer, term *ghostty.Terminal, key ghostty.Key, text string, mods ghostty.Mods) error {
	data, err := encodeKey(term, key, text, mods)
	if err != nil {
		return err
	}
	n, err := w.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

// screenSnapshot copies the terminal's current viewport into fixed-width
// text rows. This is intentionally a text snapshot: an assertion can compare
// these rows directly, while a renderer can use the same RenderState and cell
// APIs to inspect colors and styles when needed.
func screenSnapshot(term *ghostty.Terminal) ([]string, error) {
	styled, err := styledScreenSnapshot(term)
	if err != nil {
		return nil, err
	}
	return styledTextSnapshot(styled), nil
}

func styledTextSnapshot(styled styledScreen) []string {
	result := make([]string, 0, len(styled))
	for _, styledRow := range styled {
		var row strings.Builder
		for _, cell := range styledRow {
			if cell.Text == "" {
				row.WriteByte(' ')
			} else {
				row.WriteString(cell.Text)
			}
		}
		result = append(result, row.String())
	}
	return result
}

func main() {
	term, err := ghostty.NewTerminal(ghostty.WithSize(32, 6))
	if err != nil {
		log.Fatal(err)
	}
	defer term.Close()

	// This is application output: a real harness copies bytes read from the
	// child PTY into VTWrite.
	term.VTWrite([]byte("\x1b[2J\x1b[H\x1b[1;36mjjui\x1b[0m test\r\n> "))

	// This is application input: a real harness writes these bytes to the
	// child PTY. The buffer is only a stand-in for that PTY in this demo.
	var childInput bytes.Buffer
	if err := sendKey(&childInput, term, ghostty.KeyJ, "j", 0); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("encoded input: %q\n", childInput.String())

	// Simulate the child application echoing what it received. Do not feed
	// childInput directly to VTWrite in a real harness: VTWrite consumes output
	// from the application, not user input.
	term.VTWrite(append(childInput.Bytes(), '\r', '\n'))

	screen, err := screenSnapshot(term)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range screen {
		fmt.Printf("%q\n", strings.TrimRight(row, " "))
	}
}
