package flow

import (
	"fmt"
	"sort"
	"sync"
)

const (
	beforeF uint64 = 1 << iota
	afterF  uint64 = 1 << iota
	beforeS        = "Before"
	afterS         = "After"
)

type BuildChain interface {
	SyncPoint
	After(depends ...any) BuildChain
}

type SyncPoint interface {
	points() []string
}

type buildChain struct {
	*ProcessMeta
	isSerial bool
	group    []*StepMeta
}

type ProcessMeta struct {
	processConfig
	procCallback
	nodeRouter
	belong   *FlowMeta
	init     sync.Once
	name     string
	steps    map[string]*StepMeta
	tailStep string
	nodeNum  uint
}

func (bc *buildChain) After(depends ...any) BuildChain {
	if bc.isSerial {
		bc.buildSerial(bc.findSteps(depends...))
		return bc
	}
	bc.buildParallel(bc.findSteps(depends...))
	return bc
}

func (bc *buildChain) buildParallel(depends []*StepMeta) {
	for _, step := range bc.group {
		bc.addDepends(step, depends)
	}
}

func (bc *buildChain) buildSerial(depends []*StepMeta) {
	bc.addDepends(bc.group[0], depends)
}

func (bc *buildChain) addDepends(step *StepMeta, depends []*StepMeta) {
	current := createSetBySliceFunc[*StepMeta, string](step.depends, func(t *StepMeta) string {
		return t.name
	})
	for _, depend := range depends {
		if !current.Contains(depend.name) {
			step.depends = append(step.depends, depend)
		}
	}
	step.wireDepends()
}

func (bc *buildChain) findSteps(depends ...any) []*StepMeta {
	ds := make([]*StepMeta, 0)
	names := bc.normalize(depends...)
	for _, name := range names {
		meta, ok := bc.steps[name]
		if !ok {
			panic(fmt.Sprintf("[Step: %s] not found in process [Process: %s]", name, bc.name))
		}
		ds = append(ds, meta)
	}
	return ds
}

func (bc *buildChain) points() []string {
	points := make([]string, len(bc.group))
	for i, step := range bc.group {
		points[i] = step.name
	}
	return points
}

func (pm *ProcessMeta) Name() string {
	return pm.name
}

func (pm *ProcessMeta) Flow() *FlowMeta {
	return pm.belong
}

func (pm *ProcessMeta) register() {
	_, load := allProcess.LoadOrStore(pm.name, pm)
	if load {
		panic(fmt.Sprintf("[Process: %s] has exist", pm.name))
	}
}

func (pm *ProcessMeta) initialize() {
	pm.init.Do(func() {
		pm.constructVisible()
		copyProperties(&pm.belong.processConfig, &pm.processConfig, true)
		copyProperties(&pm.belong.stepConfig, &pm.stepConfig, true)
		for _, step := range pm.steps {
			if pm.stepCfgInit {
				copyProperties(&pm.stepConfig, &step.stepConfig, true)
			}
			for _, cfg := range step.extern {
				copyProperties(cfg, &step.stepConfig, true)
			}
		}
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
	wrap, find := allProcess.Load(name)
	if !find {
		panic(fmt.Sprintf("can't merge not exist [Process: %s]", name))
	}
	mergedProcess := wrap.(*ProcessMeta)
	for _, merged := range mergedProcess.sortedSteps() {
		if target, exist := pm.steps[merged.name]; exist {
			pm.mergeStep(merged)
			if merged.stepCfgInit {
				target.extern = append(target.extern, &merged.stepConfig)
			}
			continue
		}
		depends := make([]any, 0, len(merged.depends))
		for _, depend := range merged.depends {
			depends = append(depends, depend.name)
		}
		step := pm.CustomStep(merged.run, merged.name, depends...)
		step.restrict = merged.restrict
		step.condition = merged.condition
		step.position.append(mergedE)
		if merged.stepCfgInit {
			step.extern = append(step.extern, &merged.stepConfig)
		}
	}

	// ensure step index bigger than all depends index
	for index, step := range pm.sortedSteps() {
		step.index = uint64(index)
		step.nodePath = 1 << index
		pm.toName[step.index] = step.name
		pm.toIndex[step.name] = step.index
	}
}

func (pm *ProcessMeta) mergeStep(merge *StepMeta) {
	target := pm.steps[merge.name]

	for k, v := range merge.restrict {
		if _, exist := target.restrict[k]; exist {
			continue
		}
		target.restrict[k] = v
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
			panic(fmt.Sprintf("merge failed, a circle is formed between [Step: %s] and [Step: %s].",
				depend.name, target.name))
		}
		if depend.layer+1 > target.layer {
			target.layer = depend.layer + 1
			pm.updateWaitersLayer(target)
		}
		target.depends = append(target.depends, depend)
	}

	target.wireDepends()
	target.mergeCond(&merge.condition)
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

// This function was inspired by the liteflow project (https://github.com/dromara/liteflow),
// which implements similar functionality in Java. The implementation here is independently
// developed in Go, with a different approach and code structure.
func (pm *ProcessMeta) Follow(steps ...func(ctx Step) (any, error)) BuildChain {
	if len(steps) == 0 {
		panic(fmt.Sprintf("serial steps can't be empty"))
	}
	group := make([]*StepMeta, len(steps))
	group[0] = pm.CustomStep(steps[0], getFuncName(steps[0]))
	prev := group[0].name
	for i := 1; i < len(steps); i++ {
		group[i] = pm.CustomStep(steps[i], getFuncName(steps[i]), prev)
		prev = group[i].name
	}
	chain := &buildChain{
		ProcessMeta: pm,
		group:       group,
		isSerial:    true,
	}
	return chain
}

// This function was inspired by the liteflow project (https://github.com/dromara/liteflow),
// which implements similar functionality in Java. The implementation here is independently
// developed in Go, with a different approach and code structure.
func (pm *ProcessMeta) Parallel(steps ...func(ctx Step) (any, error)) BuildChain {
	if len(steps) == 0 {
		panic(fmt.Sprintf("[Process: %s] parallel steps is empty", pm.name))
	}
	group := make([]*StepMeta, len(steps))
	for i, step := range steps {
		group[i] = pm.CustomStep(step, getFuncName(step))
	}
	chain := &buildChain{
		ProcessMeta: pm,
		group:       group,
	}
	return chain
}

func (pm *ProcessMeta) SyncAll(run func(ctx Step) (any, error), alias string) *StepMeta {
	depends := make([]any, 0)
	for name, step := range pm.steps {
		if step.position.Has(endE) {
			depends = append(depends, name)
		}
	}
	return pm.CustomStep(run, alias, depends...)
}

func (pm *ProcessMeta) CustomStep(run func(ctx Step) (any, error), name string, depends ...any) *StepMeta {
	if !isValidIdentifier(name) {
		panic(patternHint)
	}
	meta := &StepMeta{
		name: name,
	}
	names := pm.normalize(depends...)
	for _, dependName := range names {
		depend, exist := pm.steps[dependName]
		if !exist {
			panic(fmt.Sprintf("[Step: %s]'s depend[%s] not found.", name, dependName))
		}
		meta.depends = append(meta.depends, depend)
	}

	if old, exist := pm.steps[name]; exist {
		if !old.position.Has(mergedE) {
			panic(fmt.Sprintf("[Step: %s] already exist, can used %s to avoid name duplicate",
				name, getFuncName(pm.CustomStep)))
		}
		pm.mergeStep(meta)
		old.run = run
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

func (pm *ProcessMeta) normalize(depends ...any) []string {
	exist := newRoutineUnsafeSet[string]()
	names := make([]string, 0, len(depends))
	for _, depend := range depends {
		candidates := make([]string, 1)
		if point, ok := depend.(SyncPoint); ok {
			candidates = point.points()
		} else {
			candidates[0] = toStepName(depend)
		}
		for _, name := range candidates {
			if exist.Contains(name) {
				continue
			}
			names = append(names, name)
			exist.Add(name)
		}
	}
	return names
}

func (pm *ProcessMeta) addRouterInfo(step *StepMeta) {
	if pm.nodeNum == 62 {
		panic(fmt.Sprintf("[Process: %s] exceeds max nodes num, max node num is 62", pm.name))
	}
	step.nodeRouter = nodeRouter{
		nodePath: 1 << pm.nodeNum,
		index:    uint64(pm.nodeNum),
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
