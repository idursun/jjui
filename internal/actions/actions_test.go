package actions

import "testing"

func TestLoad(t *testing.T) {
	const cfg = `
[actions."revisions.new"]
lua = '''
jj("new")
'''

[actions."revisions.commit"]
lua = 'jj("commit")'
`
	registry, err := Load(cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(registry) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(registry))
	}
	if registry["revisions.new"].Lua == "" {
		t.Fatalf("expected lua content for revisions.new")
	}
}

func TestLoad_Validation(t *testing.T) {
	_, err := Load(`
[actions."revisions.new"]
lua = ""
`)
	if err == nil {
		t.Fatal("expected error for empty lua")
	}
}
