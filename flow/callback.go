package flow

/*
 *
 *   Copyright (c) 2024 Bilibotter
 *
 *   This software is licensed under the MIT License.
 *   For more details, see the LICENSE file or visit:
 *   https://opensource.org/licenses/MIT
 *
 *   Project: https://github.com/Bilibotter/light-flow
 *
 */

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	stepScope    = "Step"
	procScope    = "Process"
	flowScope    = "WorkFlow"
	defaultScope = "Default"
)

const (
	defaultStage = 1 << iota
	flowStage
	procStage
)

const (
	beforeStage = 1 << (8 + iota)
	afterStage
)

const (
	flowBP = "|||%s"
	procBP = "||%s"
	stepBP = "|%s"
)

const (
	mustS    = "Must"
	nonMustS = "Non-Must"
)

var (
	defaultCallback = buildFlowCallback(defaultScope)
)

type FlowCallback interface {
	ProcessCallback
	BeforeFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) *decorator[WorkFlow]
	AfterFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) *decorator[WorkFlow]
}

type ProcessCallback interface {
	StepCallback
	DisableDefaultCallback()
	BeforeProcess(must bool, callback func(Process) (keepOn bool, err error)) *decorator[Process]
	AfterProcess(must bool, callback func(Process) (keepOn bool, err error)) *decorator[Process]
}

type StepCallback interface {
	BeforeStep(must bool, callback func(Step) (keepOn bool, err error)) *decorator[Step]
	AfterStep(must bool, callback func(Step) (keepOn bool, err error)) *decorator[Step]
}

type Constraint interface {
	If(condition func() bool) Constraint
	When(status ...*StatusEnum) Constraint
	Exclude(status ...*StatusEnum) Constraint
	OnlyFor(name ...string) Constraint
	NotFor(name ...string) Constraint
}

type breakPoint struct {
	Stage   int
	Index   int
	SkipRun bool
	Used    bool
}

type decorator[T proto] struct {
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

type funcChain[T proto] struct {
	Index  int
	Before bool
	Scope  string
	Stage  string
	Stage0 int // used to compare with breakPoint's stage
	Chain  []*decorator[T]
}

func DefaultCallback() FlowCallback {
	return &defaultCallback
}

func ResetDefaultCallback() {
	defaultCallback = buildFlowCallback(defaultScope)
}

func callbackDetails(necessity, location, scope, order string) map[string]string {
	details := map[string]string{
		"Necessity": necessity,
		"Location":  location,
		"Scope":     scope,
		"Order":     order,
	}
	return details
}

func buildFlowCallback(scope string) flowCallback {
	fc := flowCallback{}
	fc.beforeFlow.Scope = scope
	fc.afterFlow.Scope = scope
	fc.beforeFlow.Stage = beforeS + flowScope
	fc.afterFlow.Stage = afterS + flowScope
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
	pc.beforeProc.Stage = beforeS + procScope
	pc.afterProc.Stage = afterS + procScope
	pc.beforeStep.Scope = scope
	pc.beforeProc.Before = true
	pc.afterStep.Scope = scope
	pc.beforeStep.Stage = beforeS + stepScope
	pc.afterStep.Stage = afterS + stepScope
	pc.beforeStep.Before = true
	pc.beforeStep.buildStage0()
	pc.afterStep.buildStage0()
	pc.beforeProc.buildStage0()
	pc.afterProc.buildStage0()
	return pc
}

func (f *flowCallback) DisableDefaultCallback() {
	if f.beforeFlow.Scope == defaultScope {
		panic("default callback can't disable itself")
	}
	f.disableDefault = true
}

func (f *flowCallback) BeforeFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) *decorator[WorkFlow] {
	return f.beforeFlow.add(must, callback)
}

func (f *flowCallback) AfterFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) *decorator[WorkFlow] {
	return f.afterFlow.add(must, callback)
}

func (f *flowCallback) flowFilter(flag uint64, runtime WorkFlow) (runNext bool) {
	switch flag {
	case beforeF:
		return f.beforeFlow.filter(runtime)
	case afterF:
		return f.afterFlow.filter(runtime)
	}
	return
}

func (p *procCallback) BeforeProcess(must bool, callback func(Process) (keepOn bool, err error)) *decorator[Process] {
	return p.beforeProc.add(must, callback)
}

func (p *procCallback) AfterProcess(must bool, callback func(Process) (keepOn bool, err error)) *decorator[Process] {
	return p.afterProc.add(must, callback)
}

func (p *procCallback) DisableDefaultCallback() {
	p.disableDefault = true
}

func (p *procCallback) procFilter(flag uint64, runtime Process) (runNext bool) {
	switch flag {
	case beforeF:
		return p.beforeProc.filter(runtime)
	case afterF:
		return p.afterProc.filter(runtime)
	}
	return
}

func (s *stepCallback) BeforeStep(must bool, callback func(Step) (keepOn bool, err error)) *decorator[Step] {
	return s.beforeStep.add(must, callback)
}

func (s *stepCallback) AfterStep(must bool, callback func(Step) (keepOn bool, err error)) *decorator[Step] {
	return s.afterStep.add(must, callback)
}

func (s *stepCallback) stepFilter(flag uint64, runtime Step) (runNext bool) {
	switch flag {
	case beforeF:
		return s.beforeStep.filter(runtime)
	case afterF:
		return s.afterStep.filter(runtime)
	}
	return
}

func (chain *funcChain[T]) add(must bool, callback func(T) (keepOn bool, err error)) *decorator[T] {
	if must && chain.Index != len(chain.Chain) {
		panic("must callback shouldn't be added before non-must callback")
	}
	chain.Chain = append(chain.Chain, &decorator[T]{callback})
	if must {
		chain.Index = len(chain.Chain)
	}
	return chain.Chain[len(chain.Chain)-1]
}

func (chain *funcChain[T]) filter(runtime T) (runNext bool) {
	runNext = true
	if len(chain.Chain) == 0 {
		return
	}
	var index, begin int
	if runtime.Has(Recovering) {
		bp := chain.loadBreakPoint(runtime)
		if bp != nil {
			if chain.Stage0 < bp.Stage {
				return
			}
			begin = bp.Index
		}
		// Don't execute ProcessCallback and WorkFlowCallback again
		if bp == nil && chain.Before && !strings.HasSuffix(chain.Stage, stepScope) {
			return
		}
	}
	defer func() {
		r := recover()
		if r != nil {
			event := panicEvent(runtime, InCallback, r, stack())
			event.details = callbackDetails(chain.necessity(index), chain.Stage, chain.Scope, strconv.Itoa(index+1))
			dispatcher.send(event)
			// non-must callback panic, ignore it
			if index >= chain.Index {
				return
			}
			runNext = false
			if handler, ok := any(runtime).(errorHandler); ok {
				handler.composeError(callbackStage, fmt.Errorf("panic: %v", r))
			}
		}
		if runNext {
			return
		}
		runtime.append(CallbackFail)
		if chain.Before {
			runtime.append(Cancel)
		}
		if !runtime.isRecoverable() {
			return
		}
		runtime.append(Suspend)
		// If both 'Step' and 'Callback' fail, then use 'Step' as the breakpoint.
		if bp := chain.loadBreakPoint(runtime); bp != nil && !bp.Used {
			return
		}
		point := &breakPoint{
			Stage:   chain.Stage0,
			Index:   index,
			SkipRun: !chain.Before,
		}
		chain.saveBreakPoint(runtime, point)
	}()

	for index = begin; index < len(chain.Chain); index++ {
		keepOn, err := chain.Chain[index].call(runtime)
		// keepOn will be ignored if err is not nil
		if keepOn && err == nil {
			continue
		}
		if err != nil {
			event := errorEvent(runtime, InCallback, err)
			event.details = callbackDetails(chain.necessity(index), chain.Stage, chain.Scope, strconv.Itoa(index+1))
			dispatcher.send(event)
			if index < chain.Index {
				runNext = false
				if handler, ok := any(runtime).(errorHandler); ok {
					handler.composeError(callbackStage, err)
				}
			}
		}
		break
	}
	return
}

func (chain *funcChain[T]) loadBreakPoint(runtime T) (point *breakPoint) {
	var wrap any
	if strings.HasSuffix(chain.Stage, stepScope) {
		wrap, _ = runtime.getInternal(fmt.Sprintf(stepBP, runtime.Name()))
	} else if strings.HasSuffix(chain.Stage, procScope) {
		wrap, _ = runtime.getInternal(fmt.Sprintf(procBP, runtime.Name()))
	} else if strings.HasSuffix(chain.Stage, flowScope) {
		wrap, _ = runtime.getInternal(fmt.Sprintf(flowBP, runtime.Name()))
	}
	if wrap != nil {
		point = wrap.(*breakPoint)
	}
	return
}

func (chain *funcChain[T]) saveBreakPoint(runtime T, point *breakPoint) {
	// Replay the post-callback of flow and process and recover the post-callback of step
	if !chain.Before && !strings.HasSuffix(chain.Stage, stepScope) {
		point.Index = 0
		point.Stage = defaultStage | afterStage
	}
	if strings.HasSuffix(chain.Stage, stepScope) {
		runtime.setInternal(fmt.Sprintf(stepBP, runtime.Name()), point)
	} else if strings.HasSuffix(chain.Stage, procScope) {
		runtime.setInternal(fmt.Sprintf(procBP, runtime.Name()), point)
	} else if strings.HasSuffix(chain.Stage, flowScope) {
		runtime.setInternal(fmt.Sprintf(flowBP, runtime.Name()), point)
	}
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
		chain.Stage0 |= defaultStage
	case flowScope:
		chain.Stage0 |= flowStage
	case procScope:
		chain.Stage0 |= procStage
	default:
		panic(fmt.Sprintf("unknown scope %s", chain.Scope))
	}
	if chain.Before {
		chain.Stage0 |= beforeStage
	} else {
		chain.Stage0 |= afterStage
	}
}

func (dec *decorator[T]) If(condition func() bool) Constraint {
	old := dec.call
	f := func(runtime T) (bool, error) {
		if !condition() {
			return true, nil
		}
		return old(runtime)
	}
	dec.call = f
	return dec
}

func (dec *decorator[T]) NotFor(name ...string) Constraint {
	s := createSetBySliceFunc(name, func(value string) string { return value })
	old := dec.call
	f := func(runtime T) (bool, error) {
		if s.Contains(runtime.Name()) {
			return true, nil
		}
		return old(runtime)
	}

	dec.call = f
	return dec
}

func (dec *decorator[T]) OnlyFor(name ...string) Constraint {
	s := createSetBySliceFunc(name, func(value string) string { return value })
	old := dec.call
	f := func(runtime T) (bool, error) {
		if !s.Contains(runtime.Name()) {
			return true, nil
		}
		return old(runtime)
	}

	dec.call = f
	return dec
}

func (dec *decorator[T]) When(status ...*StatusEnum) Constraint {
	old := dec.call
	f := func(runtime T) (bool, error) {
		for _, match := range status {
			if runtime.Has(match) {
				return old(runtime)
			}
		}
		return true, nil
	}
	dec.call = f
	return dec
}

func (dec *decorator[T]) Exclude(status ...*StatusEnum) Constraint {
	old := dec.call
	f := func(runtime T) (bool, error) {
		for _, match := range status {
			if runtime.Has(match) {
				return true, nil
			}
		}
		return old(runtime)
	}
	dec.call = f
	return dec
}
