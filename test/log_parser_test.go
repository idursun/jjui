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
	parse := parser.Parse()
	assert.Len(t, parse, 1)
}
