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

type ProcessInfo struct {
	Id      string
	FlowId  string
	Name    string
	Status  int64
	StepMap map[string]*StepInfo
	Ctx     *Context
}

type ProcProcessor struct {
	// If keepOn is false, the next processors will not be executed
	Before func(info *ProcessInfo) (keepOn bool)
	// If execute success, abnormal will be nil
	After func(info *ProcessInfo) (keepOn bool)
	// If Must is false, the process will continue to execute
	// even if the processor fails
	Must bool
}

type StepProcessor struct {
	// If keepOn is false, the next processors will not be executed
	Before func(info *StepInfo) (keepOn bool)
	// If execute success, abnormal will be nil
	After func(info *StepInfo) (keepOn bool)
	// If Must is false, the process will continue to execute,
	// even if the processor fails
	Must bool
}

type ProcessConfig struct {
	StepRetry    int
	StepTimeout  time.Duration
	ProcTimeout  time.Duration
	stepCallback []*StepProcessor
	procCallback []*ProcProcessor
}

func SetDefaultProcessConfig(config *ProcessConfig) {
	lock.Lock()
	defer lock.Unlock()
	defaultProcessConfig = config
}

func (pc *ProcessConfig) AddBeforeStep(must bool, processor func(info *StepInfo) (keepOn bool)) {
	before := &StepProcessor{
		Must:   must,
		Before: processor,
	}
	pc.stepCallback = append(pc.stepCallback, before)
	sort.SliceStable(pc.stepCallback, func(i, j int) bool {
		if pc.stepCallback[i].Must == pc.stepCallback[j].Must {
			return i < j
		}
		return pc.stepCallback[i].Must
	})
}

func (pc *ProcessConfig) AddAfterStep(must bool, processor func(info *StepInfo) (keepOn bool)) {

	after := &StepProcessor{
		Must:  must,
		After: processor,
	}
	pc.stepCallback = append(pc.stepCallback, after)
	sort.SliceStable(pc.stepCallback, func(i, j int) bool {
		if pc.stepCallback[i].Must == pc.stepCallback[j].Must {
			return i < j
		}
		return pc.stepCallback[i].Must
	})
}

func (pc *ProcessConfig) AddBeforeProcess(must bool, processor func(info *ProcessInfo) (keepOn bool)) {
	before := &ProcProcessor{
		Must:   must,
		Before: processor,
	}
	pc.procCallback = append(pc.procCallback, before)
	sort.SliceStable(pc.procCallback, func(i, j int) bool {
		if pc.procCallback[i].Must == pc.procCallback[j].Must {
			return i < j
		}
		return pc.procCallback[i].Must
	})
}

func (pc *ProcessConfig) AddAfterProcess(must bool, processor func(info *ProcessInfo) (keepOn bool)) {
	after := &ProcProcessor{
		Must:  must,
		After: processor,
	}
	pc.procCallback = append(pc.procCallback, after)
	sort.SliceStable(pc.procCallback, func(i, j int) bool {
		if pc.procCallback[i].Must == pc.procCallback[j].Must {
			return i < j
		}
		return pc.procCallback[i].Must
	})
}

func (pp *ProcProcessor) callback(flag string, info *ProcessInfo) (keepOn bool, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if !pp.Must {
			return
		}
		err = fmt.Errorf("panic: %v\n\n%s", r, string(debug.Stack()))
	}()

	switch flag {
	case "before":
		if pp.Before != nil && !pp.Before(info) {
			return
		}
	case "after":
		if pp.After != nil && !pp.After(info) {
			return
		}
	}

	return true, nil
}

func (sp *StepProcessor) callback(flag string, info *StepInfo) (keepOn bool, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if !sp.Must {
			return
		}
		err = fmt.Errorf("panic: %v\n\n%s", r, string(debug.Stack()))
	}()

	switch flag {
	case "before":
		if sp.Before != nil && !sp.Before(info) {
			return
		}
	case "after":
		if sp.After != nil && !sp.After(info) {
			return
		}
	}

	return true, nil
}

func (pi *ProcessInfo) Success() bool {
	return pi.Status&Success == Success && IsStatusNormal(atomic.LoadInt64(&pi.Status))
}

func (pi *ProcessInfo) Exceptions() []string {
	if pi.Success() {
		return nil
	}
	return ExplainStatus(atomic.LoadInt64(&pi.Status))
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

func (pm *ProcessMeta) AddBeforeStep(must bool, processor func(info *StepInfo) (keepOn bool)) {
	pm.conf.AddBeforeStep(must, processor)
}

func (pm *ProcessMeta) AddAfterStep(must bool, processor func(info *StepInfo) (keepOn bool)) {
	pm.conf.AddAfterStep(must, processor)
}

func (pm *ProcessMeta) AddBeforeProcess(must bool, processor func(info *ProcessInfo) (keepOn bool)) {
	pm.conf.AddBeforeProcess(must, processor)
}

func (pm *ProcessMeta) AddAfterProcess(must bool, processor func(info *ProcessInfo) (keepOn bool)) {
	pm.conf.AddAfterProcess(must, processor)
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
