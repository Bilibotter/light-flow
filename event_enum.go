package light_flow

type EventStage uint8

type EventLevel uint8

type EventLayer uint8

/*************************************************************
 * Event Layer
 *************************************************************/
const (
	FlowLayer EventLayer = iota
	ProcLayer
	StepLayer
)

/*************************************************************
 * Event Stage
 *************************************************************/

const (
	InCallback EventStage = iota
	InPersist
	InSuspend
	InRecover
	InResource
	inEvent
)

/*************************************************************
 * Event Level
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
	case FlowLayer:
		return "Flow"
	case ProcLayer:
		return "Process"
	case StepLayer:
		return "Step"
	default:
		return "Unknown"
	}
}

func (s EventStage) String() string {
	switch s {
	case InCallback:
		return "Callback"
	case InPersist:
		return "Persist"
	case InSuspend:
		return "Suspend"
	case InRecover:
		return "Recover"
	case InResource:
		return "Resource"
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
