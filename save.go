package light_flow

const (
	beginAction     = "begin"
	completedAction = "completed"
)

var (
	stepPersist = &stepPersistor{
		onInsert: emptyStepFunc,
		onUpdate: emptyStepFunc,
	}
	procPersist = &procPersistor{
		onInsert: emptyProcFunc,
		onUpdate: emptyProcFunc,
	}
	flowPersist = &flowPersistor{
		onInsert: emptyFlowFunc,
		onUpdate: emptyFlowFunc,
	}
)

type StepPersistor interface {
	OnInsert(func(Step) error) StepPersistor
	OnUpdate(func(Step) error) StepPersistor
}

type ProcPersistor interface {
	OnInsert(func(Process) error) ProcPersistor
	OnUpdate(func(Process) error) ProcPersistor
}

type FlowPersistor interface {
	OnInsert(func(WorkFlow) error) FlowPersistor
	OnUpdate(func(WorkFlow) error) FlowPersistor
}

type stepPersistor struct {
	onInsert func(Step)
	onUpdate func(Step)
}

type procPersistor struct {
	onInsert func(Process)
	onUpdate func(Process)
}

type flowPersistor struct {
	onInsert func(WorkFlow)
	onUpdate func(WorkFlow)
}

func (sp *stepPersistor) OnInsert(f func(Step) error) StepPersistor {
	sp.onInsert = wrapPersist[Step](beginAction, f)
	return sp
}

func (sp *stepPersistor) OnUpdate(f func(Step) error) StepPersistor {
	sp.onUpdate = wrapPersist[Step](completedAction, f)
	return sp
}

func (pp *procPersistor) OnInsert(f func(Process) error) ProcPersistor {
	pp.onInsert = wrapPersist[Process](beginAction, f)
	return pp
}

func (pp *procPersistor) OnUpdate(f func(Process) error) ProcPersistor {
	pp.onUpdate = wrapPersist[Process](completedAction, f)
	return pp
}

func (fp *flowPersistor) OnInsert(f func(WorkFlow) error) FlowPersistor {
	fp.onInsert = wrapPersist[WorkFlow](beginAction, f)
	return fp
}

func (fp *flowPersistor) OnUpdate(f func(WorkFlow) error) FlowPersistor {
	fp.onUpdate = wrapPersist[WorkFlow](completedAction, f)
	return fp
}

func StepPersist() StepPersistor {
	return stepPersist
}

func ProcPersist() ProcPersistor {
	return procPersist
}

func FlowPersist() FlowPersistor {
	return flowPersist
}

func wrapPersist[p proto](action string, f func(p) error) func(p) {
	return func(foo p) {
		defer func() {
			if r := recover(); r != nil {
				var t string
				switch any(foo).(type) {
				case Step:
					t = "Step"
				case Process:
					t = "Process"
				case WorkFlow:
					t = "WorkFlow"
				}
				logger.Errorf(persistPanicLog, t, foo.Name(), foo.ID(), action, r, stack())
			}
		}()
		if action == beginAction {
			foo.append(Pending)
			// Entity was already inserted before recovery, avoid to insert it again.
			if foo.Has(Recovering) {
				return
			}
		}
		err := f(foo)
		if err != nil {
			var t string
			switch any(foo).(type) {
			case Step:
				t = "Step"
			case Process:
				t = "Process"
			case WorkFlow:
				t = "WorkFlow"
			}
			logger.Errorf(persistErrorLog, t, foo.Name(), foo.ID(), action, err.Error())
		}
	}
}

func emptyStepFunc(_ Step) {} // avoid to judge nil

func emptyProcFunc(_ Process) {} // avoid to judge nil

func emptyFlowFunc(_ WorkFlow) {} // avoid to judge nil
