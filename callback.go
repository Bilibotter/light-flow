package light_flow

const (
	stepScope = "Step"
	procScope = "Process"
	flowScope = "WorkFlow"
)

const (
	mustS    = "must"
	nonMustS = "non-must"
)

const (
	panicLog = "[Panic]%s-callback execute panic, scope=%s, stage=%s;\n recover=%v\n%s\n"
	errorLog = "[Error]%s-callback execute error, scope=%s, stage=%s;\n error=%s\n"
)

type FlowCallback interface {
	ProcessCallback
	BeforeFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) FlowCallback
	AfterFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) FlowCallback
}

type ProcessCallback interface {
	StepCallback
	BeforeProcess(must bool, callback func(Process) (keepOn bool, err error)) ProcessCallback
	AfterProcess(must bool, callback func(Process) (keepOn bool, err error)) ProcessCallback
}

type StepCallback interface {
	BeforeStep(must bool, callback func(Step) (keepOn bool, err error)) StepCallback
	AfterStep(must bool, callback func(Step) (keepOn bool, err error)) StepCallback
}

type flowCallback struct {
	procCallback
	beforeFlow *funcChain[WorkFlow]
	afterFlow  *funcChain[WorkFlow]
}

type procCallback struct {
	stepCallback
	beforeProc *funcChain[Process]
	afterProc  *funcChain[Process]
}

type stepCallback struct {
	beforeStep *funcChain[Step]
	afterStep  *funcChain[Step]
}

type funcChain[T runtimeI] struct {
	index  int
	before bool
	scope  string
	stage  string
	chain  []func(info T) (keepOn bool, err error)
}

func (f *flowCallback) BeforeFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) FlowCallback {
	f.beforeFlow.add(must, callback)
	return f
}

func (f *flowCallback) AfterFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) FlowCallback {
	f.afterFlow.add(must, callback)
	return f
}

func (p *procCallback) BeforeProcess(must bool, callback func(Process) (keepOn bool, err error)) ProcessCallback {
	p.beforeProc.add(must, callback)
	return p
}

func (p *procCallback) AfterProcess(must bool, callback func(Process) (keepOn bool, err error)) ProcessCallback {
	p.afterProc.add(must, callback)
	return p
}

func (s *stepCallback) BeforeStep(must bool, callback func(Step) (keepOn bool, err error)) StepCallback {
	s.beforeStep.add(must, callback)
	return s
}

func (s *stepCallback) AfterStep(must bool, callback func(Step) (keepOn bool, err error)) StepCallback {
	s.afterStep.add(must, callback)
	return s
}

func (chain *funcChain[T]) add(must bool, callback func(T) (keepOn bool, err error)) {
	if must && chain.index != len(chain.chain) {
		panic("must callback shouldn't be added before non-must callback")
	}
	chain.chain = append(chain.chain, callback)
	if must {
		chain.index = len(chain.chain)
	}
}

func (chain *funcChain[T]) filter(begin int, info T) (keepOn bool, err PanicError) {
	if len(chain.chain) == 0 {
		return true, nil
	}
	if chain.before && info.Has(skipPreCallback) {
		return true, nil
	}
	var index int
	defer func() {
		r := recover()
		if r == nil && err == nil {
			return
		}
		info.append(CallbackFail)
		if chain.before {
			info.append(Cancel)
		}
		if r != nil {
			logger.Error(panicLog, chain.necessity(index), chain.scope, chain.stage, r, stack())
			err = newPanicError(nil, r, stack())
			keepOn = false
			return
		}
		if err != nil {
			keepOn = false
			logger.Error(errorLog, chain.necessity(index), chain.scope, chain.stage, err.Error())
		}
	}()
	var e error
	for index = begin; index < len(chain.chain); index++ {
		keepOn, e = chain.chain[index](info)
		if keepOn && e == nil {
			continue
		}
		err = newPanicError(e, nil, nil)
		break
	}
	return
}

func (chain *funcChain[T]) necessity(index int) string {
	if index < chain.index {
		return mustS
	}
	return nonMustS
}
