package light_flow

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	Before = "Before"
	After  = "After"
)

type ProcessMeta struct {
	*ProcessConfig
	init        sync.Once
	processName string
	steps       map[string]*StepMeta
	tailStep    string
}

type ProcessInfo struct {
	*BasicInfo
	FlowId string
	Ctx    *Context
}

type ProcessConfig struct {
	StepRetry     int
	StepTimeout   time.Duration
	ProcTimeout   time.Duration
	notUseDefault bool
	stepChain     *CallbackChain[*StepInfo]
	procChain     *CallbackChain[*ProcessInfo]
}

func (pc *ProcessConfig) merge(merged *ProcessConfig) *ProcessConfig {
	CopyPropertiesSkipNotEmpty(merged, pc)
	if merged.stepChain != nil {
		pc.stepChain.filters = append(merged.stepChain.CopyChain(), pc.stepChain.filters...)
		pc.stepChain.sort()
	}
	if merged.procChain != nil {
		pc.procChain.filters = append(merged.procChain.CopyChain(), pc.procChain.filters...)
		pc.procChain.sort()
	}
	return pc
}

func (pc *ProcessConfig) addStepCallback(flag string, must bool, callback func(*StepInfo) bool) *Callback[*StepInfo] {
	if pc.stepChain == nil {
		pc.stepChain = &CallbackChain[*StepInfo]{}
	}
	return pc.stepChain.Add(flag, must, callback)
}

func (pc *ProcessConfig) addProcessCallback(flag string, must bool, callback func(*ProcessInfo) bool) *Callback[*ProcessInfo] {
	if pc.procChain == nil {
		pc.procChain = &CallbackChain[*ProcessInfo]{}
	}
	return pc.procChain.Add(flag, must, callback)
}

func (pm *ProcessMeta) register() {
	_, load := allProcess.LoadOrStore(pm.processName, pm)
	if load {
		panic(fmt.Sprintf("process[%s] has exist", pm.processName))
	}
}

func (pm *ProcessMeta) initialize() {
	pm.init.Do(func() {
		if pm.notUseDefault {
			return
		}
		if defaultConfig == nil || defaultConfig.ProcessConfig == nil {
			return
		}
		if pm.ProcessConfig == nil {
			pm.ProcessConfig = defaultConfig.ProcessConfig
			return
		}
		pm.ProcessConfig.merge(defaultConfig.ProcessConfig)
	})
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
		if !contains(&step.position, HasNext) {
			depends = append(depends, name)
		}
	}

	return pm.AddStepWithAlias(alias, run, depends...)
}

func (pm *ProcessMeta) AddStepWithAlias(alias string, run func(ctx *Context) (any, error), depends ...any) *StepMeta {
	var meta *StepMeta
	var oldDepends *Set

	if old, exist := pm.steps[alias]; exist {
		if !contains(&pm.steps[alias].position, Merged) {
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
		if contains(&depend.position, End) {
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

func (pm *ProcessMeta) NotUseDefault() {
	pm.notUseDefault = true
}

func (pm *ProcessMeta) AddBeforeStep(must bool, callback func(*StepInfo) (keepOn bool)) *Callback[*StepInfo] {
	return pm.addStepCallback(Before, must, callback)
}

func (pm *ProcessMeta) AddAfterStep(must bool, callback func(*StepInfo) (keepOn bool)) *Callback[*StepInfo] {
	return pm.addStepCallback(After, must, callback)
}

func (pm *ProcessMeta) AddBeforeProcess(must bool, callback func(*ProcessInfo) (keepOn bool)) *Callback[*ProcessInfo] {
	return pm.addProcessCallback(Before, must, callback)
}

func (pm *ProcessMeta) AddAfterProcess(must bool, callback func(*ProcessInfo) (keepOn bool)) *Callback[*ProcessInfo] {
	return pm.addProcessCallback(After, must, callback)
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
