package light_flow

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type runProcess struct {
	*ProcessMeta
	*state
	*visibleContext
	id        string
	flowId    string
	flowSteps map[string]*runStep
	pause     sync.WaitGroup
	running   sync.WaitGroup
	finish    sync.WaitGroup
}

func (rp *runProcess) buildRunStep(meta *StepMeta) *runStep {
	step := runStep{
		visibleContext: &visibleContext{
			accessInfo:  &meta.accessInfo,
			linkedTable: rp.linkedTable,
			prev:        rp.visibleContext.prev,
		},
		state:     emptyStatus(),
		StepMeta:  meta,
		id:        generateId(),
		processId: rp.id,
		flowId:    rp.flowId,
		segments:  segment(len(meta.depends)),
		finish:    make(chan bool, 1),
	}

	rp.flowSteps[meta.stepName] = &step

	return &step
}

func (rp *runProcess) Resume() {
	if rp.clear(Pause) {
		rp.pause.Done()
	}
}

func (rp *runProcess) Pause() {
	if rp.set(Pause) {
		rp.pause.Add(1)
	}
}

func (rp *runProcess) Stop() {
	rp.set(Stop)
	rp.set(Failed)
}

func (rp *runProcess) schedule() (future *Future) {
	rp.initialize()
	future = &Future{
		basicInfo: &basicInfo{
			Id:    rp.id,
			Name:  rp.processName,
			state: rp.state,
		},
		state:  rp.state,
		finish: &rp.finish,
	}

	// CallbackFail from Flow
	if rp.Has(CallbackFail) {
		rp.set(Cancel)
		return
	}

	rp.set(Running)
	rp.procCallback(Before)
	for _, step := range rp.flowSteps {
		if step.layer == 1 {
			go rp.startStep(step)
		}
	}

	rp.finish.Add(1)
	go rp.finalize()
	return
}

func (rp *runProcess) startStep(step *runStep) {
	defer rp.startNextSteps(step)

	if _, finish := step.GetResult(step.stepName); finish {
		step.set(Success)
		return
	}

	if !rp.Normal() {
		step.set(Cancel)
		return
	}

	// If prev step is abnormal, the current step will mark as cancel
	// But other tasks that do not depend on the failed task will continue to execute
	if !step.Normal() {
		return
	}

	for rp.Has(Pause) {
		rp.pause.Wait()
	}

	if step.Has(skipped) {
		return
	}

	if len(step.depends) > 0 && !rp.evaluateCondition(step) {
		step.set(skipped)
		return
	}

	timeout := 3 * time.Hour
	if rp.StepTimeout != 0 {
		timeout = rp.StepTimeout
	}
	if step.StepTimeout != 0 {
		timeout = step.StepTimeout
	}

	timer := time.NewTimer(timeout)
	go rp.runStep(step)
	select {
	case <-timer.C:
		step.set(Timeout)
		step.set(Failed)
	case <-step.finish:
		return
	}
}

func (rp *runProcess) evaluateCondition(step *runStep) bool {
	for _, group := range step.evalGroups {
		named, unnamed, exist := rp.getCond(group.depend)
		if !exist {
			return false
		}
		if !group.evaluateConditions(named, unnamed) {
			return false
		}
	}
	return true
}

func (rp *runProcess) finalize() {
	timeout := 3 * time.Hour
	if rp.ProcTimeout != 0 {
		timeout = rp.ProcTimeout
	}

	timer := time.NewTimer(timeout)
	finish := make(chan bool, 1)
	go func() {
		rp.running.Wait()
		finish <- true
	}()
	select {
	case <-timer.C:
		rp.set(Timeout)
		rp.set(Failed)
	case <-finish:
	}

	for _, step := range rp.flowSteps {
		rp.add(step.state.load())
	}

	if rp.Normal() {
		rp.set(Success)
	}

	rp.procCallback(After)
	rp.finish.Done()
}

func (rp *runProcess) startNextSteps(step *runStep) {
	// all step not execute should cancel while process is timeout
	cancel := !step.Normal() || !rp.Normal()
	skip := step.Has(skipped)
	for _, waiter := range step.waiters {
		next := rp.flowSteps[waiter.stepName]
		waiting := next.segments.addWaiting(-1)
		if cancel {
			next.set(Cancel)
		}
		if atomic.LoadInt64(&waiting) != 0 {
			continue
		}
		if step.Success() {
			next.set(Running)
		} else if skip {
			next.set(skipped)
		}
		go rp.startStep(next)
	}

	rp.running.Done()
}

func (rp *runProcess) runStep(step *runStep) {
	defer func() { step.finish <- true }()

	step.Start = time.Now().UTC()
	rp.stepCallback(step, Before)
	// if beforeStep panic occur, the step will mark as panic
	if !step.Normal() {
		return
	}

	retry := 0
	if rp.StepRetry > 0 {
		retry = rp.StepRetry
	}
	if step.StepRetry > 0 {
		retry = step.StepRetry
	}

	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("panic: %v\n\n%s\n", r, string(debug.Stack()))
			step.set(Panic)
			step.set(Failed)
			step.Err = panicErr
			step.End = time.Now().UTC()
		}
		rp.stepCallback(step, After)
	}()

	for i := 0; i <= retry; i++ {
		result, err := step.run(step)
		step.End = time.Now().UTC()
		step.Err = err
		if err != nil {
			step.set(Error)
			step.set(Failed)
			continue
		}
		if result != nil {
			step.setResult(step.stepName, result)
		}
		step.set(Success)
		break
	}
}

func (rp *runProcess) stepCallback(step *runStep, flag string) {
	info := rp.summaryStepInfo(step)
	panicStack := ""
	defer func() {
		if len(panicStack) > 0 && step.Err == nil {
			step.Err = fmt.Errorf(panicStack)
		}
	}()
	if !rp.belong.ProcNotUseDefault && !rp.ProcNotUseDefault && defaultConfig != nil {
		panicStack = defaultConfig.stepChain.handle(flag, info)
	}
	if len(panicStack) != 0 {
		return
	}
	panicStack = rp.belong.stepChain.handle(flag, info)
	if len(panicStack) != 0 {
		return
	}
	panicStack = rp.stepChain.handle(flag, info)
}

func (rp *runProcess) procCallback(flag string) {
	info := &Process{
		procCtx: rp.visibleContext,
		basicInfo: basicInfo{
			state: rp.state,
			Id:    rp.id,
			Name:  rp.processName,
		},
		FlowId: rp.flowId,
	}
	if !rp.belong.ProcNotUseDefault && !rp.ProcNotUseDefault && defaultConfig != nil {
		defaultConfig.procChain.handle(flag, info)
	}
	rp.belong.procChain.handle(flag, info)
	rp.procChain.handle(flag, info)
}

func (rp *runProcess) summaryStepInfo(step *runStep) *Step {
	if step.infoCache != nil {
		step.syncInfo()
		return step.infoCache
	}
	info := &Step{
		basicInfo: &basicInfo{
			state: step.state,
			Id:    step.id,
			Name:  step.stepName,
		},
		StepCtx:   step,
		ProcessId: step.processId,
		FlowId:    step.flowId,
		Start:     step.Start,
		End:       step.End,
		Err:       step.Err,
	}

	step.infoCache = info
	return info
}
