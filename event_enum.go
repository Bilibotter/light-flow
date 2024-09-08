package light_flow

type eventStage int

type eventSeverity int

type extraIndex int

const (
	InCondition eventStage = iota
	InCallback
	InPersist
	InSuspend
	InRecover
)

const (
	HintLevel eventSeverity = iota
	WarnLevel
	ErrorLevel
	PanicLevel
)
