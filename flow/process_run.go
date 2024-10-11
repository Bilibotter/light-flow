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
	runSteps  map[string]*runStep
	resources map[string]*resource
	start     time.Time
	end       time.Time
	pause     sync.WaitGroup
	running   sync.WaitGroup
}

func (process *runProcess) HasAny(enum ...*StatusEnum) bool {
	for _, step := range process.runSteps {
		if step.Has(enum...) {
			return true
		}
	}
	return process.Has(enum...)
}

func (process *runProcess) ID() string {
	return process.id
}

func (process *runProcess) StartTime() *time.Time {
	return &process.start
}

func (process *runProcess) EndTime() *time.Time {
	return &process.end
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

func (process *runProcess) Steps() []FinishedStep {
	res := make([]FinishedStep, 0, len(process.runSteps))
	for _, step := range process.runSteps {
		res = append(res, step)
	}
	return res
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

func (process *runProcess) FlowName() string {
	return process.belong.name
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

func (process *runProcess) schedule() {
	defer process.finalize()
	process.initialize()
	if process.Has(executed) {
		return
	}
	process.start = time.Now().UTC()
	if process.Has(Recovering) {
		process.manageResources(recoverR)
	}
	procPersist.onInsert(process)
	process.advertise(beforeF)
	if process.Has(CallbackFail) {
		process.end = time.Now().UTC()
		return
	}
	for _, step := range process.runSteps {
		if step.layer == 1 {
			go process.startStep(step)
		}
	}
	process.waitFinish()
	process.end = time.Now().UTC()
	process.advertise(afterF)
	return
}

func (process *runProcess) startStep(step *runStep) {
	defer process.startNextSteps(step)

	if step.Has(executed) {
		return
	}

	// If prev step is abnormal, the current step will mark as cancel
	// But other tasks that do not depend on the failed task will continue to execute
	if step.Has(Cancel) {
		return
	}

	// process is timeout or stop
	if !process.Normal() {
		step.append(Stop)
		return
	}

	if step.Has(skipped) {
		if _, exclude := step.getInternal(fmt.Sprintf(stepCD, step.Name())); !exclude {
			return
		}
	}

	// If the step has the Recovering flag, it means the step met the condition before recovering,
	// so it should be executed again.
	if !step.meetCondition(step) && !step.Has(Recovering) {
		return
	}

	for process.Has(Pause) {
		process.pause.Wait()
	}

	timeout := step.stepTimeout
	if timeout == 0 {
		timeout = 3 * time.Hour
	} else if timeout < 0 {
		timeout = time.Duration(1<<63 - 1)
	}

	timer := time.NewTimer(timeout)
	defer step.finalize()
	step.start = time.Now().UTC()
	stepPersist.onInsert(step)
	process.stepCallback(step, beforeF)
	// if beforeStep panic occur, the step will mark as panic
	if step.Has(CallbackFail) {
		step.end = time.Now().UTC()
		return
	}
	go process.runStep(step)
	select {
	case <-timer.C:
		step.append(Timeout)
		step.composeError(execStage, fmt.Errorf("execute timeout"))
	case <-step.finish:
	}
	if !step.Normal() && step.isRecoverable() {
		step.append(Suspend)
		step.setInternal(fmt.Sprintf(stepBP, step.Name()), &breakPoint{
			Stage:   1<<0 | 1<<9, // execute default after step callback from 0
			Index:   0,
			SkipRun: false,
		})
	}
	step.end = time.Now().UTC()
	process.stepCallback(step, afterF)
}

func (process *runProcess) waitFinish() {
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
		if !step.Normal() {
			process.append(Failed)
			break
		}
	}
	if !process.Has(Failed) {
		process.append(Success)
	}
}

func (process *runProcess) finalize() {
	// any step suspend will cause the process suspend
	if process.Has(Suspend) {
		process.belong.append(Suspend)
		process.manageResources(suspendR)
	}
	process.manageResources(releaseR)
	procPersist.onUpdate(process)
	process.belong.running.Done()
}

func (process *runProcess) manageResources(action string) {
	if process.resources == nil {
		return
	}
	resName := "-"
	defer func() {
		if r := recover(); r != nil {
			event := panicEvent(process, InResource, r, stack())
			event.write("Action", action)
			event.write("Resource", resName)
			dispatcher.send(event)
		}
	}()
	var err error
	for name, res := range process.resources {
		resName = name
		switch action {
		case releaseR:
			err = resourceManagers[name].onRelease(res)
		case recoverR:
			err = resourceManagers[name].onRecover(res)
		case suspendR:
			err = resourceManagers[name].onSuspend(res)
		}
		if err != nil {
			event := errorEvent(process, InResource, err)
			event.write("Action", action)
			event.write("Resource", name)
			dispatcher.send(event)
		}
	}
	if action == suspendR {
		process.makeResSerializable()
	}
}

func (process *runProcess) makeResSerializable() {
	if process.resources == nil {
		return
	}
	saver := make(map[string]*resSerializable, len(process.resources))
	for k, v := range process.resources {
		saver[k] = &resSerializable{
			Name:   v.resName,
			Ctx:    v.ctx,
			Entity: v.entity,
		}
	}
	process.setInternal(resSerializableKey, saver)
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
		if skip {
			next.append(skipped)
		}
		if atomic.LoadInt64(&waiting) != 0 {
			continue
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

	retry := step.stepRetry
	if retry < 0 {
		retry = 0
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[Step: %s] execute panic;\nID=%s;\nPanic=%v\n%s\n", step.name, step.id, r, stack())
			step.append(Panic)
			step.composeError(execStage, fmt.Errorf("execute panic: %v", r))
		}
	}()

	if step.Has(Recovering) {
		point, exist := step.getInternal(fmt.Sprintf(stepBP, step.Name()))
		if exist && point.(*breakPoint).SkipRun {
			return
		}
	}

	var exception error
	for i := 0; i <= retry; i++ {
		result, err := step.run(step)
		if err != nil {
			exception = err
			continue
		}
		if result != nil {
			step.setResult(step.name, result)
		}
		step.append(Success)
		return
	}

	step.append(Error)
	step.composeError(execStage, exception)
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

func (process *runProcess) Attach(resName string, initParam any) (_ Resource, err error) {
	defer func() {
		if r := recover(); r != nil {
			event := panicEvent(process, InResource, r, stack())
			event.write("Action", attachR)
			event.write("Resource", resName)
			dispatcher.send(event)
			err = fmt.Errorf("panic=%v", r)
		}
	}()
	if foo, find := process.Acquire(resName); find {
		return foo, nil
	}
	manager, exist := resourceManagers[resName]
	if !exist {
		err = fmt.Errorf("[Resource: %s] not registered", resName)
		return nil, err
	}
	res := &resource{boundProc: process, resName: resName}
	instance, err := manager.onInitialize(res, initParam)
	if err != nil {
		event := errorEvent(process, InResource, err)
		event.write("Action", initializeR)
		event.write("Resource", resName)
		dispatcher.send(event)
		return nil, err
	}
	res.entity = instance
	process.Lock()
	defer process.Unlock()
	if process.resources == nil {
		process.resources = make(map[string]*resource, 1)
	}
	// DCL
	if ret, find := process.resources[resName]; find {
		return ret, nil
	}
	process.resources[resName] = res
	return res, nil
}

func (process *runProcess) Acquire(resName string) (res Resource, exist bool) {
	process.RLock()
	defer process.RUnlock()
	if process.resources == nil {
		return
	}
	res, exist = process.resources[resName]
	return
}
