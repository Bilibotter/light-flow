package light_flow

import (
	"fmt"
	"time"
)

type StepMeta struct {
	stepName    string
	funcName    string
	order       int
	position    int64
	config      *StepConfig
	depends     []*StepMeta // prev
	waiters     []*StepMeta // next
	ctxPriority map[string]string
	run         func(ctx *Context) (any, error)
}

type FlowStep struct {
	*StepMeta
	*Context
	id        string
	flowId    string
	processId string
	waiting   int64 // the num of wait for dependent step to complete
	status    int64
	finish    chan bool
	Start     time.Time
	End       time.Time
	Err       error
}

type StepInfo struct {
	Id        string
	ProcessId string
	FlowId    string
	Name      string
	Status    int64
	Prev      map[string]string // prev step stepName to step id
	Next      map[string]string // next step stepName to step id
	Ctx       *Context
	Config    *StepConfig
	Start     time.Time
	End       time.Time
	Err       error
}

type StepConfig struct {
	Timeout  time.Duration
	MaxRetry int
}

func (meta *StepMeta) AddPriority(priority map[string]any) {
	for key, stepName := range priority {
		if meta.stepName == stepName {
			panic(fmt.Sprintf("step [%s] can't add self to priority", stepName))
		}
		meta.ctxPriority[key] = toStepName(stepName)
	}
	meta.checkPriority()
}

// checkPriority checks if the priority key corresponds to an existing step.
// If not it will panic.
func (meta *StepMeta) checkPriority() {
	for _, stepName := range meta.ctxPriority {
		if meta.backTrackSearch(stepName) {
			continue
		}
		panic(fmt.Sprintf("step [%s] can't be back tracking by the current step", stepName))
	}
}

func (meta *StepMeta) backTrackSearch(searched string) bool {
	if meta.stepName == searched {
		return true
	}

	for _, depend := range meta.depends {
		if depend.backTrackSearch(searched) {
			return true
		}
	}

	return false
}

// AddConfig allow step not using process's config
func (meta *StepMeta) AddConfig(config *StepConfig) {
	meta.config = config
}
