package light_flow

type EventStage uint8

type EventLevel uint8

type EventLayer uint8

/*************************************************************
 * Event Scope
 *************************************************************/
const (
	FlowLyr EventLayer = iota
	ProcLyr
	StepLyr
)

/*************************************************************
 * Event Stage
 *************************************************************/

const (
	InCondition EventStage = iota
	InCallback
	InPersist
	InSuspend
	InRecover
	InResource
)

/*************************************************************
 * Event Severity
 *************************************************************/

const (
	HintLevel EventLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	PanicLevel
)

func (s EventLayer) String() string {
	switch s {
	case FlowLyr:
		return "Flow"
	case ProcLyr:
		return "Process"
	case StepLyr:
		return "Step"
	default:
		return "Unknown"
	}
}

func (s EventStage) String() string {
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

func (s EventLevel) String() string {
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
