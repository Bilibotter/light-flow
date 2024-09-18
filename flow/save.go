package flow

const (
	insertOP = "Insert"
	updateOP = "Update"
)

var (
	stepPersist = &stepPersistence{
		onInsert: emptyStepFunc,
		onUpdate: emptyStepFunc,
	}
	procPersist = &procPersistence{
		onInsert: emptyProcFunc,
		onUpdate: emptyProcFunc,
	}
	flowPersist = &flowPersistence{
		onInsert: emptyFlowFunc,
		onUpdate: emptyFlowFunc,
	}
)

type StepPersistence interface {
	OnInsert(func(Step) error) StepPersistence
	OnUpdate(func(Step) error) StepPersistence
}

type ProcPersistence interface {
	OnInsert(func(Process) error) ProcPersistence
	OnUpdate(func(Process) error) ProcPersistence
}

type FlowPersistence interface {
	OnInsert(func(WorkFlow) error) FlowPersistence
	OnUpdate(func(WorkFlow) error) FlowPersistence
}

type stepPersistence struct {
	onInsert func(Step)
	onUpdate func(Step)
}

type procPersistence struct {
	onInsert func(Process)
	onUpdate func(Process)
}

type flowPersistence struct {
	onInsert func(WorkFlow)
	onUpdate func(WorkFlow)
}

func (sp *stepPersistence) OnInsert(f func(Step) error) StepPersistence {
	sp.onInsert = wrapPersist[Step](insertOP, f)
	return sp
}

func (sp *stepPersistence) OnUpdate(f func(Step) error) StepPersistence {
	sp.onUpdate = wrapPersist[Step](updateOP, f)
	return sp
}

func (pp *procPersistence) OnInsert(f func(Process) error) ProcPersistence {
	pp.onInsert = wrapPersist[Process](insertOP, f)
	return pp
}

func (pp *procPersistence) OnUpdate(f func(Process) error) ProcPersistence {
	pp.onUpdate = wrapPersist[Process](updateOP, f)
	return pp
}

func (fp *flowPersistence) OnInsert(f func(WorkFlow) error) FlowPersistence {
	fp.onInsert = wrapPersist[WorkFlow](insertOP, f)
	return fp
}

func (fp *flowPersistence) OnUpdate(f func(WorkFlow) error) FlowPersistence {
	fp.onUpdate = wrapPersist[WorkFlow](updateOP, f)
	return fp
}

func StepPersist() StepPersistence {
	return stepPersist
}

func ProcPersist() ProcPersistence {
	return procPersist
}

func FlowPersist() FlowPersistence {
	return flowPersist
}

func wrapPersist[p proto](action string, f func(p) error) func(p) {
	return func(foo p) {
		defer func() {
			if r := recover(); r != nil {
				event := panicEvent(foo, InPersist, r, stack())
				event.write("Action", action)
				dispatcher.send(event)
			}
		}()
		if action == insertOP {
			foo.append(Pending)
			// Entity was already inserted before recovery, avoid to insert it again.
			if foo.Has(Recovering) {
				return
			}
		}
		err := f(foo)
		if err != nil {
			event := errorEvent(foo, InPersist, err)
			event.write("Action", action)
			dispatcher.send(event)
		}
	}
}

func emptyStepFunc(_ Step) {} // avoid to judge nil

func emptyProcFunc(_ Process) {} // avoid to judge nil

func emptyFlowFunc(_ WorkFlow) {} // avoid to judge nil
