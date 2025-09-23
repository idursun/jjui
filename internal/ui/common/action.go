package common

type WaitResult int

const (
	WaitResultContinue WaitResult = iota
	WaitResultCancel
)

type WaitChannel chan WaitResult

type InlineDescribeAction struct {
	ChangeId string
}
