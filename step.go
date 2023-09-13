package light_flow

import "time"

type Step struct {
	id        string
	flowId    string
	processId string
	name      string
	waiting   int64 // the num of wait for dependent step to complete
	status    int64
	position  int64
	ctx       *Context
	config    *StepConfig
	receive   []*Step // steps that the current step depends on
	send      []*Step // steps that depend on the current step
	run       func(ctx *Context) (any, error)
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
	Prev      []string // prev step id slice
	Next      []string // next step id slice
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

func buildInfo(step *Step) *StepInfo {
	info := &StepInfo{
		Id:        step.id,
		ProcessId: step.processId,
		FlowId:    step.flowId,
		Name:      step.name,
		Status:    step.status,
		Ctx:       step.ctx,
		Config:    step.config,
		Start:     step.Start,
		End:       step.End,
		Err:       step.Err,
		Prev:      make([]string, 0, len(step.receive)),
		Next:      make([]string, 0, len(step.send)),
	}
	for _, prev := range step.receive {
		info.Prev = append(info.Prev, prev.name)
	}
	for _, next := range step.send {
		info.Next = append(info.Next, next.name)
	}
	return info
}

func (step *Step) AddPriority(priority map[string]any) {
	step.ctx.priority = priority
	step.ctx.checkPriority()
}

func (step *Step) AddConfig(config *StepConfig) {
	step.config = config
}
