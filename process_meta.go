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
	ProcessConfig
	accessInfo
	belong      *FlowMeta
	init        sync.Once
	processName string
	steps       map[string]*StepMeta
	tailStep    string
	nodeNum     int
}

type Process struct {
	basicInfo
	procCtx
	FlowId string
}

type ProcessConfig struct {
	StepConfig
	ProcTimeout       time.Duration
	ProcNotUseDefault bool
	stepChain         handler[*Step]    `flow:"skip"`
	procChain         handler[*Process] `flow:"skip"`
}

func newProcessConfig() ProcessConfig {
	config := ProcessConfig{}
	return config
}

// Fix ContextName return first step name
func (pi *Process) ContextName() string {
	return pi.Name
}

func (pc *ProcessConfig) clone() ProcessConfig {
	config := ProcessConfig{}
	copyPropertiesSkipNotEmpty(pc, config)
	config.StepConfig = pc.StepConfig.clone()
	config.stepChain = pc.stepChain.clone()
	config.procChain = pc.procChain.clone()
	return config
}

func (pc *ProcessConfig) combine(config *ProcessConfig) {
	copyPropertiesSkipNotEmpty(pc, config)
	pc.StepConfig.combine(&config.StepConfig)
	pc.stepChain.combine(&config.stepChain)
	pc.procChain.combine(&config.procChain)
}

func (pc *ProcessConfig) ProcessTimeout(timeout time.Duration) *ProcessConfig {
	pc.ProcTimeout = timeout
	return pc
}

func (pc *ProcessConfig) NotUseDefault() {
	pc.ProcNotUseDefault = true
}

func (pc *ProcessConfig) StepsRetry(retry int) *ProcessConfig {
	pc.StepConfig.StepRetry = retry
	return pc
}

func (pc *ProcessConfig) StepsTimeout(timeout time.Duration) *ProcessConfig {
	pc.StepConfig.StepTimeout = timeout
	return pc
}

func (pc *ProcessConfig) BeforeStep(must bool, callback func(*Step) (keepOn bool, err error)) *callback[*Step] {
	return pc.stepChain.addCallback(Before, must, callback)
}

func (pc *ProcessConfig) AfterStep(must bool, callback func(*Step) (keepOn bool, err error)) *callback[*Step] {
	return pc.stepChain.addCallback(After, must, callback)
}

func (pc *ProcessConfig) BeforeProcess(must bool, callback func(*Process) (keepOn bool, err error)) *callback[*Process] {
	return pc.procChain.addCallback(Before, must, callback)
}

func (pc *ProcessConfig) AfterProcess(must bool, callback func(*Process) (keepOn bool, err error)) *callback[*Process] {
	return pc.procChain.addCallback(After, must, callback)
}

func (pm *ProcessMeta) register() {
	_, load := allProcess.LoadOrStore(pm.processName, pm)
	if load {
		panic(fmt.Sprintf("process[%s] has exist", pm.processName))
	}
}

func (pm *ProcessMeta) clone() *ProcessMeta {
	meta := &ProcessMeta{
		ProcessConfig: pm.ProcessConfig.clone(),
		init:          sync.Once{},
		processName:   pm.processName,
		tailStep:      pm.tailStep,
		nodeNum:       pm.nodeNum,
	}
	meta.steps = make(map[string]*StepMeta, len(pm.steps))
	for k, v := range pm.steps {
		meta.steps[k] = v
	}
	return meta
}

// todo:delete it
func (pm *ProcessMeta) initialize() {
	pm.init.Do(func() {
		pm.constructVisible()
	})
}

func (pm *ProcessMeta) constructVisible() {
	for _, step := range pm.sortedSteps() {
		for _, waiter := range step.waiters {
			waiter.passing = step.passing | waiter.passing
		}
	}
	return
}

// Merge will not merge config,
// because has not effective design to not use merged config.
func (pm *ProcessMeta) Merge(name string) {
	wrap, exist := allProcess.Load(name)
	if !exist {
		panic(fmt.Sprintf("can't merge not exist process[%s]", name))
	}
	mergedProcess := wrap.(*ProcessMeta)
	for _, merge := range mergedProcess.sortedSteps() {
		if _, exist = pm.steps[merge.stepName]; exist {
			pm.mergeStep(merge)
			continue
		}
		depends := make([]any, 0, len(merge.depends))
		for _, depend := range merge.depends {
			depends = append(depends, depend.stepName)
		}
		step := pm.NameStep(merge.run, merge.stepName, depends...)
		step.position.set(mergedE)
	}

	// ensure step index bigger than all depends index
	for index, step := range pm.sortedSteps() {
		step.index = int64(index)
		step.passing = 1 << index
		pm.names[step.index] = step.stepName
		pm.indexes[step.stepName] = step.index
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
	current := createSetBySliceFunc[*StepMeta](target.depends,
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

func (pm *ProcessMeta) Step(run func(ctx StepCtx) (any, error), depends ...any) *StepMeta {
	return pm.NameStep(run, getFuncName(run), depends...)
}

func (pm *ProcessMeta) Tail(run func(ctx StepCtx) (any, error), alias ...string) *StepMeta {
	depends := make([]any, 0)
	for name, step := range pm.steps {
		if step.position.Has(endE) {
			depends = append(depends, name)
		}
	}
	if len(alias) == 1 {
		return pm.NameStep(run, alias[0], depends...)
	}
	return pm.Step(run, depends...)
}

func (pm *ProcessMeta) NameStep(run func(ctx StepCtx) (any, error), name string, depends ...any) *StepMeta {
	meta := &StepMeta{
		stepName: name,
	}
	for _, wrap := range depends {
		dependName := toStepName(wrap)
		depend, exist := pm.steps[dependName]
		if !exist {
			panic(fmt.Sprintf("step[%s]'s depend[%s] not found.]", name, dependName))
		}
		meta.depends = append(meta.depends, depend)
	}

	if old, exist := pm.steps[name]; exist {
		if !old.position.Has(mergedE) {
			panic(fmt.Sprintf("step named [%s] already exist, can used %s to avoid stepName duplicate",
				name, getFuncName(pm.NameStep)))
		}
		pm.mergeStep(meta)
		return old
	}

	meta.belong = pm
	meta.run = run
	meta.layer = 1
	meta.wireDepends()
	pm.addPassingInfo(meta)

	pm.tailStep = meta.stepName
	pm.steps[name] = meta

	return meta
}

func (pm *ProcessMeta) addPassingInfo(step *StepMeta) {
	if pm.nodeNum == 62 {
		panic(fmt.Sprintf("step[%s] exceeds max nodes num, max node num is 62", step.stepName))
	}
	step.accessInfo = accessInfo{
		passing: 1 << pm.nodeNum,
		index:   int64(pm.nodeNum),
		names:   pm.names,
		indexes: pm.indexes,
	}
	pm.names[step.index] = step.stepName
	pm.indexes[step.stepName] = step.index
	pm.nodeNum++
}

func (pm *ProcessMeta) sortedSteps() []*StepMeta {
	steps := make([]*StepMeta, 0, len(pm.steps))
	for _, step := range pm.steps {
		steps = append(steps, step)
	}
	sort.SliceStable(steps, func(i, j int) bool {
		return steps[i].layer < steps[j].layer
	})
	return steps
}
