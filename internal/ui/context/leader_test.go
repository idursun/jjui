package context

import (
	"testing"
)

const exampleLeaderToml = `
[leader.M]
help = "Set bookmark main and push"
send = ["b/move main", "down", "enter", "gp", "enter"]

[leader.g]
help = "Git"

[leader.gff]
help = "Git Fetch"
send = ["gf", "enter"]

[leader.gfa]
help = "Git Fetch All"
send = ["g/fetch --all", "down", "enter"]
`

func TestLoadLeader(t *testing.T) {
	lm, err := LoadLeader(exampleLeaderToml)
	if err != nil {
		t.Fatalf("LoadLeader failed: %v", err)
	}

	// Test leader.M
	m := lm["M"]
	if m == nil {
		t.Error("leader.M not found")
	} else {
		if m.Bind == nil || m.Bind.Help().Desc != "Set bookmark main and push" {
			t.Errorf("leader.M help mismatch: got %q", m.Bind.Help().Desc)
		}
		if len(m.Send) != 5 || m.Send[0] != "b/move main" || m.Send[4] != "enter" {
			t.Errorf("leader.M send mismatch: got %v", m.Send)
		}
	}

	// Test leader.g
	g := lm["g"]
	if g == nil {
		t.Error("leader.g not found")
	} else {
		if g.Bind == nil || g.Bind.Help().Desc != "Git" {
			t.Errorf("leader.g help mismatch: got %q", g.Bind.Help().Desc)
		}
		if len(g.Send) != 0 {
			t.Errorf("leader.g send should be empty: got %v", g.Send)
		}
	}

	// Test leader.gff
	gff := lm["g"].Nest["f"].Nest["f"]
	if gff == nil {
		t.Error("leader.gff not found")
	} else {
		if gff.Bind == nil || gff.Bind.Help().Desc != "Git Fetch" {
			t.Errorf("leader.gff help mismatch: got %q", gff.Bind.Help().Desc)
		}
		if len(gff.Send) != 2 || gff.Send[0] != "gf" || gff.Send[1] != "enter" {
			t.Errorf("leader.gff send mismatch: got %v", gff.Send)
		}
	}

	// Test leader.gf
	h := lm["g"].Nest["f"]
	if h == nil {
		t.Error("leader.gf not found")
	} else {
		if h.Bind == nil || h.Bind.Help().Desc != "" {
			t.Errorf("leader.gf help mismatch: got %q", h.Bind.Help().Desc)
		}
		if len(h.Send) != 0 {
			t.Errorf("leader.gf send mismatch: got %v", h.Send)
		}
	}
}
