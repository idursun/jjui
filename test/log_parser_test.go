package test

import (
	"github.com/idursun/jjui/internal/jj"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	file, _ := os.Open("testdata/output.log")

	parser := jj.NewNoTemplateParser(file)
	rows := parser.Parse()
	assert.Len(t, rows, 11)
}

func TestParser_Parse_Disconnected(t *testing.T) {
	file, _ := os.Open("testdata/disconnected.log")

	parser := jj.NewNoTemplateParser(file)
	rows := parser.Parse()
	assert.Len(t, rows, 5)

}
