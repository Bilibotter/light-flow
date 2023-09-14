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
	Prev      map[string]string // prev step name to step id
	Next      map[string]string // next step name to step id
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
		Prev:      make(map[string]string, len(step.receive)),
		Next:      make(map[string]string, len(step.send)),
	}
	for _, prev := range step.receive {
		info.Prev[prev.name] = prev.id
	}
	for _, next := range step.send {
		info.Next[next.name] = next.id
	}
	return info
}

// AddPriority changes the order to retrieve a specified key.
func (step *Step) AddPriority(priority map[string]any) {
	step.ctx.priority = priority
	step.ctx.checkPriority()
}

// AddConfig allow step not using process's config
func (step *Step) AddConfig(config *StepConfig) {
	step.config = config
}
