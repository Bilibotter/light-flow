package light_flow

type eventType uint8

type eventStage uint8

type eventLevel uint8

type eventScope uint8

/*************************************************************
 * Event Scope
 *************************************************************/
const (
	FlowScp eventScope = iota
	ProcScp
	StepScp
)

/*************************************************************
 * Event Stage
 *************************************************************/

const (
	InCondition eventStage = iota
	InCallback
	InPersist
	InSuspend
	InRecover
)

/*************************************************************
 * Event Severity
 *************************************************************/

const (
	HintLevel eventLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	PanicLevel
)

func (s eventScope) String() string {
	switch s {
	case FlowScp:
		return "Flow"
	case ProcScp:
		return "Process"
	case StepScp:
		return "Step"
	default:
		return "Unknown"
	}
}

func (s eventStage) String() string {
	switch s {
	case InCondition:
		return "Condition"
	case InCallback:
		return "Callback"
	case InPersist:
		return "Persist"
	case InSuspend:
		return "Suspend"
	case InRecover:
		return "Recover"
	default:
		return "Unknown"
	}
}

func (s eventLevel) String() string {
	switch s {
	case HintLevel:
		return "Hint"
	case InfoLevel:
		return "Info"
	case WarnLevel:
		return "Warning"
	case ErrorLevel:
		return "Error"
	case PanicLevel:
		return "Panic"
	default:
		return "Unknown"
	}
}
