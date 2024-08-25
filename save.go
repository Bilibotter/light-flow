package light_flow

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
	sp.onBegin = wrapPersist[Step]("initialize", f)
	return sp
}

func (sp *stepPersistor) OnComplete(f func(Step) error) StepPersistor {
	sp.onComplete = wrapPersist[Step]("finished", f)
	return sp
}

func (pp *procPersistor) OnBegin(f func(Process) error) ProcPersistor {
	pp.onBegin = wrapPersist[Process]("initialize", f)
	return pp
}

func (pp *procPersistor) OnComplete(f func(Process) error) ProcPersistor {
	pp.onComplete = wrapPersist[Process]("finished", f)
	return pp
}

func (fp *flowPersistor) OnBegin(f func(WorkFlow) error) FlowPersistor {
	fp.onBegin = wrapPersist[WorkFlow]("initialize", f)
	return fp
}

func (fp *flowPersistor) OnComplete(f func(WorkFlow) error) FlowPersistor {
	fp.onComplete = wrapPersist[WorkFlow]("finished", f)
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

func emptyStepFunc(_ Step) {}

func emptyProcFunc(_ Process) {}

func emptyFlowFunc(_ WorkFlow) {}
