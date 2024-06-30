package light_flow

import (
	"fmt"
)

const (
	stepScope    = "Step"
	procScope    = "Process"
	flowScope    = "WorkFlow"
	defaultScope = "Default"
)

const (
	mustS    = "Must"
	nonMustS = "Non-Must"
)

const (
	panicLog = "%s-Callback execute panic;\nScope=%s, Stage=%s, Panic=%v\n%s\n"
	errorLog = "%s-Callback execute error;\nScope=%s, Stage=%s, Error=%s\n"
)

var (
	defaultCallback = buildFlowCallback(defaultScope)
)

type FlowCallback interface {
	ProcessCallback
	BeforeFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) decorator[WorkFlow]
	AfterFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) decorator[WorkFlow]
}

type ProcessCallback interface {
	StepCallback
	DisableDefaultCallback()
	BeforeProcess(must bool, callback func(Process) (keepOn bool, err error)) decorator[Process]
	AfterProcess(must bool, callback func(Process) (keepOn bool, err error)) decorator[Process]
}

type StepCallback interface {
	BeforeStep(must bool, callback func(Step) (keepOn bool, err error)) decorator[Step]
	AfterStep(must bool, callback func(Step) (keepOn bool, err error)) decorator[Step]
}

type Decorator interface {
	If(condition func() bool) Decorator
	When(status ...*StatusEnum) Decorator
	Exclude(status ...*StatusEnum) Decorator
	OnlyFor(name ...string) Decorator
	NotFor(name ...string) Decorator
}

type breakPoint struct {
	Stage int
	Index int
}

type decorator[T runtimeI] struct {
	call func(info T) (keepOn bool, err error)
}

type flowCallback struct {
	procCallback
	beforeFlow funcChain[WorkFlow]
	afterFlow  funcChain[WorkFlow]
}

type procCallback struct {
	stepCallback
	beforeProc     funcChain[Process]
	afterProc      funcChain[Process]
	disableDefault bool
}

type stepCallback struct {
	beforeStep funcChain[Step]
	afterStep  funcChain[Step]
}

type funcChain[T runtimeI] struct {
	Index  int
	Before bool
	Scope  string
	Stage  string
	Stage0 int // used to compare with breakPoint's stage
	Chain  []decorator[T]
}

func buildFlowCallback(scope string) flowCallback {
	fc := flowCallback{}
	fc.beforeFlow.Scope = scope
	fc.afterFlow.Scope = scope
	fc.beforeFlow.Stage = Before + "-" + flowScope
	fc.afterFlow.Stage = After + "-" + flowScope
	fc.beforeFlow.Before = true
	fc.procCallback = buildProcCallback(scope)
	fc.beforeFlow.buildStage0()
	fc.afterFlow.buildStage0()
	return fc
}

func buildProcCallback(scope string) procCallback {
	pc := procCallback{}
	pc.beforeProc.Scope = scope
	pc.afterProc.Scope = scope
	pc.beforeProc.Stage = Before + "-" + procScope
	pc.afterProc.Stage = After + "-" + procScope
	pc.beforeStep.Scope = scope
	pc.beforeProc.Before = true
	pc.afterStep.Scope = scope
	pc.beforeStep.Stage = Before + "-" + stepScope
	pc.afterStep.Stage = After + "-" + stepScope
	pc.beforeStep.Before = true
	pc.beforeStep.buildStage0()
	pc.afterStep.buildStage0()
	pc.beforeProc.buildStage0()
	pc.afterProc.buildStage0()
	return pc
}

func (f *flowCallback) BeforeFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) decorator[WorkFlow] {
	return f.beforeFlow.add(must, callback)
}

func (f *flowCallback) AfterFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) decorator[WorkFlow] {
	return f.afterFlow.add(must, callback)
}

func (f *flowCallback) flowFilter(flag string, lastTime *breakPoint, info WorkFlow) (breakOff bool, point *breakPoint) {
	switch flag {
	case Before:
		return f.beforeFlow.filter(lastTime, info)
	case After:
		return f.afterFlow.filter(lastTime, info)
	}
	return false, nil
}

func (p *procCallback) BeforeProcess(must bool, callback func(Process) (keepOn bool, err error)) decorator[Process] {
	return p.beforeProc.add(must, callback)
}

func (p *procCallback) AfterProcess(must bool, callback func(Process) (keepOn bool, err error)) decorator[Process] {
	return p.afterProc.add(must, callback)
}

func (p *procCallback) DisableDefaultCallback() {
	p.disableDefault = true
}

func (p *procCallback) procFilter(flag string, lastTime *breakPoint, info Process) (breakOff bool, point *breakPoint) {
	switch flag {
	case Before:
		return p.beforeProc.filter(lastTime, info)
	case After:
		return p.afterProc.filter(lastTime, info)
	}
	return false, nil
}

func (s *stepCallback) BeforeStep(must bool, callback func(Step) (keepOn bool, err error)) decorator[Step] {
	return s.beforeStep.add(must, callback)
}

func (s *stepCallback) AfterStep(must bool, callback func(Step) (keepOn bool, err error)) decorator[Step] {
	return s.afterStep.add(must, callback)
}

func (s *stepCallback) stepFilter(flag string, lastTime *breakPoint, info Step) (breakOff bool, point *breakPoint) {
	switch flag {
	case Before:
		return s.beforeStep.filter(lastTime, info)
	case After:
		return s.afterStep.filter(lastTime, info)
	}
	return false, nil
}

func (chain *funcChain[T]) add(must bool, callback func(T) (keepOn bool, err error)) decorator[T] {
	if must && chain.Index != len(chain.Chain) {
		panic("must callback shouldn't be added before non-must callback")
	}
	chain.Chain = append(chain.Chain, decorator[T]{callback})
	if must {
		chain.Index = len(chain.Chain)
	}
	return chain.Chain[len(chain.Chain)-1]
}

func (chain *funcChain[T]) filter(lastTime *breakPoint, info T) (breakOff bool, point *breakPoint) {
	if len(chain.Chain) == 0 {
		return
	}
	if lastTime != nil && chain.Stage0 < lastTime.Stage {
		return
	}
	var index, begin int
	var keepOn bool
	var err error
	defer func() {
		r := recover()
		if r == nil && err == nil {
			return
		}
		breakOff = index < chain.Index
		if breakOff {
			point = &breakPoint{
				Stage: chain.Stage0,
				Index: index,
			}
			info.append(CallbackFail)
			if chain.Before {
				info.append(Cancel)
			}
		}
		if r != nil {
			logger.Errorf(panicLog, chain.necessity(index), chain.Scope, chain.Stage, r, stack())
			return
		}
		if err != nil {
			logger.Errorf(errorLog, chain.necessity(index), chain.Scope, chain.Stage, err.Error())
		}
	}()
	if lastTime != nil {
		begin = lastTime.Index
	}
	for index = begin; index < len(chain.Chain); index++ {
		keepOn, err = chain.Chain[index].call(info)
		if keepOn && err == nil {
			continue
		}
		break
	}
	return
}

func (chain *funcChain[T]) necessity(index int) string {
	if index < chain.Index {
		return mustS
	}
	return nonMustS
}

func (chain *funcChain[T]) buildStage0() {
	chain.Stage0 = 0
	switch chain.Scope {
	case defaultScope:
		chain.Stage0 |= 1 << 0
	case flowScope:
		chain.Stage0 |= 1 << 1
	case procScope:
		chain.Stage0 |= 1 << 2
	default:
		panic(fmt.Sprintf("unknown scope %s", chain.Scope))
	}
	if chain.Before {
		chain.Stage0 |= 1 << 8
	} else {
		chain.Stage0 |= 1 << 9
	}
}

func (dec decorator[T]) If(condition func() bool) Decorator {
	old := dec.call
	f := func(info T) (bool, error) {
		if !condition() {
			return true, nil
		}
		return old(info)
	}
	dec.call = f
	return dec
}

func (dec decorator[T]) NotFor(name ...string) Decorator {
	s := createSetBySliceFunc(name, func(value string) string { return value })
	old := dec.call
	f := func(info T) (bool, error) {
		if s.Contains(info.Name()) {
			return true, nil
		}
		return old(info)
	}

	dec.call = f
	return dec
}

func (dec decorator[T]) OnlyFor(name ...string) Decorator {
	s := createSetBySliceFunc(name, func(value string) string { return value })
	old := dec.call
	f := func(info T) (bool, error) {
		if !s.Contains(info.Name()) {
			return true, nil
		}
		return old(info)
	}

	dec.call = f
	return dec
}

func (dec decorator[T]) When(status ...*StatusEnum) Decorator {
	old := dec.call
	f := func(info T) (bool, error) {
		for _, match := range status {
			if info.Has(match) {
				return old(info)
			}
		}
		return true, nil
	}
	dec.call = f
	return dec
}

func (dec decorator[T]) Exclude(status ...*StatusEnum) Decorator {
	old := dec.call
	f := func(info T) (bool, error) {
		for _, match := range status {
			if info.Has(match) {
				return true, nil
			}
		}
		return old(info)
	}
	dec.call = f
	return dec
}
