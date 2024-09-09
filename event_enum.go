package light_flow

type eventType uint8

type eventStage uint8

type eventSeverity uint8

type eventScope uint8

type extraIndex uint8

const (
	FlowScp eventScope = iota
	ProcScp
	StepScp
)

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
