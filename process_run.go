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
	id        string
	flowId    string
	flowSteps map[string]*RunStep
	pcsScope  map[string]*Context
	pause     sync.WaitGroup
	needRun   sync.WaitGroup
	finish    int64
	status    int64
}

func (rp *RunProcess) addStep(meta *StepMeta) *RunStep {
	step := RunStep{
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
	if PopStatus(&rp.status, Pause) {
		rp.pause.Done()
	}
}

func (rp *RunProcess) Pause() {
	if AppendStatus(&rp.status, Pause) {
		rp.pause.Add(1)
	}
}

func (rp *RunProcess) Stop() {
	AppendStatus(&rp.status, Stop)
}

func (rp *RunProcess) flow() *Feature {
	AppendStatus(&rp.status, Running)
	rp.beforeProc()
	for _, step := range rp.flowSteps {
		if step.layer == 1 {
			go rp.scheduleStep(step)
		}
	}

	go rp.finalize()

	feature := Feature{
		status:  &rp.status,
		finish:  &rp.finish,
		running: &rp.needRun,
	}

	return &feature
}

func (rp *RunProcess) scheduleStep(step *RunStep) {
	defer rp.scheduleNextSteps(step)

	if _, finish := step.GetStepResult(step.stepName); finish {
		AppendStatus(&step.status, Success)
		return
	}

	status := atomic.LoadInt64(&rp.status)
	if !IsStatusNormal(status) {
		AppendStatus(&step.status, Cancel)
		return
	}

	// If prev step is abnormal, the current step will mark as cancel
	// But other tasks that do not depend on the failed task will continue to execute
	if status = atomic.LoadInt64(&step.status); !IsStatusNormal(status) {
		return
	}

	for status = atomic.LoadInt64(&rp.status); status&Pause == Pause; {
		rp.pause.Wait()
		status = atomic.LoadInt64(&rp.status)
	}

	go rp.runStep(step)

	timeout := 3 * time.Hour
	if rp.conf != nil && rp.conf.StepTimeout != 0 {
		timeout = rp.conf.StepTimeout
	}
	if step.config != nil && step.config.Timeout != 0 {
		timeout = step.config.Timeout
	}

	timer := time.NewTimer(timeout)
	select {
	case <-timer.C:
		AppendStatus(&step.status, Timeout)
	case <-step.finish:
		return
	}
}

func (rp *RunProcess) finalize() {
	finish := make(chan bool, 1)
	go func() {
		rp.needRun.Wait()
		rp.afterProc()
		finish <- true
	}()

	timeout := 3 * time.Hour
	if rp.conf != nil && rp.conf.ProcTimeout != 0 {
		timeout = rp.conf.ProcTimeout
	}

	timer := time.NewTimer(timeout)
	select {
	case <-timer.C:
		AppendStatus(&rp.status, Timeout)
	case <-finish:
	}

	for _, step := range rp.flowSteps {
		AppendStatus(&rp.status, step.status)
	}
	if IsStatusNormal(rp.status) {
		AppendStatus(&rp.status, Success)
	}
	atomic.StoreInt64(&rp.finish, 1)
}

func (rp *RunProcess) scheduleNextSteps(step *RunStep) {
	// all step not execute should cancel while process is timeout
	cancel := !IsStatusNormal(step.status) || !IsStatusNormal(rp.status)
	for _, waiter := range step.waiters {
		next := rp.flowSteps[waiter.stepName]
		current := atomic.AddInt64(&next.waiting, -1)
		if cancel {
			AppendStatus(&next.status, Cancel)
		}
		if atomic.LoadInt64(&current) != 0 {
			continue
		}
		if step.status&Success == Success {
			AppendStatus(&next.status, Running)
		}
		go rp.scheduleStep(next)
	}

	rp.needRun.Done()
}

func (rp *RunProcess) runStep(step *RunStep) {
	defer func() { step.finish <- true }()

	step.Start = time.Now().UTC()
	rp.beforeStep(step)
	// if beforeStep panic occur, the step will mark as panic
	if !IsStatusNormal(step.status) {
		return
	}

	retry := 1
	if rp.conf != nil && rp.conf.StepRetry > 0 {
		retry = rp.conf.StepRetry
	}
	if step.config != nil && step.config.MaxRetry > 0 {
		retry = step.config.MaxRetry
	}

	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("panic: %v\n\n%s", r, string(debug.Stack()))
			AppendStatus(&step.status, Panic)
			step.Err = panicErr
			step.End = time.Now().UTC()
		}
		rp.afterStep(step)
	}()

	for i := 0; i < retry; i++ {
		result, err := step.run(step.Context)
		step.End = time.Now().UTC()
		step.Err = err
		step.Set(step.stepName, result)
		if err != nil {
			AppendStatus(&step.status, Error)
			continue
		}
		rp.setStepResult(step.stepName, result)
		AppendStatus(&step.status, Success)
		break
	}
}

func (rp *RunProcess) beforeStep(step *RunStep) {
	rp.stepCallback(step, Before)
}

func (rp *RunProcess) afterStep(step *RunStep) {
	rp.stepCallback(step, After)
}

func (rp *RunProcess) stepCallback(step *RunStep, flag string) {
	if rp.conf == nil || rp.conf.stepChain == nil {
		return
	}
	info := rp.summaryStepInfo(step)
	if panicStack := rp.conf.stepChain.process(flag, info); len(panicStack) != 0 {
		step.Err = fmt.Errorf(panicStack)
	}
}

func (rp *RunProcess) beforeProc() {
	rp.procCallback(Before)
}

func (rp *RunProcess) afterProc() {
	rp.procCallback(After)
}

func (rp *RunProcess) procCallback(flag string) {
	if rp.conf == nil || rp.conf.procChain == nil {
		return
	}

	info := &ProcessInfo{
		BasicInfo: &BasicInfo{
			Id:     rp.id,
			Name:   rp.processName,
			status: &rp.status,
		},
		FlowId: rp.flowId,
		Ctx:    rp.Context,
	}

	rp.conf.procChain.process(flag, info)
}

func (rp *RunProcess) summaryStepInfo(step *RunStep) *StepInfo {
	info := &StepInfo{
		BasicInfo: &BasicInfo{
			Id:     step.id,
			Name:   step.stepName,
			status: &step.status,
		},
		ProcessId: step.processId,
		FlowId:    step.flowId,
		Ctx:       step.Context,
		Config:    step.config,
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
