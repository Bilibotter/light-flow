package light_flow

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type runProcess struct {
	*state
	*ProcessMeta
	*dependentContext
	belong   *runFlow
	id       string
	compress state
	runSteps map[string]*runStep
	start    time.Time
	end      time.Time
	pause    sync.WaitGroup
	running  sync.WaitGroup
	finish   sync.WaitGroup
}

func (process *runProcess) HasAny(enum ...*StatusEnum) bool {
	return process.compress.Has(enum...)
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

func (process *runProcess) isRecoverable() bool {
	return process.belong.isRecoverable()
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

	process.runSteps[meta.name] = &step

	return &step
}

func (process *runProcess) clearExecutedFromRoot(root string) {
	current := process.runSteps[root]
	current.clear(executed)
	for _, meta := range current.waiters {
		process.clearExecutedFromRoot(meta.name)
	}
}

func (process *runProcess) Step(name string) (StepController, bool) {
	if step, ok := process.runSteps[name]; ok {
		return step, true
	}
	return nil, false
}

func (process *runProcess) Exceptions() []FinishedStep {
	finished := make([]FinishedStep, 0)
	for _, step := range process.runSteps {
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
	return process
}

func (process *runProcess) schedule() (finish *sync.WaitGroup) {
	process.initialize()
	finish = &process.finish
	// pre-flow callback due to cancel
	if process.Has(Cancel) || process.Has(executed) {
		return
	}
	process.start = time.Now().UTC()
	process.append(Running)
	process.advertise(beforeF)
	for _, step := range process.runSteps {
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

	timeout := step.stepTimeout
	if timeout == 0 {
		timeout = 3 * time.Hour
	} else if timeout < 0 {
		timeout = time.Duration(1<<63 - 1)
	}

	timer := time.NewTimer(timeout)
	go process.runStep(step)
	select {
	case <-timer.C:
		step.append(Timeout)
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
	timeout := process.procTimeout
	if timeout == 0 {
		timeout = 3 * time.Hour
	} else if timeout < 0 {
		timeout = time.Duration(1<<63 - 1)
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
	case <-finish:
	}

	for _, step := range process.runSteps {
		process.compress.add(step.load())
	}
	process.compress.add(process.load())
	if process.compress.Normal() {
		process.append(Success)
	} else {
		process.append(Failed)
	}
	process.advertise(afterF)
	if process.Has(Suspend) {
		process.belong.append(Suspend)
	}
	process.releaseResources()
	process.compress.add(process.load())
	process.end = time.Now().UTC()
	process.finish.Done()
}

func (process *runProcess) releaseResources() {
	if !process.Has(resAttached) {
		return
	}
	defer func() {
		_ = recover()
	}()
	for key, head := range process.nodes {
		if head.Path != internalMark {
			continue
		}
		if !strings.HasPrefix(key, string(resPrefix)) {
			continue
		}
		res, ok := head.Value.(*resource)
		if !ok {
			continue
		}
		err := resourceManagers[key[1:]].onRelease(res)
		if err != nil {
			logger.Errorf("Process[ %s ] release Resource[ %s ] failed;\n    ID=%s\n,    error=%v", process.Name(), key[1:], process.id, err)
		}
	}
}

func (process *runProcess) startNextSteps(step *runStep) {
	// all step not execute should cancel while process is timeout
	cancel := !step.Normal() || !process.Normal()
	skip := step.Has(skipped)
	// execute the next step in the current goroutine to reduce the frequency of goroutine creation.
	var bind *runStep
	for _, waiter := range step.waiters {
		next := process.runSteps[waiter.name]
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
		if bind != nil {
			go process.startStep(bind)
		}
		bind = next
	}
	process.running.Done()
	if bind != nil {
		process.startStep(bind)
	}
}

func (process *runProcess) runStep(step *runStep) {
	defer func() { step.finish <- true }()

	step.start = time.Now().UTC()
	process.stepCallback(step, beforeF)
	// if beforeStep panic occur, the step will mark as panic
	if !step.Normal() {
		return
	}

	retry := step.stepRetry
	if retry < 0 {
		retry = 0
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("Step[ %s ] execute panic;\n    ID=%s;\n    Panic=%v\n%s\n", step.name, step.id, r, stack())
			step.append(Panic)
			step.exception = fmt.Errorf("%v", r)
			step.end = time.Now().UTC()
		}
		if !step.isRecoverable() {
			process.stepCallback(step, afterF)
			return
		}
		if !step.Normal() && !step.Has(CallbackFail) {
			step.append(Suspend)
			step.setInternal(fmt.Sprintf(stepBP, step.Name()), &stepPanicBreakPoint)
		}
		process.stepCallback(step, afterF)
		if step.Has(Suspend) {
			step.belong.append(Suspend)
			step.belong.belong.append(Suspend)
		}
	}()

	if step.Has(Recovering) {
		point, exist := step.getInternal(fmt.Sprintf(stepBP, step.Name()))
		if exist {
			if point.(*breakPoint).SkipRun {
				return
			}
		}
	}

	for i := 0; i <= retry; i++ {
		result, err := step.run(step)
		step.end = time.Now().UTC()
		step.exception = err
		if err != nil {
			step.append(Error)
			continue
		}
		if result != nil {
			step.setResult(step.name, result)
		}
		step.append(Success)
		break
	}
}

func (process *runProcess) stepCallback(step *runStep, flag uint64) {
	if !process.belong.disableDefault && !process.disableDefault {
		if !defaultCallback.stepFilter(flag, step) {
			return
		}
	}
	if !process.belong.stepFilter(flag, step) {
		return
	}
	if !process.stepFilter(flag, step) {
		return
	}
}

func (process *runProcess) advertise(flag uint64) {
	if !process.belong.disableDefault && !process.disableDefault {
		if !defaultCallback.procFilter(flag, process) {
			return
		}
	}
	if !process.belong.procFilter(flag, process) {
		return
	}
	if !process.procFilter(flag, process) {
		return
	}
}

func (process *runProcess) Attach(resName string, initParam any) (res Resource, err error) {
	res, err = process.attach(process, resName, initParam)
	if !process.Has(resAttached) {
		process.append(resAttached)
	}
	return
}
