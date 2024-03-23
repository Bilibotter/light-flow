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
	*status
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
		status:    emptyStatus(),
		StepMeta:  meta,
		id:        generateId(),
		processId: rp.id,
		flowId:    rp.flowId,
		waiting:   int64(len(meta.depends)),
		finish:    make(chan bool, 1),
	}

	rp.flowSteps[meta.stepName] = &step

	return &step
}

func (rp *runProcess) Resume() {
	if rp.remove(Pause) {
		rp.pause.Done()
	}
}

func (rp *runProcess) Pause() {
	if rp.add(Pause) {
		rp.pause.Add(1)
	}
}

func (rp *runProcess) Stop() {
	rp.add(Stop)
	rp.add(Failed)
}

func (rp *runProcess) schedule() (future *Future) {
	rp.initialize()
	future = &Future{
		basicInfo: &basicInfo{
			Id:     rp.id,
			Name:   rp.processName,
			status: rp.status,
		},
		status: rp.status,
		finish: &rp.finish,
	}

	// CallbackFail from Flow
	if rp.Has(CallbackFail) {
		rp.add(Cancel)
		return
	}

	rp.add(Running)
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
		step.add(Success)
		return
	}

	if !rp.Normal() {
		step.add(Cancel)
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
		step.add(Timeout)
		step.add(Failed)
	case <-step.finish:
		return
	}
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
		rp.add(Timeout)
		rp.add(Failed)
	case <-finish:
	}

	for _, step := range rp.flowSteps {
		rp.adds(step.status.load())
	}

	if rp.Normal() {
		rp.add(Success)
	}

	rp.procCallback(After)
	rp.finish.Done()
}

func (rp *runProcess) startNextSteps(step *runStep) {
	// all step not execute should cancel while process is timeout
	cancel := !step.Normal() || !rp.Normal()
	for _, waiter := range step.waiters {
		next := rp.flowSteps[waiter.stepName]
		waiting := atomic.AddInt64(&next.waiting, -1)
		if cancel {
			next.add(Cancel)
		}
		if atomic.LoadInt64(&waiting) != 0 {
			continue
		}
		if step.Success() {
			next.add(Running)
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

	retry := 1
	if rp.StepRetry > 0 {
		retry = rp.StepRetry
	}
	if step.StepRetry > 0 {
		retry = step.StepRetry
	}

	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("panic: %v\n\n%s", r, string(debug.Stack()))
			step.add(Panic)
			step.add(Failed)
			step.Err = panicErr
			step.End = time.Now().UTC()
		}
		rp.stepCallback(step, After)
	}()

	for i := 0; i < retry; i++ {
		result, err := step.run(step)
		step.End = time.Now().UTC()
		step.Err = err
		if err != nil {
			step.add(Error)
			step.add(Failed)
			continue
		}
		if result != nil {
			step.setResult(step.stepName, result)
		}
		step.add(Success)
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
		ProcCtx: rp.visibleContext,
		basicInfo: basicInfo{
			status: rp.status,
			Id:     rp.id,
			Name:   rp.processName,
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
			status: step.status,
			Id:     step.id,
			Name:   step.stepName,
		},
		StepCtx:   step.visibleContext,
		ProcessId: step.processId,
		FlowId:    step.flowId,
		Start:     step.Start,
		End:       step.End,
		Err:       step.Err,
	}

	step.infoCache = info
	return info
}
