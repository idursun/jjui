package intents

//jjui:bind scope=ui action=open_command_history
type CommandHistoryToggle struct{}

func (CommandHistoryToggle) isIntent() {}

//jjui:bind scope=command_history action=move_up set=Delta:-1
//jjui:bind scope=command_history action=move_down set=Delta:1
type CommandHistoryNavigate struct{ Delta int }

func (CommandHistoryNavigate) isIntent() {}

//jjui:bind scope=command_history action=close
type CommandHistoryClose struct{}

func (CommandHistoryClose) isIntent() {}

//jjui:bind scope=command_history action=delete_selected
type CommandHistoryDeleteSelected struct{}

func (CommandHistoryDeleteSelected) isIntent() {}

type AddMessage struct {
	Text   string
	Err    error
	Sticky bool
}

func (AddMessage) isIntent() {}

type DismissOldest struct{}

func (DismissOldest) isIntent() {}
