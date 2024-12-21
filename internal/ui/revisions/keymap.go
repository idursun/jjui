package revisions

import "github.com/charmbracelet/bubbles/key"

type keymap struct {
	current  rune
	bindings map[rune]interface{}
	up       key.Binding
	down     key.Binding
	cancel   key.Binding
	apply    key.Binding
}

type baseLayer struct {
	abandon      key.Binding
	edit         key.Binding
	diffedit     key.Binding
	split        key.Binding
	rebaseMode   key.Binding
	bookmarkMode key.Binding
	gitMode      key.Binding
	description  key.Binding
	diff         key.Binding
	new          key.Binding
	revset       key.Binding
	quit         key.Binding
}

type rebaseLayer struct {
	revision key.Binding
	branch   key.Binding
}

type bookmarkLayer struct {
	move   key.Binding
	set    key.Binding
	delete key.Binding
}

type gitLayer struct {
	fetch key.Binding
	push  key.Binding
}

func newKeyMap() keymap {
	bindings := make(map[rune]interface{})
	bindings[' '] = baseLayer{
		abandon:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "abandon")),
		edit:         key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		diffedit:     key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "diff edit")),
		split:        key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "split")),
		rebaseMode:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rebase")),
		bookmarkMode: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "bookmark")),
		gitMode:      key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "git")),
		description:  key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "description")),
		diff:         key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "diff")),
		new:          key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
		revset:       key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "revset")),
		quit:         key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}

	bindings['r'] = rebaseLayer{
		revision: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rebase revision")),
		branch:   key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "rebase branch")),
	}

	bindings['b'] = bookmarkLayer{
		move:   key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "bookmark move")),
		set:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "bookmark set")),
		delete: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "bookmark delete")),
	}

	bindings['g'] = gitLayer{
		fetch: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "git fetch")),
		push:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "git push")),
	}

	return keymap{
		current:  ' ',
		bindings: bindings,
		up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
		down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
		apply:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply")),
		cancel:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}

func (k *keymap) gitMode() {
	k.current = 'g'
}

func (k *keymap) rebaseMode() {
	k.current = 'r'
}

func (k *keymap) bookmarkMode() {
	k.current = 'b'
}

func (k *keymap) resetMode() {
	k.current = ' '
}

func (k *keymap) ShortHelp() []key.Binding {
	switch b := k.bindings[k.current].(type) {
	case baseLayer:
		return []key.Binding{k.up, k.down, b.revset, b.new, b.edit, b.diffedit, b.diff, b.abandon, b.description, b.split, b.rebaseMode, b.gitMode, b.bookmarkMode, b.quit}
	case rebaseLayer:
		return []key.Binding{b.revision, b.branch}
	case gitLayer:
		return []key.Binding{b.push, b.fetch}
	case bookmarkLayer:
		return []key.Binding{b.move, b.set, b.delete}
	default:
		return []key.Binding{}
	}
}

func (k *keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}
