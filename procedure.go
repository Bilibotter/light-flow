package light_flow

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Procedure struct {
	name              string
	stepMap           map[string]*Step
	procedureContexts map[string]*Context
	tail              string
	context           *Context
	pause             sync.WaitGroup
	running           sync.WaitGroup
	finish            int64
	status            int64
	conf              *ProduceConfig
}

type ProcedureInfo struct {
	Name    string
	StepMap map[string]*StepInfo
	Ctx     *Context
}

type ProduceConfig struct {
	StepTimeout        time.Duration
	ProcedureTimeout   time.Duration
	StepRetry          int
	PreProcessors      []func(stepName string, ctx *Context) (keepOn bool)
	PostProcessors     []func(info *StepInfo) (keepOn bool)
	CompleteProcessors []func(info *ProcedureInfo) (keepOn bool)
}

func (pcd *Procedure) SupplyCtxByMap(update map[string]any) {
	for key, value := range update {
		pcd.context.Set(key, value)
	}
}

func (pcd *Procedure) SupplyCtx(key string, value any) {
	pcd.context.Set(key, value)
}

func (pcd *Procedure) AddConfig(config *ProduceConfig) {
	pcd.conf = config
}

func (pcd *Procedure) AddStep(run func(ctx *Context) (any, error), depends ...any) *Step {
	return pcd.AddStepWithAlias(GetFuncName(run), run, depends...)
}

func (pcd *Procedure) AddWaitBefore(alias string, run func(ctx *Context) (any, error)) *Step {
	return pcd.AddStepWithAlias(alias, run, pcd.tail)
}

func (pcd *Procedure) AddWaitAll(alias string, run func(ctx *Context) (any, error)) *Step {
	depends := make([]any, 0)
	for name, step := range pcd.stepMap {
		if step.position == End {
			depends = append(depends, name)
		}
		if step.position == Start && len(step.send) == 0 {
			depends = append(depends, name)
		}
	}
	return pcd.AddStepWithAlias(alias, run, depends...)
}

func (pcd *Procedure) AddStepWithAlias(alias string, run func(ctx *Context) (any, error), depends ...any) *Step {
	step := &Step{
		run:     run,
		name:    alias,
		waiting: int64(len(depends)),
		finish:  make(chan bool, 1),
		receive: make([]*Step, 0, len(depends)),
		send:    make([]*Step, 0),
		ctx: &Context{
			scopeContexts: pcd.procedureContexts,
			scope:         ProcedureCtx,
			table:         sync.Map{},
		},
	}

	for _, depend := range depends {
		index := GetIndex(depend)
		prev, exist := pcd.stepMap[index]
		if !exist {
			panic(fmt.Sprintf("can't find step named %s", index))
		}

		step.receive = append(step.receive, prev)
		step.ctx.parents = append(step.ctx.parents, prev.ctx)
		prev.send = append(prev.send, step)
		if prev.position == End {
			prev.position = NoUse
		}
	}

	if _, exist := pcd.stepMap[alias]; exist {
		panic(fmt.Sprintf("step named %s already exist, can used %s to avoid name duplicate",
			alias, GetFuncName(pcd.AddStepWithAlias)))
	}

	pcd.tail = step.name
	pcd.stepMap[alias] = step
	pcd.procedureContexts[alias] = step.ctx
	if len(depends) == 0 {
		step.ctx.parents = append(step.ctx.parents, pcd.context)
		step.position = Start
	}
	pcd.running.Add(1)

	return step
}

func (pcd *Procedure) Pause() {
	if AppendStatus(&pcd.status, Pause) {
		pcd.pause.Add(1)
	}
}

func (pcd *Procedure) DrawRelation() (procedureName string, depends map[string][]string) {
	procedureName = pcd.name
	for name, step := range pcd.stepMap {
		depends[name] = make([]string, 0, len(step.send))
		for _, send := range step.send {
			depends[name] = append(depends[name], send.name)
		}
	}
	return
}

func (pcd *Procedure) run() *Feature {
	AppendStatus(&pcd.status, Running)
	for _, step := range pcd.stepMap {
		if step.position == Start {
			go pcd.planStep(step)
		}
	}

	go pcd.updateStatusAfterFinish()

	feature := Feature{
		status:  &pcd.status,
		finish:  &pcd.finish,
		running: &pcd.running,
	}

	return &feature
}

func (pcd *Procedure) planStep(step *Step) {
	defer pcd.planNextSteps(step)

	if _, finish := pcd.getStepResult(step.name); finish {
		AppendStatus(&step.status, Success)
		return
	}
	status := atomic.LoadInt64(&pcd.status)
	// If a step fails, the procedure fails.
	// But other tasks that do not depend on the failed task will continue to execute
	if !IsStatusNormal(status) {
		AppendStatus(&step.status, Cancel)
		return
	}

	// if prev step is abnormal, the current step will mark as cancel
	if status = atomic.LoadInt64(&step.status); !IsStatusNormal(status) {
		return
	}

	for status = atomic.LoadInt64(&pcd.status); status&Pause == Pause; {
		pcd.pause.Wait()
		status = atomic.LoadInt64(&pcd.status)
	}

	go pcd.executeStep(step)

	timeout := 3 * time.Hour
	if pcd.conf != nil && pcd.conf.StepTimeout != 0 {
		timeout = pcd.conf.StepTimeout
	}
	if step.config != nil && step.config.Timeout != 0 {
		timeout = step.config.Timeout
	}

	timer := time.NewTimer(timeout)
	select {
	case <-timer.C:
		AppendStatus(&step.status, Timeout)
	case <-step.finish:
		return
	}
}

func (pcd *Procedure) executeStep(step *Step) {
	step.Start = time.Now().UTC()

	if pcd.conf != nil && len(pcd.conf.PreProcessors) > 0 {
		for _, processor := range pcd.conf.PreProcessors {
			if !processor(step.name, step.ctx) {
				break
			}
		}
	}

	retry := 1
	if pcd.conf != nil && pcd.conf.StepRetry > 0 {
		retry = pcd.conf.StepRetry
	}
	if step.config != nil && step.config.MaxRetry > 0 {
		retry = step.config.MaxRetry
	}

	defer func() {
		if r := recover(); r != nil {
			AppendStatus(&step.status, Panic)
			step.finish <- true
		}
		if pcd.conf == nil || len(pcd.conf.PostProcessors) == 0 {
			step.finish <- true
			return
		}
		for _, processor := range pcd.conf.PostProcessors {
			if !processor(buildInfo(step)) {
				break
			}
		}
		step.finish <- true
	}()

	for i := 0; i < retry; i++ {
		result, err := step.run(step.ctx)
		step.End = time.Now().UTC()
		step.Err = err
		step.ctx.Set(step.name, result)
		if err != nil {
			AppendStatus(&step.status, Failed)
			continue
		}
		pcd.storeStepResult(step.name, result)
		AppendStatus(&step.status, Success)
		break
	}
}

func (pcd *Procedure) updateStatusAfterFinish() {
	finish := make(chan bool, 1)
	go func() {
		pcd.running.Wait()
		info := &ProcedureInfo{
			Name:    pcd.name,
			StepMap: make(map[string]*StepInfo, len(pcd.stepMap)),
			Ctx:     pcd.context,
		}
		for name, step := range pcd.stepMap {
			info.StepMap[name] = buildInfo(step)
		}
		if pcd.conf == nil || len(pcd.conf.CompleteProcessors) == 0 {
			finish <- true
			return
		}
		for _, processor := range pcd.conf.CompleteProcessors {
			if !processor(info) {
				break
			}
		}
		finish <- true
	}()

	timeout := 3 * time.Hour
	if pcd.conf != nil && pcd.conf.ProcedureTimeout != 0 {
		timeout = pcd.conf.ProcedureTimeout
	}

	timer := time.NewTimer(timeout)
	select {
	case <-timer.C:
		AppendStatus(&pcd.status, Timeout)
	case <-finish:
	}

	for _, step := range pcd.stepMap {
		AppendStatus(&pcd.status, step.status)
	}
	if IsStatusNormal(pcd.status) {
		AppendStatus(&pcd.status, Success)
	}
	atomic.StoreInt64(&pcd.finish, 1)
}

func (pcd *Procedure) planNextSteps(step *Step) {
	// all step not execute should cancel while procedure is timeout
	cancel := !IsStatusNormal(step.status) || !IsStatusNormal(pcd.status)
	for _, send := range step.send {
		current := atomic.AddInt64(&send.waiting, -1)
		if cancel {
			AppendStatus(&send.status, Cancel)
		}
		if current != 0 {
			continue
		}
		if step.status&Success == Success {
			AppendStatus(&send.status, Running)
		}
		go pcd.planStep(send)
	}

	pcd.running.Done()
}

func (pcd *Procedure) SkipFinishedStep(name string, result any) {
	pcd.storeStepResult(name, result)
}

func (pcd *Procedure) storeStepResult(name string, value any) {
	key := InternalPrefix + name
	if _, exist := pcd.context.Get(key); !exist {
		pcd.context.Set(key, value)
	}
}

func (pcd *Procedure) getStepResult(name string) (any, bool) {
	key := InternalPrefix + name
	return pcd.context.Get(key)
}
