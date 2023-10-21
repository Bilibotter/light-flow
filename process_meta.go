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
	*processConfig
	init        sync.Once
	processName string
	steps       map[string]*StepMeta
	tailStep    string
}

type ProcessInfo struct {
	*BasicInfo
	*Context
	FlowId string
}

type processConfig struct {
	*StepConfig
	ProcTimeout   time.Duration
	notUseDefault bool
	stepChain     *CallbackChain[*StepInfo]
	procChain     *CallbackChain[*ProcessInfo]
}

func NewProcessConfig() *processConfig {
	stepChain := &CallbackChain[*StepInfo]{}
	procChain := &CallbackChain[*ProcessInfo]{}
	config := &processConfig{
		StepConfig: &StepConfig{},
		stepChain:  stepChain,
		procChain:  procChain,
	}
	return config
}

func (pc *processConfig) merge(merged *processConfig) *processConfig {
	CopyPropertiesSkipNotEmpty(merged, pc)
	if pc.stepChain == nil {
		pc.stepChain = &CallbackChain[*StepInfo]{}
	}
	if merged.stepChain != nil {
		pc.stepChain.filters = append(merged.stepChain.CopyChain(), pc.stepChain.filters...)
		pc.stepChain.maintain()
	}
	if pc.procChain == nil {
		pc.procChain = &CallbackChain[*ProcessInfo]{}
	}
	if merged.procChain != nil {
		pc.procChain.filters = append(merged.procChain.CopyChain(), pc.procChain.filters...)
		pc.procChain.maintain()
	}
	return pc
}

func (pc *processConfig) AddStepRetry(retry int) *processConfig {
	if pc.StepConfig == nil {
		pc.StepConfig = &StepConfig{}
	}
	pc.StepConfig.StepRetry = retry
	return pc
}

func (pc *processConfig) AddStepTimeout(timeout time.Duration) *processConfig {
	if pc.StepConfig == nil {
		pc.StepConfig = &StepConfig{}
	}
	pc.StepConfig.StepTimeout = timeout
	return pc
}

func (pc *processConfig) AddBeforeStep(must bool, callback func(*StepInfo) (keepOn bool, err error)) *Callback[*StepInfo] {
	return pc.stepChain.AddCallback(Before, must, callback)
}

func (pc *processConfig) AddAfterStep(must bool, callback func(*StepInfo) (keepOn bool, err error)) *Callback[*StepInfo] {
	return pc.stepChain.AddCallback(After, must, callback)
}

func (pc *processConfig) AddBeforeProcess(must bool, callback func(*ProcessInfo) (keepOn bool, err error)) *Callback[*ProcessInfo] {
	return pc.procChain.AddCallback(Before, must, callback)
}

func (pc *processConfig) AddAfterProcess(must bool, callback func(*ProcessInfo) (keepOn bool, err error)) *Callback[*ProcessInfo] {
	return pc.procChain.AddCallback(After, must, callback)
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
		if defaultConfig == nil || defaultConfig.processConfig == nil {
			return
		}
		if pm.processConfig == nil {
			pm.processConfig = defaultConfig.processConfig
			return
		}
		pm.processConfig.merge(defaultConfig.processConfig)
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
		step := pm.AliasStep(merge.stepName, merge.run, depends...)
		step.Append(Merged)
	}
}

func (pm *ProcessMeta) mergeStep(merge *StepMeta) {
	target := pm.steps[merge.stepName]

	for k, v := range merge.ctxPriority {
		if _, exist := target.ctxPriority[k]; exist {
			continue
		}
		target.ctxPriority[k] = v
	}

	// create a set contains all depended on target flowName
	current := CreateFromSliceFunc[*StepMeta](target.depends,
		func(meta *StepMeta) string { return meta.stepName })

	stepNames := make([]string, 0, len(merge.depends))
	for _, step := range merge.depends {
		stepNames = append(stepNames, step.stepName)
	}
	// Can't use sort stepNames instead, because stepNames order is context access order.
	sort.Slice(stepNames, func(i, j int) bool {
		return pm.steps[stepNames[i]].layer < pm.steps[stepNames[j]].layer
	})

	for _, name := range stepNames {
		if current.Contains(name) {
			continue
		}
		depend := pm.steps[name]
		if depend.layer > target.layer && target.forwardSearch(name) {
			panic(fmt.Sprintf("merge failed, a circle is formed between target[%s] and target[%s].",
				depend.stepName, target.stepName))
		}
		if depend.layer+1 > target.layer {
			target.layer = depend.layer + 1
			pm.updateWaitersLayer(target)
		}
		target.depends = append(target.depends, depend)
	}

	target.wireDepends()
	pm.tailStep = target.stepName
}

func (pm *ProcessMeta) updateWaitersLayer(step *StepMeta) {
	for _, waiters := range step.waiters {
		if step.layer+1 > waiters.layer {
			waiters.layer = step.layer + 1
			pm.updateWaitersLayer(waiters)
		}
	}
}

func (pm *ProcessMeta) Step(run func(ctx *Context) (any, error), depends ...any) *StepMeta {
	return pm.AliasStep(GetFuncName(run), run, depends...)
}

func (pm *ProcessMeta) Tail(alias string, run func(ctx *Context) (any, error)) *StepMeta {
	depends := make([]any, 0)
	for name, step := range pm.steps {
		if step.Contain(End) {
			depends = append(depends, name)
		}
	}

	return pm.AliasStep(alias, run, depends...)
}

func (pm *ProcessMeta) AliasStep(alias string, run func(ctx *Context) (any, error), depends ...any) *StepMeta {
	meta := &StepMeta{stepName: alias}
	for _, wrap := range depends {
		name := toStepName(wrap)
		depend, exist := pm.steps[name]
		if !exist {
			panic(fmt.Sprintf("can't find step[%s]", name))
		}
		meta.depends = append(meta.depends, depend)
	}

	if old, exist := pm.steps[alias]; exist {
		if !old.Contain(Merged) {
			panic(fmt.Sprintf("step named [%s] already exist, can used %s to avoid stepName duplicate",
				alias, GetFuncName(pm.AliasStep)))
		}
		pm.mergeStep(meta)
		return old
	}

	meta.belong = pm
	meta.run = run
	meta.layer = 1
	meta.wireDepends()

	pm.tailStep = meta.stepName
	pm.steps[alias] = meta

	return meta
}

func (pm *ProcessMeta) NotUseDefault() {
	pm.notUseDefault = true
}

func (pm *ProcessMeta) sortedStepMeta() []*StepMeta {
	steps := make([]*StepMeta, 0, len(pm.steps))
	for _, step := range pm.steps {
		steps = append(steps, step)
	}
	sort.SliceStable(steps, func(i, j int) bool {
		return steps[i].layer < steps[j].layer
	})
	return steps
}
