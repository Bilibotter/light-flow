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
	processConfig
	visitor
	init        sync.Once
	processName string
	steps       map[string]*StepMeta
	tailStep    string
	nodeNum     int
}

type ProcessInfo struct {
	basicInfo
	*visibleContext
	FlowId string
}

type processConfig struct {
	stepConfig
	procTimeout   time.Duration
	notUseDefault bool
	stepChain     callbackChain[*StepInfo]
	procChain     callbackChain[*ProcessInfo]
}

func newProcessConfig() processConfig {
	config := processConfig{}
	return config
}

func (pc *processConfig) merge(merged *processConfig) *processConfig {
	CopyPropertiesSkipNotEmpty(merged, pc)
	pc.stepChain.filters = append(merged.stepChain.copyChain(), pc.stepChain.filters...)
	pc.stepChain.maintain()
	pc.procChain.filters = append(merged.procChain.copyChain(), pc.procChain.filters...)
	pc.procChain.maintain()
	return pc
}

func (pc *processConfig) ProcessTimeout(timeout time.Duration) *processConfig {
	pc.procTimeout = timeout
	return pc
}

func (pc *processConfig) StepsRetry(retry int) *processConfig {
	pc.stepConfig.stepRetry = retry
	return pc
}

func (pc *processConfig) StepsTimeout(timeout time.Duration) *processConfig {
	pc.stepConfig.stepTimeout = timeout
	return pc
}

func (pc *processConfig) BeforeStep(must bool, callback func(*StepInfo) (keepOn bool, err error)) *callback[*StepInfo] {
	return pc.stepChain.addCallback(Before, must, callback)
}

func (pc *processConfig) AfterStep(must bool, callback func(*StepInfo) (keepOn bool, err error)) *callback[*StepInfo] {
	return pc.stepChain.addCallback(After, must, callback)
}

func (pc *processConfig) BeforeProcess(must bool, callback func(*ProcessInfo) (keepOn bool, err error)) *callback[*ProcessInfo] {
	return pc.procChain.addCallback(Before, must, callback)
}

func (pc *processConfig) AfterProcess(must bool, callback func(*ProcessInfo) (keepOn bool, err error)) *callback[*ProcessInfo] {
	return pc.procChain.addCallback(After, must, callback)
}

func (pm *ProcessMeta) register() {
	_, load := allProcess.LoadOrStore(pm.processName, pm)
	if load {
		panic(fmt.Sprintf("process[%s] has exist", pm.processName))
	}
}

func (pm *ProcessMeta) initialize() {
	pm.init.Do(func() {
		pm.completeVisitorInfo()
		if pm.notUseDefault {
			return
		}
		if defaultConfig == nil {
			return
		}
		pm.processConfig.merge(&defaultConfig.processConfig)
	})
}

func (pm *ProcessMeta) completeVisitorInfo() {
	for _, step := range pm.sortedStep() {
		for _, waiter := range step.waiters {
			waiter.visible = step.visible | waiter.visible
		}
	}
	return
}

func (pm *ProcessMeta) Merge(name string) {
	wrap, exist := allProcess.Load(name)
	if !exist {
		panic(fmt.Sprintf("can't merge not exist process[%s]", name))
	}
	mergedProcess := wrap.(*ProcessMeta)
	for _, merge := range mergedProcess.sortedStep() {
		if _, exist = pm.steps[merge.stepName]; exist {
			pm.mergeStep(merge)
			continue
		}
		depends := make([]any, 0, len(merge.depends))
		for _, depend := range merge.depends {
			depends = append(depends, depend.stepName)
		}
		step := pm.AliasStep(merge.stepName, merge.run, depends...)
		step.position.Append(Merged)
	}
}

func (pm *ProcessMeta) mergeStep(merge *StepMeta) {
	target := pm.steps[merge.stepName]

	for k, v := range merge.priority {
		if _, exist := target.priority[k]; exist {
			continue
		}
		target.priority[k] = v
	}

	// create a set contains all depended on target flowName
	current := CreateSetBySliceFunc[*StepMeta](target.depends,
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

func (pm *ProcessMeta) Step(run func(ctx Context) (any, error), depends ...any) *StepMeta {
	return pm.AliasStep(GetFuncName(run), run, depends...)
}

func (pm *ProcessMeta) Tail(alias string, run func(ctx Context) (any, error)) *StepMeta {
	depends := make([]any, 0)
	for name, step := range pm.steps {
		if step.position.Contain(End) {
			depends = append(depends, name)
		}
	}

	return pm.AliasStep(alias, run, depends...)
}

func (pm *ProcessMeta) AliasStep(alias string, run func(ctx Context) (any, error), depends ...any) *StepMeta {
	meta := &StepMeta{
		stepName: alias,
	}
	for _, wrap := range depends {
		name := toStepName(wrap)
		depend, exist := pm.steps[name]
		if !exist {
			panic(fmt.Sprintf("can't find step[%s]", name))
		}
		meta.depends = append(meta.depends, depend)
	}

	if old, exist := pm.steps[alias]; exist {
		if !old.position.Contain(Merged) {
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
	pm.addVisitorInfo(meta)

	pm.tailStep = meta.stepName
	pm.steps[alias] = meta

	return meta
}

func (pm *ProcessMeta) NotUseDefault() {
	pm.notUseDefault = true
}

func (pm *ProcessMeta) addVisitorInfo(step *StepMeta) {
	if pm.nodeNum == 62 {
		panic(fmt.Sprintf("step[%s] exceeds max nodes num, max node num is 62", step.stepName))
	}
	step.visitor = visitor{
		visible: 1 << pm.nodeNum,
		index:   int32(pm.nodeNum),
		roster:  pm.roster,
	}
	pm.roster[step.index] = step.stepName
	pm.nodeNum++
}

func (pm *ProcessMeta) sortedStep() []*StepMeta {
	steps := make([]*StepMeta, 0, len(pm.steps))
	for _, step := range pm.steps {
		steps = append(steps, step)
	}
	sort.SliceStable(steps, func(i, j int) bool {
		return steps[i].layer < steps[j].layer
	})
	return steps
}
