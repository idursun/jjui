package bindings

import "testing"

func TestLoad_Bindings(t *testing.T) {
	const config = `
[[keybindings]]
keys = ["n"]
action = "revisions.new"
when = "revisions.focused"

[[keybindings]]
keys = ["n"]
action = "-revisions.new"
when = "revisions.focused && inline_describe.active"

[[keybindings]]
keys = ["c"]
action = "revisions.commit"
args = { message_prompt = true }

[[keybindings]]
key_sequence = ["g", "o"]
action = "revisions.go"
`
	bs, err := Load(config)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(bs) != 4 {
		t.Fatalf("expected 4 bindings, got %d", len(bs))
	}

	b := bs[0]
	if b.Action != "revisions.new" || b.Disabled {
		t.Fatalf("unexpected first binding: %+v", b)
	}
	if b.When != "revisions.focused" {
		t.Fatalf("unexpected when: %q", b.When)
	}
	if !b.Condition.Eval(map[string]any{"revisions.focused": true}) {
		t.Fatalf("expected condition to evaluate to true")
	}

	disabled := bs[1]
	if !disabled.Disabled {
		t.Fatalf("expected binding to be disabled: %+v", disabled)
	}
	if disabled.Action != "revisions.new" {
		t.Fatalf("unexpected action for disabled binding: %q", disabled.Action)
	}

	withArgs := bs[2]
	if withArgs.Args["message_prompt"] != true {
		t.Fatalf("expected args to include message_prompt=true, got %+v", withArgs.Args)
	}

	withSequence := bs[3]
	if len(withSequence.Keys) != 0 {
		t.Fatalf("expected no direct keys for sequence binding, got %+v", withSequence.Keys)
	}
	if withSequence.Action != "revisions.go" || len(withSequence.KeySequence) != 2 {
		t.Fatalf("unexpected sequence binding: %+v", withSequence)
	}
}

func TestLoad_Validations(t *testing.T) {
	_, err := Load(`[[keybindings]]`)
	if err == nil {
		t.Fatal("expected error for missing keys and action")
	}

	_, err = Load(`[[keybindings]] keys=["n"] action="  "`)
	if err == nil {
		t.Fatal("expected error for empty action")
	}

	_, err = Load(`[[keybindings]] keys=["n"] key_sequence=["g"] action="revisions.new"`)
	if err == nil {
		t.Fatal("expected error when both keys and key_sequence are set")
	}
}

func TestResolve(t *testing.T) {
	state := map[string]any{
		"revisions.focused":       true,
		"inline_describe.active":  false,
		"inline_describe.enabled": true,
	}
	bs, err := Load(`
[[keybindings]]
keys = ["n"]
action = "revisions.new"
when = "revisions.focused && inline_describe.enabled"
`)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	b, ok := Resolve("revisions.new", bs, state)
	if !ok {
		t.Fatalf("expected binding to resolve")
	}
	if b.Disabled {
		t.Fatalf("expected resolved binding to be enabled")
	}
}
