package light_flow

import (
	"fmt"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var (
	lock                 = sync.Mutex{}
	defaultProcessConfig *ProcessConfig
)

type ProcessMeta struct {
	processName string
	steps       map[string]*StepMeta
	tailStep    string
	conf        *ProcessConfig
}

type RunProcess struct {
	*ProcessMeta
	*Context
	id        string
	flowId    string
	flowSteps map[string]*RunStep
	pcsScope  map[string]*Context
	pause     sync.WaitGroup
	needRun   sync.WaitGroup
	finish    int64
	status    int64
}

type ProcessInfo struct {
	Id      string
	FlowId  string
	Name    string
	StepMap map[string]*StepInfo
	Ctx     *Context
}

type ProcessConfig struct {
	StepTimeout        time.Duration
	ProcessTimeout     time.Duration
	StepRetry          int
	PreProcessors      []func(info *StepInfo) (keepOn bool)
	PostProcessors     []func(info *StepInfo) (keepOn bool)
	CompleteProcessors []func(info *ProcessInfo) (keepOn bool)
}

func SetDefaultProcessConfig(config *ProcessConfig) {
	lock.Lock()
	defer lock.Unlock()
	defaultProcessConfig = config
}

func (pm *ProcessMeta) register() {
	_, load := allProcess.LoadOrStore(pm.processName, pm)
	if load {
		panic(fmt.Sprintf("process[%s] has exist", pm.processName))
	}
}

func (pm *ProcessMeta) Merge(name string) {
	wrap, exist := allProcess.Load(name)
	if !exist {
		panic(fmt.Sprintf("can't merge not exist process[%s]", name))
	}
	mergedProcess := wrap.(*ProcessMeta)
	for _, merge := range mergedProcess.sortedStepMeta() {
		if _, exist = pm.steps[merge.stepName]; exist {
			pm.mergeStep(merge)
			continue
		}
		depends := make([]any, 0, len(merge.depends))
		for _, depend := range merge.depends {
			depends = append(depends, depend.stepName)
		}
		step := pm.AddStepWithAlias(merge.stepName, merge.run, depends...)
		AppendStatus(&step.position, Merged)
	}
}

func (pm *ProcessMeta) mergeStep(merge *StepMeta) {
	step := pm.steps[merge.stepName]

	for k, v := range merge.ctxPriority {
		if _, exist := step.ctxPriority[k]; exist {
			continue
		}
		step.ctxPriority[k] = v
	}

	// create a set contains all depended on step name
	current := CreateFromSliceFunc[*StepMeta](step.depends,
		func(meta *StepMeta) string { return meta.stepName })

	depends := make([]*StepMeta, len(merge.depends))
	copy(depends, merge.depends)
	sort.Slice(depends, func(i, j int) bool {
		return pm.steps[depends[i].stepName].layer < pm.steps[depends[j].stepName].layer
	})

	for _, add := range depends {
		if current.Contains(add.stepName) {
			continue
		}
		depend := pm.steps[add.stepName]
		if depend.layer > step.layer {
			if step.forwardSearch(depend.stepName) {
				panic(fmt.Sprintf("merge failed, because there is a cycle formed by step[%s] and step[%s].",
					depend.stepName, step.stepName))
			}
		}
		if depend.layer+1 > step.layer {
			step.layer = depend.layer + 1
			pm.updateWaitersLayer(step)
		}
		step.depends = append(step.depends, depend)
		depend.waiters = append(depend.waiters, step)
	}
}

func (pm *ProcessMeta) updateWaitersLayer(step *StepMeta) {
	for _, waiters := range step.waiters {
		if step.layer+1 > waiters.layer {
			waiters.layer = step.layer + 1
			pm.updateWaitersLayer(waiters)
		}
	}
}

func (pm *ProcessMeta) AddConfig(config *ProcessConfig) {
	pm.conf = config
}

func (pm *ProcessMeta) AddPostProcessor(processor func(info *StepInfo) (keepOn bool)) {
	pm.conf.PostProcessors = append(pm.conf.PostProcessors, processor)
}

func (pm *ProcessMeta) AddPreProcessor(processor func(info *StepInfo) (keepOn bool)) {
	pm.conf.PreProcessors = append(pm.conf.PreProcessors, processor)
}

func (pm *ProcessMeta) AddCompleteProcessor(processor func(info *ProcessInfo) (keepOn bool)) {
	pm.conf.CompleteProcessors = append(pm.conf.CompleteProcessors, processor)
}

func (pm *ProcessMeta) AddStep(run func(ctx *Context) (any, error), depends ...any) *StepMeta {
	return pm.AddStepWithAlias(GetFuncName(run), run, depends...)
}

// AddWaitBefore method treats the last added step as a dependency
func (pm *ProcessMeta) AddWaitBefore(alias string, run func(ctx *Context) (any, error)) *StepMeta {
	return pm.AddStepWithAlias(alias, run, pm.tailStep)
}

func (pm *ProcessMeta) AddWaitAll(alias string, run func(ctx *Context) (any, error)) *StepMeta {
	depends := make([]any, 0)
	for name, step := range pm.steps {
		if !Contains(&step.position, HasNext) {
			depends = append(depends, name)
		}
	}

	return pm.AddStepWithAlias(alias, run, depends...)
}

func (pm *ProcessMeta) AddStepWithAlias(alias string, run func(ctx *Context) (any, error), depends ...any) *StepMeta {
	var meta *StepMeta
	var oldDepends *Set

	if old, exist := pm.steps[alias]; exist {
		if !Contains(&pm.steps[alias].position, Merged) {
			panic(fmt.Sprintf("step named [%s] already exist, can used %s to avoid stepName duplicate",
				alias, GetFuncName(pm.AddStepWithAlias)))
		}
		oldDepends = CreateFromSliceFunc(meta.depends, func(value *StepMeta) string { return value.stepName })
		meta = old
	}

	if meta == nil {
		meta = &StepMeta{
			belong:      pm,
			stepName:    alias,
			run:         run,
			layer:       1,
			ctxPriority: make(map[string]string),
		}
	}

	for _, wrap := range depends {
		name := toStepName(wrap)
		if oldDepends != nil && oldDepends.Contains(name) {
			continue
		}
		depend, exist := pm.steps[name]
		if !exist {
			panic(fmt.Sprintf("can't find step[%s]", name))
		}
		meta.depends = append(meta.depends, depend)
		depend.waiters = append(depend.waiters, meta)
		if Contains(&depend.position, End) {
			AppendStatus(&depend.position, HasNext)
		}
		if depend.layer+1 > meta.layer {
			meta.layer = depend.layer + 1
		}
	}

	AppendStatus(&meta.position, End)
	if len(depends) == 0 {
		AppendStatus(&meta.position, Start)
	}

	pm.tailStep = meta.stepName
	pm.steps[alias] = meta

	return meta
}

func (pm *ProcessMeta) copyDepends(src, dst string) {
	srcStep := pm.steps[src]
	dstStep := pm.steps[dst]
	s := CreateFromSliceFunc(srcStep.depends, func(meta *StepMeta) string { return meta.stepName })
	for _, depend := range srcStep.depends {
		if s.Contains(depend.stepName) {
			continue
		}
		dstStep.depends = append(srcStep.depends, depend)
		depend.waiters = append(depend.waiters, dstStep)
	}
}

func (pm *ProcessMeta) sortedStepMeta() []*StepMeta {
	steps := make([]*StepMeta, 0, len(pm.steps))
	for _, step := range pm.steps {
		steps = append(steps, step)
	}
	sort.Slice(steps, func(i, j int) bool {
		return steps[i].layer < steps[j].layer
	})
	return steps
}

func (fp *RunProcess) addStep(meta *StepMeta) *RunStep {
	step := RunStep{
		StepMeta:  meta,
		id:        generateId(),
		processId: fp.id,
		flowId:    fp.flowId,
		waiting:   int64(len(meta.depends)),
		finish:    make(chan bool, 1),
		Context: &Context{
			table:     sync.Map{},
			name:      meta.stepName,
			scopes:    []string{ProcessCtx},
			scopeCtxs: fp.pcsScope,
			priority:  meta.ctxPriority,
		},
	}

	fp.flowSteps[meta.stepName] = &step
	fp.pcsScope[meta.stepName] = step.Context

	for _, depend := range meta.depends {
		parent := step.scopeCtxs[depend.stepName]
		step.parents = append(step.parents, parent)
	}
	step.parents = append(step.parents, fp.Context)
	return &step
}

func (fp *RunProcess) Resume() {
	if PopStatus(&fp.status, Pause) {
		fp.pause.Done()
	}
}

func (fp *RunProcess) Pause() {
	if AppendStatus(&fp.status, Pause) {
		fp.pause.Add(1)
	}
}

func (fp *RunProcess) Stop() {
	AppendStatus(&fp.status, Stop)
}

func (fp *RunProcess) flow() *Feature {
	AppendStatus(&fp.status, Running)
	for _, step := range fp.flowSteps {
		if step.layer == 1 {
			go fp.scheduleStep(step)
		}
	}

	go fp.finalize()

	feature := Feature{
		status:  &fp.status,
		finish:  &fp.finish,
		running: &fp.needRun,
	}

	return &feature
}

func (fp *RunProcess) scheduleStep(step *RunStep) {
	defer fp.scheduleNextSteps(step)

	if _, finish := step.GetStepResult(step.stepName); finish {
		AppendStatus(&step.status, Success)
		return
	}
	status := atomic.LoadInt64(&fp.status)
	// If a step fails, the process fails.
	// But other tasks that do not depend on the failed task will continue to execute
	if !IsStatusNormal(status) {
		AppendStatus(&step.status, Cancel)
		return
	}

	// if prev step is abnormal, the current step will mark as cancel
	if status = atomic.LoadInt64(&step.status); !IsStatusNormal(status) {
		return
	}

	for status = atomic.LoadInt64(&fp.status); status&Pause == Pause; {
		fp.pause.Wait()
		status = atomic.LoadInt64(&fp.status)
	}

	go fp.runStep(step)

	timeout := 3 * time.Hour
	if fp.conf != nil && fp.conf.StepTimeout != 0 {
		timeout = fp.conf.StepTimeout
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

func (fp *RunProcess) runStep(step *RunStep) {
	step.Start = time.Now().UTC()

	if fp.conf != nil && len(fp.conf.PreProcessors) > 0 {
		for _, processor := range fp.conf.PreProcessors {
			if !processor(fp.summaryStepInfo(step)) {
				break
			}
		}
	}

	retry := 1
	if fp.conf != nil && fp.conf.StepRetry > 0 {
		retry = fp.conf.StepRetry
	}
	if step.config != nil && step.config.MaxRetry > 0 {
		retry = step.config.MaxRetry
	}

	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("panic: %v\n\n%s", r, string(debug.Stack()))
			AppendStatus(&step.status, Panic)
			step.Err = panicErr
			step.End = time.Now().UTC()
			step.finish <- true
		}
		if fp.conf == nil || len(fp.conf.PostProcessors) == 0 {
			step.finish <- true
			return
		}
		for _, processor := range fp.conf.PostProcessors {
			if !processor(fp.summaryStepInfo(step)) {
				break
			}
		}
		step.finish <- true
	}()

	for i := 0; i < retry; i++ {
		result, err := step.run(step.Context)
		step.End = time.Now().UTC()
		step.Err = err
		step.Set(step.stepName, result)
		if err != nil {
			AppendStatus(&step.status, Error)
			continue
		}
		fp.setStepResult(step.stepName, result)
		AppendStatus(&step.status, Success)
		break
	}
}

func (fp *RunProcess) finalize() {
	finish := make(chan bool, 1)
	go func() {
		fp.needRun.Wait()
		info := &ProcessInfo{
			Id:      fp.id,
			FlowId:  fp.flowId,
			Name:    fp.processName,
			StepMap: make(map[string]*StepInfo, len(fp.flowSteps)),
			Ctx:     fp.Context,
		}
		for name, step := range fp.flowSteps {
			info.StepMap[name] = fp.summaryStepInfo(step)
		}
		if fp.conf == nil || len(fp.conf.CompleteProcessors) == 0 {
			finish <- true
			return
		}
		for _, processor := range fp.conf.CompleteProcessors {
			if !processor(info) {
				break
			}
		}
		finish <- true
	}()

	timeout := 3 * time.Hour
	if fp.conf != nil && fp.conf.ProcessTimeout != 0 {
		timeout = fp.conf.ProcessTimeout
	}

	timer := time.NewTimer(timeout)
	select {
	case <-timer.C:
		AppendStatus(&fp.status, Timeout)
	case <-finish:
	}

	for _, step := range fp.flowSteps {
		AppendStatus(&fp.status, step.status)
	}
	if IsStatusNormal(fp.status) {
		AppendStatus(&fp.status, Success)
	}
	atomic.StoreInt64(&fp.finish, 1)
}

func (fp *RunProcess) summaryStepInfo(step *RunStep) *StepInfo {
	info := &StepInfo{
		Id:        step.id,
		ProcessId: step.processId,
		FlowId:    step.flowId,
		Name:      step.stepName,
		Status:    step.status,
		Ctx:       step.Context,
		Config:    step.config,
		Start:     step.Start,
		End:       step.End,
		Err:       step.Err,
		Prev:      make(map[string]string, len(step.depends)),
		Next:      make(map[string]string, len(step.waiters)),
	}
	for _, prev := range step.depends {
		info.Prev[prev.stepName] = fp.flowSteps[prev.stepName].id
	}

	for _, next := range step.waiters {
		info.Next[next.stepName] = fp.flowSteps[next.stepName].id
	}

	return info
}

func (fp *RunProcess) scheduleNextSteps(step *RunStep) {
	// all step not execute should cancel while process is timeout
	cancel := !IsStatusNormal(step.status) || !IsStatusNormal(fp.status)
	for _, waiter := range step.waiters {
		next := fp.flowSteps[waiter.stepName]
		current := atomic.AddInt64(&next.waiting, -1)
		if cancel {
			AppendStatus(&next.status, Cancel)
		}
		if atomic.LoadInt64(&current) != 0 {
			continue
		}
		if step.status&Success == Success {
			AppendStatus(&next.status, Running)
		}
		go fp.scheduleStep(next)
	}

	fp.needRun.Done()
}

func (fp *RunProcess) SkipFinishedStep(name string, result any) {
	fp.setStepResult(name, result)
}
