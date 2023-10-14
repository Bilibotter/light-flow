package light_flow

import (
	"fmt"
	"time"
)

type StepMeta struct {
	*Status
	*StepConfig
	belong      *ProcessMeta
	stepName    string
	layer       int
	depends     []*StepMeta // prev
	waiters     []*StepMeta // next
	ctxPriority map[string]string
	run         func(ctx *Context) (any, error)
}

type RunStep struct {
	*StepMeta
	*Context
	*Status
	id        string
	flowId    string
	processId string
	waiting   int64 // the num of wait for dependent step to complete
	finish    chan bool
	Start     time.Time
	End       time.Time
	Err       error
}

type StepInfo struct {
	*BasicInfo
	*Context
	ProcessId string
	FlowId    string
	Prev      map[string]string // prev step stepName to step id
	Next      map[string]string // next step stepName to step id
	Start     time.Time
	End       time.Time
	Err       error
}

type StepConfig struct {
	StepTimeout time.Duration
	StepRetry   int
}

func (si *StepInfo) Error() error {
	return si.Err
}

func (meta *StepMeta) AddPriority(priority map[string]any) {
	if meta.ctxPriority == nil {
		meta.ctxPriority = make(map[string]string)
	}
	for key, stepName := range priority {
		meta.ctxPriority[key] = toStepName(stepName)
	}
	meta.checkPriority()
}

func (meta *StepMeta) CopyDepends(src ...any) {
	for _, wrap := range src {
		name := toStepName(wrap)
		meta.belong.copyDepends(name, meta.stepName)
	}
}

func (meta *StepMeta) wireDepends() {
	if meta.Status == nil {
		meta.Status = emptyStatus()
	}

	for _, depend := range meta.depends {
		for _, waiter := range depend.waiters {
			if waiter.stepName == meta.stepName {
				continue
			}
		}
		depend.waiters = append(depend.waiters, meta)
		if depend.Contain(End) {
			depend.Append(HasNext)
			depend.Pop(End)
		}
		if depend.layer+1 > meta.layer {
			meta.layer = depend.layer + 1
		}
	}

	if len(meta.depends) == 0 {
		meta.Append(Head)
	}

	meta.Append(End)
}

// checkPriority checks if the priority key corresponds to an existing step.
// If not it will panic.
func (meta *StepMeta) checkPriority() {
	for _, stepName := range meta.ctxPriority {
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

// AddConfig allow step not using process's config
func (meta *StepMeta) AddConfig(config *StepConfig) {
	meta.StepConfig = config
}
