package light_flow

const (
	beginAction     = "begin"
	completedAction = "completed"
)

var (
	stepPersist = &stepPersistor{
		onBegin:    emptyStepFunc,
		onComplete: emptyStepFunc,
	}
	procPersist = &procPersistor{
		onBegin:    emptyProcFunc,
		onComplete: emptyProcFunc,
	}
	flowPersist = &flowPersistor{
		onBegin:    emptyFlowFunc,
		onComplete: emptyFlowFunc,
	}
)

type StepPersistor interface {
	OnBegin(func(Step) error) StepPersistor
	OnComplete(func(Step) error) StepPersistor
}

type ProcPersistor interface {
	OnBegin(func(Process) error) ProcPersistor
	OnComplete(func(Process) error) ProcPersistor
}

type FlowPersistor interface {
	OnBegin(func(WorkFlow) error) FlowPersistor
	OnComplete(func(WorkFlow) error) FlowPersistor
}

type stepPersistor struct {
	onBegin    func(Step)
	onComplete func(Step)
}

type procPersistor struct {
	onBegin    func(Process)
	onComplete func(Process)
}

type flowPersistor struct {
	onBegin    func(WorkFlow)
	onComplete func(WorkFlow)
}

func (sp *stepPersistor) OnBegin(f func(Step) error) StepPersistor {
	sp.onBegin = wrapPersist[Step](beginAction, f)
	return sp
}

func (sp *stepPersistor) OnComplete(f func(Step) error) StepPersistor {
	sp.onComplete = wrapPersist[Step](completedAction, f)
	return sp
}

func (pp *procPersistor) OnBegin(f func(Process) error) ProcPersistor {
	pp.onBegin = wrapPersist[Process](beginAction, f)
	return pp
}

func (pp *procPersistor) OnComplete(f func(Process) error) ProcPersistor {
	pp.onComplete = wrapPersist[Process](completedAction, f)
	return pp
}

func (fp *flowPersistor) OnBegin(f func(WorkFlow) error) FlowPersistor {
	fp.onBegin = wrapPersist[WorkFlow](beginAction, f)
	return fp
}

func (fp *flowPersistor) OnComplete(f func(WorkFlow) error) FlowPersistor {
	fp.onComplete = wrapPersist[WorkFlow](completedAction, f)
	return fp
}

func ConfigureStepPersist() StepPersistor {
	return stepPersist
}

func ConfigureProcPersist() ProcPersistor {
	return procPersist
}

func ConfigureFlowPersist() FlowPersistor {
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
