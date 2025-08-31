package models

import "github.com/idursun/jjui/internal/parser"

type RevisionItem struct {
	BaseItem
	Checkable
	parser.Row
	checked bool
}

func (r *RevisionItem) IsChecked() bool {
	return r.checked
}

func (r *RevisionItem) Toggle() {
	r.checked = !r.checked
}
