package core

import "time"

type Step struct {
	id       string
	name     string
	waiting  int64 // the num of wait for dependent step to complete
	status   int64
	position int64
	ctx      *Context
	config   *StepConfig
	receive  []*Step // steps that the current step depends on
	send     []*Step // steps that depend on the current step
	run      func(ctx *Context) (any, error)
	finish   chan bool
	Start    time.Time
	End      time.Time
	Err      error
}

type StepInfo struct {
	Id     string
	Name   string
	Status int64
	Ctx    *Context
	Config *StepConfig
	Start  time.Time
	End    time.Time
	Err    error
}

type StepConfig struct {
	Timeout  time.Duration
	MaxRetry int
}

func buildInfo(step *Step) *StepInfo {
	info := &StepInfo{
		Id:     step.id,
		Name:   step.name,
		Status: step.status,
		Ctx:    step.ctx,
		Config: step.config,
		Start:  step.Start,
		End:    step.End,
		Err:    step.Err,
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
