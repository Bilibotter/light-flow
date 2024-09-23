package flow

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

type sliceSet[T comparable] []T

type StepMeta struct {
	stepConfig
	condition
	nodeRouter
	belong   *ProcessMeta
	name     string
	layer    int
	position *state              // used to record the position of the step
	depends  sliceSet[*StepMeta] // prev
	waiters  sliceSet[*StepMeta] // next
	restrict map[string]uint64   // used to limit the Context to get certain keys only in the specified step
	run      func(ctx Step) (any, error)
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

func (meta *StepMeta) Next(run func(ctx Step) (any, error), name string) *StepMeta {
	return meta.belong.NamedStep(run, name, meta.name)
}

func (meta *StepMeta) Same(run func(ctx Step) (any, error), name string) *StepMeta {
	depends := make([]any, 0, len(meta.depends))
	for i := 0; i < len(meta.depends); i++ {
		depends = append(depends, meta.depends[i].name)
	}
	return meta.belong.NamedStep(run, name, depends...)
}

func (meta *StepMeta) Restrict(restrict map[string]any) {
	if meta.restrict == nil {
		meta.restrict = make(map[string]uint64, len(restrict))
	}
	for key, stepName := range restrict {
		step, exist := meta.belong.steps[toStepName(stepName)]
		if !exist {
			panic(fmt.Sprintf("[Step: %s] can't matchByHighest ", stepName))
		}
		meta.restrict[key] = step.index
	}
	meta.checkRestrict()
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

// checkRestrict checks if the restrict key corresponds to an existing step.
// If not it will panic.
func (meta *StepMeta) checkRestrict() {
	for _, index := range meta.restrict {
		stepName := meta.toName[index]
		if meta.backSearch(stepName) {
			continue
		}
		panic(fmt.Sprintf("[Step: %s] can't be back tracking by the current step", stepName))
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

func (step *runStep) FlowID() string {
	return step.belong.FlowID()
}

func (step *runStep) FlowName() string {
	return step.belong.FlowName()
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

func (step *runStep) ProcessName() string {
	return step.belong.name
}

func (step *runStep) StartTime() *time.Time {
	return &step.start
}

func (step *runStep) EndTime() *time.Time {
	return &step.end
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

func (step *runStep) finalize() {
	stepPersist.onUpdate(step)
	if step.Has(Suspend) {
		step.belong.append(Suspend)
		step.belong.belong.append(Suspend)
	}
}

func (step *runStep) composeError(stage string, err error) {
	if step.exception == nil {
		step.exception = fmt.Errorf("[Step: %s] [Stage: %s] [ID: %s ] - Failed | Error: %s", step.name, stage, step.id, err.Error())
		return
	}
	step.exception = fmt.Errorf("%s; Additionally, [Stage: %s] - Error: %s", step.exception.Error(), stage, err.Error())
}
