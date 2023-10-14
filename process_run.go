package light_flow

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type RunProcess struct {
	*ProcessMeta
	*Context
	*Status
	id        string
	flowId    string
	flowSteps map[string]*RunStep
	pcsScope  map[string]*Context
	pause     sync.WaitGroup
	running   sync.WaitGroup
	finish    sync.WaitGroup
}

func (rp *RunProcess) buildRunStep(meta *StepMeta) *RunStep {
	step := RunStep{
		Status:    emptyStatus(),
		StepMeta:  meta,
		id:        generateId(),
		processId: rp.id,
		flowId:    rp.flowId,
		waiting:   int64(len(meta.depends)),
		finish:    make(chan bool, 1),
		Context: &Context{
			table:     sync.Map{},
			name:      meta.stepName,
			scopes:    []string{ProcessCtx},
			scopeCtxs: rp.pcsScope,
			priority:  meta.ctxPriority,
		},
	}

	rp.flowSteps[meta.stepName] = &step
	rp.pcsScope[meta.stepName] = step.Context

	for _, depend := range meta.depends {
		parent := step.scopeCtxs[depend.stepName]
		step.parents = append(step.parents, parent)
	}
	step.parents = append(step.parents, rp.Context)
	return &step
}

func (rp *RunProcess) Resume() {
	if rp.Pop(Pause) {
		rp.pause.Done()
	}
}

func (rp *RunProcess) Pause() {
	if rp.Append(Pause) {
		rp.pause.Add(1)
	}
}

func (rp *RunProcess) Stop() {
	rp.Append(Stop)
	rp.Append(Failed)
}

func (rp *RunProcess) flow() *Feature {
	rp.initialize()
	rp.Append(Running)
	rp.procCallback(Before)
	for _, step := range rp.flowSteps {
		if step.layer == 1 {
			go rp.scheduleStep(step)
		}
	}

	rp.finish.Add(1)
	go rp.finalize()

	feature := Feature{
		BasicInfo: &BasicInfo{
			Id:     rp.id,
			Name:   rp.processName,
			Status: rp.Status,
		},
		Status: rp.Status,
		finish: &rp.finish,
	}

	return &feature
}

func (rp *RunProcess) scheduleStep(step *RunStep) {
	defer rp.scheduleNextSteps(step)

	if _, finish := step.GetStepResult(step.stepName); finish {
		step.Append(Success)
		return
	}

	if !rp.Normal() {
		step.Append(Cancel)
		return
	}

	// If prev step is abnormal, the current step will mark as cancel
	// But other tasks that do not depend on the failed task will continue to execute
	if !step.Normal() {
		return
	}

	for rp.Contain(Pause) {
		rp.pause.Wait()
	}

	timeout := 3 * time.Hour
	if rp.processConfig != nil && rp.StepConfig != nil && rp.StepTimeout != 0 {
		timeout = rp.StepTimeout
	}
	if step.StepConfig != nil && step.StepTimeout != 0 {
		timeout = step.StepTimeout
	}

	timer := time.NewTimer(timeout)
	go rp.runStep(step)
	select {
	case <-timer.C:
		step.Append(Timeout)
		step.Append(Failed)
	case <-step.finish:
		return
	}
}

func (rp *RunProcess) finalize() {
	timeout := 3 * time.Hour
	if rp.processConfig != nil && rp.ProcTimeout != 0 {
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
		rp.Append(Timeout)
		rp.Append(Failed)
	case <-finish:
	}

	for _, step := range rp.flowSteps {
		rp.combine(step.Status)
	}

	if rp.Normal() {
		rp.Append(Success)
	}

	rp.procCallback(After)

	rp.finish.Done()
}

func (rp *RunProcess) scheduleNextSteps(step *RunStep) {
	// all step not execute should cancel while process is timeout
	cancel := !step.Normal() || !rp.Normal()
	for _, waiter := range step.waiters {
		next := rp.flowSteps[waiter.stepName]
		waiting := atomic.AddInt64(&next.waiting, -1)
		if cancel {
			next.Append(Cancel)
		}
		if atomic.LoadInt64(&waiting) != 0 {
			continue
		}
		if step.Success() {
			next.Append(Running)
		}
		go rp.scheduleStep(next)
	}

	rp.running.Done()
}

func (rp *RunProcess) runStep(step *RunStep) {
	defer func() { step.finish <- true }()

	step.Start = time.Now().UTC()
	rp.stepCallback(step, Before)
	// if beforeStep panic occur, the step will mark as panic
	if !step.Normal() {
		return
	}

	retry := 1
	if rp.processConfig != nil && rp.StepConfig != nil && rp.StepRetry > 0 {
		retry = rp.StepRetry
	}
	if step.StepConfig != nil && step.StepRetry > 0 {
		retry = step.StepRetry
	}

	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("panic: %v\n\n%s", r, string(debug.Stack()))
			step.Append(Panic)
			step.Append(Failed)
			step.Err = panicErr
			step.End = time.Now().UTC()
		}
		rp.stepCallback(step, After)
	}()

	for i := 0; i < retry; i++ {
		result, err := step.run(step.Context)
		step.End = time.Now().UTC()
		step.Err = err
		step.Set(step.stepName, result)
		if err != nil {
			step.Append(Error)
			step.Append(Failed)
			continue
		}
		rp.setStepResult(step.stepName, result)
		step.Append(Success)
		break
	}
}

func (rp *RunProcess) stepCallback(step *RunStep, flag string) {
	if rp.processConfig == nil || rp.stepChain == nil {
		return
	}
	info := rp.summaryStepInfo(step)
	if panicStack := rp.stepChain.process(flag, info); step.Err == nil && len(panicStack) != 0 {
		step.Err = fmt.Errorf(panicStack)
	}
}

func (rp *RunProcess) procCallback(flag string) {
	if rp.processConfig == nil || rp.procChain == nil {
		return
	}

	info := &ProcessInfo{
		Context: rp.Context,
		BasicInfo: &BasicInfo{
			Status: rp.Status,
			Id:     rp.id,
			Name:   rp.processName,
		},
		FlowId: rp.flowId,
	}

	rp.procChain.process(flag, info)
}

func (rp *RunProcess) summaryStepInfo(step *RunStep) *StepInfo {
	info := &StepInfo{
		BasicInfo: &BasicInfo{
			Status: step.Status,
			Id:     step.id,
			Name:   step.stepName,
		},
		Context:   step.Context,
		ProcessId: step.processId,
		FlowId:    step.flowId,
		Start:     step.Start,
		End:       step.End,
		Err:       step.Err,
		Prev:      make(map[string]string, len(step.depends)),
		Next:      make(map[string]string, len(step.waiters)),
	}
	for _, prev := range step.depends {
		info.Prev[prev.stepName] = rp.flowSteps[prev.stepName].id
	}

	for _, next := range step.waiters {
		info.Next[next.stepName] = rp.flowSteps[next.stepName].id
	}

	return info
}

func (rp *RunProcess) SkipFinishedStep(name string, result any) {
	rp.setStepResult(name, result)
}
