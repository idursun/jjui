package ace_jump

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/bubbles/key"
)

type AceKey struct {
	RowIdx  int
	id      string
	bindIdx int
	bind    key.Binding
}

type AceJump struct {
	ace    []*AceKey
	prefix string
}

func NewAceJump() *AceJump {
	return &AceJump{
		ace:    []*AceKey{},
		prefix: "",
	}
}

func (j *AceJump) Prefix() *string {
	if j == nil {
		return nil
	}
	return &j.prefix
}

func (j *AceJump) Append(rowIdx int, id string, bindIdx int) {
	j.ace = append(j.ace, &AceKey{
		RowIdx:  rowIdx,
		id:      id,
		bindIdx: bindIdx,
		bind:    aceBindingAt(id, bindIdx),
	})
}

func (j *AceJump) First() *AceKey {
	return j.ace[0]
}

func (j *AceJump) bindKeys() {
	for _, a := range j.ace {
		a.bind = aceBindingAt(a.id, a.bindIdx)
	}
}

// returns the single match or nil after narrowing
func (j *AceJump) Narrow(k tea.KeyMsg) *AceKey {
	narrow := []*AceKey{}
	prefixIncremented := false
	for _, a := range j.ace {
		if key.Matches(k, a.bind) {
			if !prefixIncremented {
				prefixIncremented = true
				j.prefix = j.prefix + string(a.id[a.bindIdx])
			}
			narrow = append(narrow, a)
			if a.bindIdx+1 < len(a.id) {
				a.bindIdx++
			}
		}
	}

	if len(narrow) == 1 {
		return narrow[0]
	}

	if len(narrow) > 0 {
		j.ace = narrow
		j.bindKeys()
	}
	return nil
}

func aceBindingAt(id string, idx int) key.Binding {
	bs := string(id[idx])
	lbs, ubs := strings.ToLower(bs), strings.ToUpper(bs)
	return key.NewBinding(
		key.WithKeys(lbs, ubs),
		key.WithHelp(bs, id),
	)
}
