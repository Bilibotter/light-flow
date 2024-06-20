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
	nodeRouter
	belong   *FlowMeta
	init     sync.Once
	name     string
	steps    map[string]*StepMeta
	tailStep string
	nodeNum  int
}

type ProcessConfig struct {
	StepConfig
	ProcTimeout       time.Duration
	ProcNotUseDefault bool
	stepChain         handler[Step]    `flow:"skip"`
	procChain         handler[Process] `flow:"skip"`
}

func newProcessConfig() ProcessConfig {
	config := ProcessConfig{}
	return config
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

func (pc *ProcessConfig) BeforeStep(must bool, callback func(Step) (keepOn bool, err error)) *callback[Step] {
	return pc.stepChain.addCallback(Before, must, callback)
}

func (pc *ProcessConfig) AfterStep(must bool, callback func(Step) (keepOn bool, err error)) *callback[Step] {
	return pc.stepChain.addCallback(After, must, callback)
}

func (pc *ProcessConfig) BeforeProcess(must bool, callback func(i Process) (keepOn bool, err error)) *callback[Process] {
	return pc.procChain.addCallback(Before, must, callback)
}

func (pc *ProcessConfig) AfterProcess(must bool, callback func(Process) (keepOn bool, err error)) *callback[Process] {
	return pc.procChain.addCallback(After, must, callback)
}

func (pm *ProcessMeta) Name() string {
	return pm.name
}

func (pm *ProcessMeta) register() {
	_, load := allProcess.LoadOrStore(pm.name, pm)
	if load {
		panic(fmt.Sprintf("Process[%s] has exist", pm.name))
	}
}

func (pm *ProcessMeta) initialize() {
	pm.init.Do(func() {
		pm.constructVisible()
	})
}

func (pm *ProcessMeta) constructVisible() {
	for _, step := range pm.sortedSteps() {
		for _, waiter := range step.waiters {
			waiter.nodePath = step.nodePath | waiter.nodePath
		}
	}
}

// Merge will not merge config,
// because has not effective design to not use merged config.
func (pm *ProcessMeta) Merge(name string) {
	wrap, exist := allProcess.Load(name)
	if !exist {
		panic(fmt.Sprintf("can't merge not exist Process[%s]", name))
	}
	mergedProcess := wrap.(*ProcessMeta)
	for _, merge := range mergedProcess.sortedSteps() {
		if _, exist = pm.steps[merge.name]; exist {
			pm.mergeStep(merge)
			continue
		}
		depends := make([]any, 0, len(merge.depends))
		for _, depend := range merge.depends {
			depends = append(depends, depend.name)
		}
		step := pm.NameStep(merge.run, merge.name, depends...)
		step.position.append(mergedE)
	}

	// ensure step index bigger than all depends index
	for index, step := range pm.sortedSteps() {
		step.index = int64(index)
		step.nodePath = 1 << index
		pm.toName[step.index] = step.name
		pm.toIndex[step.name] = step.index
	}
}

func (pm *ProcessMeta) mergeStep(merge *StepMeta) {
	target := pm.steps[merge.name]

	for k, v := range merge.priority {
		if _, exist := target.priority[k]; exist {
			continue
		}
		target.priority[k] = v
	}

	// create a set contains all depended on target name
	current := createSetBySliceFunc[*StepMeta](target.depends,
		func(meta *StepMeta) string { return meta.name })

	stepNames := make([]string, 0, len(merge.depends))
	for _, step := range merge.depends {
		stepNames = append(stepNames, step.name)
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
			panic(fmt.Sprintf("merge failed, a circle is formed between Step[%s] and Step[%s].",
				depend.name, target.name))
		}
		if depend.layer+1 > target.layer {
			target.layer = depend.layer + 1
			pm.updateWaitersLayer(target)
		}
		target.depends = append(target.depends, depend)
	}

	target.wireDepends()
	pm.tailStep = target.name
}

func (pm *ProcessMeta) updateWaitersLayer(step *StepMeta) {
	for _, waiters := range step.waiters {
		if step.layer+1 > waiters.layer {
			waiters.layer = step.layer + 1
			pm.updateWaitersLayer(waiters)
		}
	}
}

func (pm *ProcessMeta) Step(run func(ctx Step) (any, error), depends ...any) *StepMeta {
	return pm.NameStep(run, getFuncName(run), depends...)
}

func (pm *ProcessMeta) Tail(run func(ctx Step) (any, error), alias ...string) *StepMeta {
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

func (pm *ProcessMeta) NameStep(run func(ctx Step) (any, error), name string, depends ...any) *StepMeta {
	meta := &StepMeta{
		name: name,
	}
	for _, wrap := range depends {
		dependName := toStepName(wrap)
		depend, exist := pm.steps[dependName]
		if !exist {
			panic(fmt.Sprintf("Step[%s]'s depend[%s] not found.", name, dependName))
		}
		meta.depends = append(meta.depends, depend)
	}

	if old, exist := pm.steps[name]; exist {
		if !old.position.Has(mergedE) {
			panic(fmt.Sprintf("Step[%s] already exist, can used %s to avoid name duplicate",
				name, getFuncName(pm.NameStep)))
		}
		pm.mergeStep(meta)
		return old
	}

	meta.belong = pm
	meta.run = run
	meta.layer = 1
	meta.wireDepends()
	pm.addRouterInfo(meta)

	pm.tailStep = meta.name
	pm.steps[name] = meta

	return meta
}

func (pm *ProcessMeta) addRouterInfo(step *StepMeta) {
	if pm.nodeNum == 62 {
		panic(fmt.Sprintf("Process[%s] exceeds max nodes num, max node num is 62", pm.name))
	}
	step.nodeRouter = nodeRouter{
		nodePath: 1 << pm.nodeNum,
		index:    int64(pm.nodeNum),
		toName:   pm.toName,
		toIndex:  pm.toIndex,
	}
	pm.nodeNum++
	pm.toName[step.index] = step.name
	pm.toIndex[step.name] = step.index
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
