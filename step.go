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
	accessInfo
	StepConfig
	belong     *ProcessMeta
	name       string
	layer      int
	position   *state              // used to record the position of the step
	depends    SliceSet[*StepMeta] // prev
	waiters    SliceSet[*StepMeta] // next
	priority   map[string]int64
	run        func(ctx StepCtx) (any, error)
	evaluators evaluators
}

type runStep struct {
	*StepMeta
	*state
	*visibleContext
	belong    *runProcess
	id        string
	segments  segment // 0-12 bits for waiting
	finish    chan bool
	Start     time.Time
	End       time.Time
	infoCache *Step // Step is needed for both before and post callbacks, so caching it down
	Err       error
}

type Step struct {
	*basicInfo
	StepCtx
	Start time.Time
	End   time.Time
	Err   error
}

type StepConfig struct {
	StepTimeout time.Duration
	StepRetry   int
}

func (seg *segment) addWaiting(i int64) int64 {
	current := atomic.AddInt64((*int64)(seg), i<<waitingOffset)
	return (current >> waitingOffset) & segMask
}

func (s *Step) Error() error {
	return s.Err
}

func (meta *StepMeta) Next(run func(ctx StepCtx) (any, error), alias ...string) *StepMeta {
	if len(alias) == 1 {
		return meta.belong.NameStep(run, alias[0], meta.name)

	}
	return meta.belong.Step(run, meta.name)
}

func (meta *StepMeta) Same(run func(ctx StepCtx) (any, error), alias ...string) *StepMeta {
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
		meta.priority = make(map[string]int64, len(priority))
	}
	for key, stepName := range priority {
		step, exist := meta.belong.steps[toStepName(stepName)]
		if !exist {
			panic(fmt.Sprintf("step[%s] can't matchByHighest ", stepName))
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
			depend.position.set(hasNextE)
			depend.position.clear(endE)
		}
		if depend.layer+1 > meta.layer {
			meta.layer = depend.layer + 1
		}
	}

	if len(meta.depends) == 0 {
		meta.position.set(headE)
	}

	meta.position.set(endE)
}

// checkPriority checks if the priority key corresponds to an existing step.
// If not it will panic.
func (meta *StepMeta) checkPriority() {
	for _, index := range meta.priority {
		stepName := meta.names[index]
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

func (meta *StepMeta) Timeout(timeout time.Duration) *StepMeta {
	meta.StepConfig.StepTimeout = timeout
	return meta
}

func (meta *StepMeta) Retry(retry int) *StepMeta {
	meta.StepConfig.StepRetry = retry
	return meta
}

func (step *runStep) GetFlowId() string {
	return step.belong.GetFlowId()
}

func (step *runStep) GetProcessId() string {
	return step.belong.id
}

func (step *runStep) SetCondition(value any, targets ...string) {
	step.setExactCond(step.name, value, targets...)
}

func (step *runStep) syncInfo() {
	if step.infoCache == nil {
		return
	}
	step.infoCache.state = step.state
	step.infoCache.Err = step.Err
	step.infoCache.Start = step.Start
	step.infoCache.End = step.End
}

func (step *runStep) needRecover() bool {
	return !step.Normal() && !step.Has(Cancel)
}
