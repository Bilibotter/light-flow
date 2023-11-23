package light_flow

import (
	"fmt"
	"time"
)

type StepMeta struct {
	visitor
	stepConfig
	belong   *ProcessMeta
	stepName string
	layer    int
	position *Status     // used to record the position of the step
	depends  []*StepMeta // prev
	waiters  []*StepMeta // next
	priority map[string]int32
	run      func(ctx Context) (any, error)
}

type runStep struct {
	*StepMeta
	*visibleContext
	*Status
	id        string
	flowId    string
	processId string
	waiting   int64 // the num of wait for dependent step to complete
	finish    chan bool
	Start     time.Time
	End       time.Time
	infoCache *StepInfo
	Err       error
}

type StepInfo struct {
	*basicInfo
	*visibleContext
	ProcessId string
	FlowId    string
	Prev      map[string]string // prev step stepName to step id
	Next      map[string]string // next step stepName to step id
	Start     time.Time
	End       time.Time
	Err       error
}

type stepConfig struct {
	stepTimeout time.Duration
	stepRetry   int
}

func (si *StepInfo) Error() error {
	return si.Err
}

func (meta *StepMeta) Next(run func(ctx Context) (any, error), alias ...string) *StepMeta {
	if len(alias) == 1 {
		return meta.belong.AliasStep(alias[0], run, meta.stepName)

	}
	return meta.belong.Step(run, meta.stepName)
}

func (meta *StepMeta) Same(run func(ctx Context) (any, error), alias ...string) *StepMeta {
	depends := make([]any, 0, len(meta.depends))
	for i := 0; i < len(meta.depends); i++ {
		depends = append(depends, meta.depends[i].stepName)
	}
	if len(alias) == 1 {
		return meta.belong.AliasStep(alias[0], run, depends...)
	}
	return meta.belong.Step(run, depends)
}

func (meta *StepMeta) Priority(priority map[string]any) {
	if meta.priority == nil {
		meta.priority = make(map[string]int32, len(priority))
	}
	for key, stepName := range priority {
		step, exist := meta.belong.steps[toStepName(stepName)]
		if !exist {
			panic(fmt.Sprintf("can't find step[%s]", stepName))
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
		wired := CreateSetBySliceFunc(depend.waiters, func(waiter *StepMeta) string { return waiter.stepName })
		if wired.Contains(meta.stepName) {
			continue
		}
		depend.waiters = append(depend.waiters, meta)
		if depend.position.Contain(End) {
			depend.position.Append(HasNext)
			depend.position.Pop(End)
		}
		if depend.layer+1 > meta.layer {
			meta.layer = depend.layer + 1
		}
	}

	if len(meta.depends) == 0 {
		meta.position.Append(Head)
	}

	meta.position.Append(End)
}

// checkPriority checks if the priority key corresponds to an existing step.
// If not it will panic.
func (meta *StepMeta) checkPriority() {
	for _, index := range meta.priority {
		stepName := meta.roster[index]
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
	meta.stepConfig.stepTimeout = timeout
	return meta
}

func (meta *StepMeta) Retry(retry int) *StepMeta {
	meta.stepConfig.stepRetry = retry
	return meta
}

func (step *runStep) syncInfo() {
	if step.infoCache == nil {
		return
	}
	step.infoCache.Status = step.Status
	step.infoCache.Err = step.Err
	step.infoCache.Start = step.Start
	step.infoCache.End = step.End
}
