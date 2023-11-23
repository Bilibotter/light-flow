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
	*Status
	visibleContext
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
			parent:  &rp.visibleContext,
			visitor: &meta.visitor,
		},
		Status:    emptyStatus(),
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
	if rp.Pop(Pause) {
		rp.pause.Done()
	}
}

func (rp *runProcess) Pause() {
	if rp.Append(Pause) {
		rp.pause.Add(1)
	}
}

func (rp *runProcess) Stop() {
	rp.Append(Stop)
	rp.Append(Failed)
}

func (rp *runProcess) flow() *Future {
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

	future := Future{
		basicInfo: &basicInfo{
			Id:     rp.id,
			Name:   rp.processName,
			Status: rp.Status,
		},
		Status: rp.Status,
		finish: &rp.finish,
	}

	return &future
}

func (rp *runProcess) scheduleStep(step *runStep) {
	defer rp.scheduleNextSteps(step)

	if _, finish := step.GetResult(step.stepName); finish {
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
	if rp.stepTimeout != 0 {
		timeout = rp.stepTimeout
	}
	if step.stepTimeout != 0 {
		timeout = step.stepTimeout
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

func (rp *runProcess) finalize() {
	timeout := 3 * time.Hour
	if rp.procTimeout != 0 {
		timeout = rp.procTimeout
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

func (rp *runProcess) scheduleNextSteps(step *runStep) {
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

func (rp *runProcess) runStep(step *runStep) {
	defer func() { step.finish <- true }()

	step.Start = time.Now().UTC()
	rp.stepCallback(step, Before)
	// if beforeStep panic occur, the step will mark as panic
	if !step.Normal() {
		return
	}

	retry := 1
	if rp.stepRetry > 0 {
		retry = rp.stepRetry
	}
	if step.stepRetry > 0 {
		retry = step.stepRetry
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
		result, err := step.run(step)
		step.End = time.Now().UTC()
		step.Err = err
		if err != nil {
			step.Append(Error)
			step.Append(Failed)
			continue
		}
		if result != nil {
			step.setResult(step.stepName, result)
		}
		step.Append(Success)
		break
	}
}

func (rp *runProcess) stepCallback(step *runStep, flag string) {
	if rp.stepChain == nil {
		return
	}
	info := rp.summaryStepInfo(step)
	if panicStack := rp.stepChain.process(flag, info); step.Err == nil && len(panicStack) != 0 {
		step.Err = fmt.Errorf(panicStack)
	}
}

func (rp *runProcess) procCallback(flag string) {
	if rp.procChain == nil {
		return
	}

	info := &ProcessInfo{
		visibleContext: rp.visibleContext,
		basicInfo: basicInfo{
			Status: rp.Status,
			Id:     rp.id,
			Name:   rp.processName,
		},
		FlowId: rp.flowId,
	}

	rp.procChain.process(flag, info)
}

func (rp *runProcess) summaryStepInfo(step *runStep) *StepInfo {
	if step.infoCache != nil {
		step.syncInfo()
		return step.infoCache
	}
	info := &StepInfo{
		basicInfo: &basicInfo{
			Status: step.Status,
			Id:     step.id,
			Name:   step.stepName,
		},
		visibleContext: step.visibleContext,
		ProcessId:      step.processId,
		FlowId:         step.flowId,
		Start:          step.Start,
		End:            step.End,
		Err:            step.Err,
		Prev:           make(map[string]string, len(step.depends)),
		Next:           make(map[string]string, len(step.waiters)),
	}
	for _, prev := range step.depends {
		info.Prev[prev.stepName] = rp.flowSteps[prev.stepName].id
	}

	for _, next := range step.waiters {
		info.Next[next.stepName] = rp.flowSteps[next.stepName].id
	}

	step.infoCache = info
	return info
}
