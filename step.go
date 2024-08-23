package light_flow

import (
	"fmt"
	"sync/atomic"
	"time"
)

const (
	segmentSize       = 12
	segMask     int64 = (1 << segmentSize) - 1
)

const (
	waitingOffset int64 = segmentSize * iota
)

type segment int64

type evaluators []*evaluator

type StepMeta struct {
	stepConfig
	stepCallback
	nodeRouter
	belong     *ProcessMeta
	name       string
	layer      int
	position   *state              // used to record the position of the step
	depends    sliceSet[*StepMeta] // prev
	waiters    sliceSet[*StepMeta] // next
	priority   map[string]uint64
	run        func(ctx Step) (any, error)
	evaluators evaluators
}

type runStep struct {
	*state
	*StepMeta
	*dependentContext
	belong    *runProcess
	id        string
	segments  segment // 0-12 bits for waiting
	finish    chan bool
	start     time.Time
	end       time.Time
	exception error
}

func (seg *segment) addWaiting(i int64) int64 {
	current := atomic.AddInt64((*int64)(seg), i<<waitingOffset)
	return (current >> waitingOffset) & segMask
}

func (meta *StepMeta) Name() string {
	return meta.name
}

func (meta *StepMeta) Next(run func(ctx Step) (any, error), alias ...string) *StepMeta {
	if len(alias) == 1 {
		return meta.belong.NameStep(run, alias[0], meta.name)

	}
	return meta.belong.Step(run, meta.name)
}

func (meta *StepMeta) Same(run func(ctx Step) (any, error), alias ...string) *StepMeta {
	depends := make([]any, 0, len(meta.depends))
	for i := 0; i < len(meta.depends); i++ {
		depends = append(depends, meta.depends[i].name)
	}
	if len(alias) == 1 {
		return meta.belong.NameStep(run, alias[0], depends...)
	}
	return meta.belong.Step(run, depends)
}

func (meta *StepMeta) Priority(priority map[string]any) {
	if meta.priority == nil {
		meta.priority = make(map[string]uint64, len(priority))
	}
	for key, stepName := range priority {
		step, exist := meta.belong.steps[toStepName(stepName)]
		if !exist {
			panic(fmt.Sprintf("Step[ %s ] can't matchByHighest ", stepName))
		}
		meta.priority[key] = step.index
	}
	meta.checkPriority()
}

func (meta *StepMeta) wireDepends() {
	if meta.position == nil {
		meta.position = emptyStatus()
	}

	for _, depend := range meta.depends {
		wired := createSetBySliceFunc(depend.waiters, func(waiter *StepMeta) string { return waiter.name })
		if wired.Contains(meta.name) {
			continue
		}
		depend.waiters = append(depend.waiters, meta)
		if depend.position.Has(endE) {
			depend.position.append(hasNextE)
			depend.position.clear(endE)
		}
		if depend.layer+1 > meta.layer {
			meta.layer = depend.layer + 1
		}
	}

	if len(meta.depends) == 0 {
		meta.position.append(headE)
	}

	meta.position.append(endE)
}

// checkPriority checks if the priority key corresponds to an existing step.
// If not it will panic.
func (meta *StepMeta) checkPriority() {
	for _, index := range meta.priority {
		stepName := meta.toName[index]
		if meta.backSearch(stepName) {
			continue
		}
		panic(fmt.Sprintf("step [%s] can't be back tracking by the current step", stepName))
	}
}

func (meta *StepMeta) forwardSearch(searched string) bool {
	for _, waiter := range meta.waiters {
		if waiter.name == searched {
			return true
		}
		if waiter.forwardSearch(searched) {
			return true
		}
	}

	return false
}

func (meta *StepMeta) backSearch(searched string) bool {
	for _, depend := range meta.depends {
		if depend.name == searched {
			return true
		}
		if depend.backSearch(searched) {
			return true
		}
	}

	return false
}

func (step *runStep) Err() error {
	return step.exception
}

func (step *runStep) ID() string {
	return step.id
}

func (step *runStep) GetResult() any {
	res, exist := step.Result(step.name)
	if !exist {
		return nil
	}
	return res
}

func (step *runStep) FlowID() string {
	return step.belong.FlowID()
}

func (step *runStep) Dependents() (stepNames []string) {
	s := make([]string, len(step.depends))
	for i, depend := range step.depends {
		s[i] = depend.name
	}
	return s
}

func (step *runStep) ProcessID() string {
	return step.belong.id
}

func (step *runStep) StartTime() time.Time {
	return step.start
}

func (step *runStep) EndTime() time.Time {
	return step.end
}

func (step *runStep) CostTime() time.Duration {
	if step.end.IsZero() {
		return 0
	}
	return step.end.Sub(step.start)
}

func (step *runStep) Attach(resName string, initParam any) (res Resource, err error) {
	return step.belong.Attach(resName, initParam)
}

func (step *runStep) Acquire(resName string) (res Resource, exist bool) {
	return step.belong.Acquire(resName)
}

func (step *runStep) isRecoverable() bool {
	return step.belong.isRecoverable()
}

func (step *runStep) needRecover() bool {
	return step.Has(CallbackFail) || (!step.Normal() && !step.Has(Cancel))
}
