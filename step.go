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
	conditionOffset
)

type segment int64

type evalGroups []*evalGroup

type StepMeta struct {
	accessInfo
	StepConfig
	belong     *ProcessMeta
	stepName   string
	layer      int
	position   *state      // used to record the position of the step
	depends    []*StepMeta // prev
	waiters    []*StepMeta // next
	priority   map[string]int64
	run        func(ctx StepCtx) (any, error)
	evalGroups evalGroups
}

type runStep struct {
	*StepMeta
	*state
	*visibleContext
	id        string
	flowId    string
	processId string
	segments  segment //  0-12 bits for waiting, 12-24 bits for condition.
	finish    chan bool
	Start     time.Time
	End       time.Time
	infoCache *Step
	Err       error
}

type Step struct {
	*basicInfo
	StepCtx
	ProcessId string
	FlowId    string
	Start     time.Time
	End       time.Time
	Err       error
}

type StepConfig struct {
	StepTimeout time.Duration
	StepRetry   int
}

func (seg *segment) loadWaiting() int64 {
	return (atomic.LoadInt64((*int64)(seg)) >> waitingOffset) & segMask
}

func (seg *segment) loadCondition() int64 {
	return (atomic.LoadInt64((*int64)(seg)) >> conditionOffset) & segMask
}

func (seg *segment) addWaiting(i int64) int64 {
	current := atomic.AddInt64((*int64)(seg), i<<waitingOffset)
	return (current >> waitingOffset) & segMask
}

func (seg *segment) addCondition(i int64) int64 {
	current := atomic.AddInt64((*int64)(seg), i<<conditionOffset)
	return (current >> conditionOffset) & segMask
}

func (si *Step) Error() error {
	return si.Err
}

func (meta *StepMeta) Next(run func(ctx StepCtx) (any, error), alias ...string) *StepMeta {
	if len(alias) == 1 {
		return meta.belong.NameStep(run, alias[0], meta.stepName)

	}
	return meta.belong.Step(run, meta.stepName)
}

func (meta *StepMeta) Same(run func(ctx StepCtx) (any, error), alias ...string) *StepMeta {
	depends := make([]any, 0, len(meta.depends))
	for i := 0; i < len(meta.depends); i++ {
		depends = append(depends, meta.depends[i].stepName)
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
		wired := createSetBySliceFunc(depend.waiters, func(waiter *StepMeta) string { return waiter.stepName })
		if wired.Contains(meta.stepName) {
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
		if waiter.stepName == searched {
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
		if depend.stepName == searched {
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

func (step *runStep) SetCond(value any) {
	step.setCond(step.stepName, value)
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

func (sc *StepConfig) combine(config *StepConfig) {
	copyPropertiesSkipNotEmpty(sc, config)
}

func (sc *StepConfig) clone() StepConfig {
	config := StepConfig{}
	copyPropertiesSkipNotEmpty(sc, config)
	return config
}
