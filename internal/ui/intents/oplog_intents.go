package intents

//jjui:bind scope=ui.oplog action=move_up set=Delta:-1
//jjui:bind scope=ui.oplog action=move_down set=Delta:1
//jjui:bind scope=ui.oplog action=page_up set=Delta:-1,IsPage:true
//jjui:bind scope=ui.oplog action=page_down set=Delta:1,IsPage:true
type OpLogNavigate struct {
	Delta  int
	IsPage bool
}

func (OpLogNavigate) isIntent() {}

//jjui:bind scope=ui action=open_oplog
type OpLogOpen struct{}

func (OpLogOpen) isIntent() {}

//jjui:bind scope=ui.oplog action=close
type OpLogClose struct{}

func (OpLogClose) isIntent() {}

//jjui:bind scope=ui.oplog action=diff
type OpLogShowDiff struct {
	OperationId string
}

func (OpLogShowDiff) isIntent() {}

//jjui:bind scope=ui.oplog action=restore
type OpLogRestore struct {
	OperationId string
}

func (OpLogRestore) isIntent() {}

//jjui:bind scope=ui.oplog action=revert
type OpLogRevert struct {
	OperationId string
}

func (OpLogRevert) isIntent() {}
