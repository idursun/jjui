package models

import (
	"github.com/idursun/jjui/internal/parser"
)

type RevisionItem struct {
	*Checkable
	parser.Row
	IsAffected bool
}
