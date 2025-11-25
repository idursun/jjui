package leader

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

func Test_sendCmds_detects_tea_keytypes(t *testing.T) {
	cmds := sendCmds([]string{"enter", "ctrl+s"})
	if len(cmds) != 2 {
		t.Fatalf("expected 2 cmds, got %d", len(cmds))
	}
	msg1 := cmds[0]()
	msg2 := cmds[1]()
	key1, ok1 := msg1.(tea.KeyMsg)
	key2, ok2 := msg2.(tea.KeyMsg)
	if !ok1 || !ok2 {
		t.Fatalf("expected tea.KeyMsg, got %T and %T", msg1, msg2)
	}
	if key1.Type != tea.KeyEnter {
		t.Errorf("expected key1.Type == KeyEnter, got %v", key1.Type)
	}
	if key2.Type != tea.KeyCtrlS {
		t.Errorf("expected key2.Type == KeyCtrlS, got %v", key2.Type)
	}
}

func Test_sendCmds_produces_rune_keys_on_non_keytypes(t *testing.T) {
	cmds := sendCmds([]string{"hi"})
	if len(cmds) != 2 {
		t.Fatalf("expected 2 cmds, got %d", len(cmds))
	}
	msg1 := cmds[0]()
	msg2 := cmds[1]()
	key1, ok1 := msg1.(tea.KeyMsg)
	key2, ok2 := msg2.(tea.KeyMsg)
	if !ok1 || !ok2 {
		t.Fatalf("expected tea.KeyMsg, got %T and %T", msg1, msg2)
	}
	if key1.Type != tea.KeyRunes || string(key1.Runes) != "h" {
		t.Errorf("expected key1 to be rune 'h', got type %v runes %q", key1.Type, string(key1.Runes))
	}
	if key2.Type != tea.KeyRunes || string(key2.Runes) != "i" {
		t.Errorf("expected key2 to be rune 'i', got type %v runes %q", key2.Type, string(key2.Runes))
	}
}

func TestUpdate_h_closes_leader_and_produces_pending_questionmark(t *testing.T) {
	content := `[leader.h]
help = "Help"
send = ["?"]
`
	lm, err := context.LoadLeader(content)
	if err != nil {
		t.Fatalf("LoadLeader failed: %v", err)
	}
	ctx := &context.MainContext{
		Leader: lm,
	}
	model := New(ctx)
	_ = model.Update(initMsg{})
	if len(model.shown) == 0 {
		t.Fatal("expected shown leader keys")
	}
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h'},
	}
	cmd := model.Update(msg)
	if len(model.shown) != 0 {
		t.Fatal("expected not to show leader keys after reaching leaf")
	}
	if cmd == nil {
		t.Fatal("expected a command returned from Update")
	}
	msgOut := cmd()
	batch, ok := msgOut.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected BatchMsg, got %T", msgOut)
	}
	if len(batch) != 2 {
		t.Fatalf("expected 2 cmds in batch, got %d", len(batch))
	}
	closeMsg := batch[0]()
	if _, ok := closeMsg.(common.CloseViewMsg); !ok {
		t.Error("expected non-nil closeMsg")
	}
	// batch[1]() is a tea.sequenceMsg (private). we test it via sendCmds tests.
}

func TestUpdate_gf_shows_git_fetch_submenu_and_returns_nil_cmd(t *testing.T) {
	content := `[leader.g]
help = "Git"

[leader.gf]
context = [ "$change_id" ] # only available if context key is present.

[leader.gff]
help = "Git Fetch"
send = ["gf", "enter"]
`
	lm, err := context.LoadLeader(content)
	if err != nil {
		t.Fatalf("LoadLeader failed: %v", err)
	}
	ctx := &context.MainContext{
		Leader: lm,
	}
	model := New(ctx)
	_ = model.Update(initMsg{})
	// Press 'g' to enter the submenu
	msgG := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'g'},
	}
	cmd := model.Update(msgG)
	if cmd != nil {
		t.Errorf("expected nil cmd when entering submenu, got %v", cmd)
	}
	if len(model.shown) > 0 {
		t.Fatal("did not expect submenu to be shown since it lacks required context")
	}

	// Test that key is available when its context is satified.
	ctx.SelectedItem = context.SelectedRevision{ChangeId: "foo"}
	model = New(ctx)
	_ = model.Update(initMsg{})
	cmd = model.Update(msgG)
	if cmd != nil {
		t.Errorf("expected nil cmd when entering submenu, got %v", cmd)
	}
	if len(model.shown) == 0 {
		t.Fatal("expected submenu to be shown after pressing 'g'")
	}
	// Press 'f' to enter the next submenu
	msgF := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'f'},
	}
	cmd = model.Update(msgF)
	if cmd != nil {
		t.Errorf("expected nil cmd when entering nested submenu, got %v", cmd)
	}
	if len(model.shown) == 0 {
		t.Fatal("expected nested submenu to be shown after pressing 'f'")
	}
}

func TestUpdate_esc_closes_menu(t *testing.T) {
	content := `[leader.h]
help = "Help"
send = ["?"]
`
	lm, err := context.LoadLeader(content)
	if err != nil {
		t.Fatalf("LoadLeader failed: %v", err)
	}
	ctx := &context.MainContext{
		Leader: lm,
	}
	model := New(ctx)
	msg := tea.KeyMsg{
		Type: tea.KeyEsc,
	}
	cmd := model.Update(msg)
	if len(model.shown) != 0 {
		t.Fatal("expected menu to be closed after pressing esc")
	}
	if cmd == nil {
		t.Fatal("expected a command returned from Update")
	}
	msgOut := cmd()
	if _, ok := msgOut.(common.CloseViewMsg); !ok {
		t.Errorf("expected CloseViewMsg, got %T", msgOut)
	}
}
