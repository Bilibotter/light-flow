package light_flow

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type runProcess struct {
	*state
	*ProcessMeta
	*dependentContext
	belong    *runFlow
	id        string
	flowSteps map[string]*runStep
	start     time.Time
	end       time.Time
	pause     sync.WaitGroup
	running   sync.WaitGroup
	finish    sync.WaitGroup
}

func (process *runProcess) ID() string {
	return process.id
}

func (process *runProcess) StartTime() time.Time {
	return process.start
}

func (process *runProcess) EndTime() time.Time {
	return process.end
}

func (process *runProcess) CostTime() time.Duration {
	if process.end.IsZero() {
		return 0
	}
	return process.end.Sub(process.start)
}

func (process *runProcess) buildRunStep(meta *StepMeta) *runStep {
	step := runStep{
		dependentContext: &dependentContext{
			routing:     &meta.nodeRouter,
			linkedTable: process.linkedTable,
			prev:        process.dependentContext.prev,
		},
		belong:   process,
		state:    emptyStatus(),
		StepMeta: meta,
		id:       generateId(),
		segments: segment(len(meta.depends)),
		finish:   make(chan bool, 1),
	}

	process.flowSteps[meta.name] = &step

	return &step
}

func (process *runProcess) clearExecutedFromRoot(root string) {
	current := process.flowSteps[root]
	current.clear(executed)
	for _, meta := range current.waiters {
		process.clearExecutedFromRoot(meta.name)
	}
}

func (process *runProcess) Step(name string) (StepController, bool) {
	if step, ok := process.flowSteps[name]; ok {
		return step, true
	}
	return nil, false
}

func (process *runProcess) Exceptions() []FinishedStep {
	finished := make([]FinishedStep, 0)
	for _, step := range process.flowSteps {
		if !step.Normal() {
			finished = append(finished, step)
		}
	}
	return finished
}

func (process *runProcess) FlowID() string {
	return process.belong.id
}

func (process *runProcess) Resume() ProcController {
	if process.clear(Pause) {
		process.pause.Done()
	}
	return process
}

func (process *runProcess) Pause() ProcController {
	if process.append(Pause) {
		process.pause.Add(1)
	}
	return process
}

func (process *runProcess) Stop() ProcController {
	process.append(Stop)
	process.append(Failed)
	return process
}

func (process *runProcess) schedule() (finish *sync.WaitGroup) {
	process.initialize()
	process.start = time.Now().UTC()
	finish = &process.finish
	// pre-flow callback due to cancel
	if process.Has(Cancel) {
		return
	}

	process.append(Running)
	process.procCallback(Before)
	for _, step := range process.flowSteps {
		if step.layer == 1 {
			go process.startStep(step)
		}
	}

	process.finish.Add(1)
	go process.finalize()
	return
}

func (process *runProcess) startStep(step *runStep) {
	defer process.startNextSteps(step)

	if step.Has(executed) {
		return
	}

	// If prev step is abnormal, the current step will mark as cancel
	// But other tasks that do not depend on the failed task will continue to execute
	if !step.Normal() {
		return
	}

	if !process.Normal() {
		step.append(Stop)
		return
	}

	for process.Has(Pause) {
		process.pause.Wait()
	}

	if step.Has(skipped) {
		return
	}

	if len(step.depends) > 0 && !process.evaluate(step) {
		step.append(skipped)
		return
	}

	timeout := 3 * time.Hour
	if process.StepTimeout != 0 {
		timeout = process.StepTimeout
	}
	if step.StepTimeout != 0 {
		timeout = step.StepTimeout
	}

	timer := time.NewTimer(timeout)
	go process.runStep(step)
	select {
	case <-timer.C:
		step.append(Timeout)
		step.append(Failed)
	case <-step.finish:
		return
	}
}

func (process *runProcess) evaluate(step *runStep) bool {
	for _, group := range step.evaluators {
		named, unnamed, exist := process.getCondition(group.depend)
		if !exist {
			return false
		}
		if !group.evaluate(named, unnamed) {
			return false
		}
	}
	return true
}

func (process *runProcess) finalize() {
	timeout := 3 * time.Hour
	if process.ProcTimeout != 0 {
		timeout = process.ProcTimeout
	}

	timer := time.NewTimer(timeout)
	finish := make(chan bool, 1)
	go func() {
		process.running.Wait()
		finish <- true
	}()
	select {
	case <-timer.C:
		process.append(Timeout)
		process.append(Failed)
	case <-finish:
	}

	for _, step := range process.flowSteps {
		process.add(step.state.load())
	}

	if process.Normal() {
		process.append(Success)
	}
	process.procCallback(After)
	process.end = time.Now().UTC()
	process.finish.Done()
}

func (process *runProcess) startNextSteps(step *runStep) {
	// all step not execute should cancel while process is timeout
	cancel := !step.Normal() || !process.Normal()
	skip := step.Has(skipped)
	for _, waiter := range step.waiters {
		next := process.flowSteps[waiter.name]
		waiting := next.segments.addWaiting(-1)
		if cancel {
			next.append(Cancel)
		}
		if atomic.LoadInt64(&waiting) != 0 {
			continue
		}
		if step.Success() {
			next.append(Running)
		} else if skip {
			next.append(skipped)
		}
		go process.startStep(next)
	}

	process.running.Done()
}

func (process *runProcess) runStep(step *runStep) {
	defer func() { step.finish <- true }()

	step.start = time.Now().UTC()
	process.stepCallback(step, Before)
	// if beforeStep panic occur, the step will mark as panic
	if !step.Normal() {
		return
	}

	retry := 0
	if process.StepRetry > 0 {
		retry = process.StepRetry
	}
	if step.StepRetry > 0 {
		retry = step.StepRetry
	}

	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("panic: %v\n\n%s\n", r, string(debug.Stack()))
			step.append(Panic)
			step.append(Failed)
			step.exception = panicErr
			step.end = time.Now().UTC()
		}
		process.stepCallback(step, After)
	}()

	for i := 0; i <= retry; i++ {
		result, err := step.run(step)
		step.end = time.Now().UTC()
		step.exception = err
		if err != nil {
			step.append(Error)
			step.append(Failed)
			continue
		}
		if result != nil {
			step.setResult(step.name, result)
		}
		step.append(Success)
		break
	}
}

func (process *runProcess) stepCallback(step *runStep, flag string) {
	//info := process.summaryStepInfo(step)
	panicStack := ""
	defer func() {
		if len(panicStack) > 0 && step.exception == nil {
			step.exception = fmt.Errorf(panicStack)
		}
	}()
	if !process.belong.ProcNotUseDefault && !process.ProcNotUseDefault && defaultConfig != nil {
		panicStack = defaultConfig.stepChain.handle(flag, step)
	}
	if len(panicStack) != 0 {
		return
	}
	panicStack = process.belong.stepChain.handle(flag, step)
	if len(panicStack) != 0 {
		return
	}
	panicStack = process.stepChain.handle(flag, step)
}

func (process *runProcess) procCallback(flag string) {
	if !process.belong.ProcNotUseDefault && !process.ProcNotUseDefault && defaultConfig != nil {
		defaultConfig.procChain.handle(flag, process)
	}
	process.belong.procChain.handle(flag, process)
	process.procChain.handle(flag, process)
}
