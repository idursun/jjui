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
	model := New(lm)
	if len(model.shown) == 0 {
		t.Fatal("expected shown leader keys")
	}
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h'},
	}
	model, cmd := model.Update(msg)
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
	pendingMsgRaw := batch[1]()
	pendingMsg, ok := pendingMsgRaw.(PendingMsg)
	if !ok {
		t.Fatalf("expected PendingMsg, got %T", pendingMsgRaw)
	}
	if len(pendingMsg.cmds) != 1 {
		t.Fatalf("expected 1 cmd in PendingMsg, got %d", len(pendingMsg.cmds))
	}
	msg2 := pendingMsg.cmds[0]()
	keyMsg, ok := msg2.(tea.KeyMsg)
	if !ok {
		t.Fatalf("expected tea.KeyMsg, got %T", msg2)
	}
	if keyMsg.Type != tea.KeyRunes || string(keyMsg.Runes) != "?" {
		t.Errorf("expected keyMsg to be rune '?', got type %v runes %q", keyMsg.Type, string(keyMsg.Runes))
	}
}

func TestUpdate_gf_shows_git_fetch_submenu_and_returns_nil_cmd(t *testing.T) {
	content := `[leader.g]
help = "Git"

[leader.gff]
help = "Git Fetch"
send = ["gf", "enter"]
`
	lm, err := context.LoadLeader(content)
	if err != nil {
		t.Fatalf("LoadLeader failed: %v", err)
	}
	model := New(lm)
	// Press 'g' to enter the submenu
	msgG := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'g'},
	}
	model, cmd := model.Update(msgG)
	if cmd != nil {
		t.Errorf("expected nil cmd when entering submenu, got %v", cmd)
	}
	if model.shown == nil || len(model.shown) == 0 {
		t.Fatal("expected submenu to be shown after pressing 'g'")
	}
	// Press 'f' to enter the next submenu
	msgF := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'f'},
	}
	model, cmd = model.Update(msgF)
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
	model := New(lm)
	msg := tea.KeyMsg{
		Type: tea.KeyEsc,
	}
	model, cmd := model.Update(msg)
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

func TestTakePending_reduces_cmds_one_by_one(t *testing.T) {
	cmds := sendCmds([]string{"ctrl+h", "b", "c"})
	pending := PendingMsg{cmds: cmds}
	if len(pending.cmds) != 3 {
		t.Fatalf("Expected 3 cmds")
	}

	cmd := TakePending(pending)
	if batch, ok := cmd().(tea.BatchMsg); !ok {
		t.Fatalf("expected BatchMsg got %v", batch)
	} else if len(batch) != 2 {
		t.Fatalf("expected 2 cmds in batch, got %d", len(batch))
	} else if k, ok := batch[0]().(tea.KeyMsg); !ok || k.String() != "ctrl+h" {
		t.Fatalf("expected rune got %v", k)
	} else if pending, ok = batch[1]().(PendingMsg); !ok || len(pending.cmds) != 2 {
		t.Fatalf("expected two pending cmds got %v", pending)
	}

	cmds = sendCmds([]string{"c"})
	pending = PendingMsg{cmds: cmds}
	cmd = TakePending(pending)
	if cmd().(tea.KeyMsg).String() != "c" {
		t.Fatalf("expected single command to be returned")
	}

	cmds = sendCmds([]string{})
	pending = PendingMsg{cmds: cmds}
	cmd = TakePending(pending)
	if cmd != nil {
		t.Fatalf("expected nil command for empty send keys")
	}
}
