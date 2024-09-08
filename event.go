package light_flow

import "time"

type FlowInfo interface {
	flowRuntime
}

type ProcInfo interface {
	procRuntime
	Get(key string) (value any, exist bool)
}

type StepInfo interface {
	stepRuntime
	Get(key string) (value any, exist bool)
}

type FlowEvent interface {
	basicEvent
	Flow() FlowInfo
}

type ProcEvent interface {
	basicEvent
	Proc() ProcInfo
}

type StepEvent interface {
	basicEvent
	Step() StepInfo
}

type basicEvent interface {
	EventID() string
	EventName() string
	Stage() eventStage
	Severity() eventSeverity
	Timestamp() time.Time
	Extra(index extraIndex) any
}
