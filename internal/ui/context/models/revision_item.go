package models

import "github.com/idursun/jjui/internal/parser"

type RevisionItem struct {
	BaseItem
	Checkable
	parser.Row
	checked bool
}
