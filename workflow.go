package light_flow

import (
	"fmt"
	"github.com/google/uuid"
	"sync"
)

var (
	generateId func() string
)

var (
	allFlows   sync.Map
	allProcess sync.Map
)

type FlowFactory struct {
	name      string
	processes []*ProcessMeta
}

type Workflow struct {
	id         string
	processMap map[string]*FlowProcess
	context    *Context
	features   map[string]*Feature
	finish     sync.WaitGroup
	lock       sync.Mutex
}

func init() {
	generateId = func() string {
		return uuid.NewString()
	}
}

func SetIdGenerator(method func() string) {
	lock.Lock()
	defer lock.Unlock()
	generateId = method
}

func AddFlowFactory(name string) *FlowFactory {
	flow := FlowFactory{name: name}
	flow.register()
	return &flow
}

func (ff *FlowFactory) register() *FlowFactory {
	if len(ff.name) == 0 {
		panic("can't register flow factory with empty stepName")
	}

	_, load := allFlows.LoadOrStore(ff.name, ff)
	if load {
		panic(fmt.Sprintf("register duplicate flow factory named [%s]", ff.name))
	}

	return ff
}

func AsyncFlow(name string, input map[string]any) *Workflow {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("flow factory [%s] not found", name))
	}
	flow := factory.(*FlowFactory).BuildWorkflow(input)
	flow.Async()
	return flow
}

func DoneFlow(name string, input map[string]any) map[string]*Feature {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("flow factory [%s] not found", name))
	}
	flow := factory.(*FlowFactory).BuildWorkflow(input)
	return flow.Done()
}

func BuildWorkflow(name string, input map[string]any) *Workflow {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("flow factory [%s] not found", name))
	}
	return factory.(*FlowFactory).BuildWorkflow(input)
}

func (ff *FlowFactory) BuildWorkflow(input map[string]any) *Workflow {
	context := Context{
		name:      WorkflowCtx,
		scopes:    []string{WorkflowCtx},
		scopeCtxs: make(map[string]*Context),
		table:     sync.Map{},
		priority:  make(map[string]string),
	}

	for k, v := range input {
		context.table.Store(k, v)
	}

	wf := Workflow{
		id:         generateId(),
		lock:       sync.Mutex{},
		context:    &context,
		processMap: make(map[string]*FlowProcess),
		finish:     sync.WaitGroup{},
	}

	for _, processMeta := range ff.processes {
		process := wf.addProcess(processMeta)
		wf.processMap[process.processName] = process
	}

	return &wf
}

func (ff *FlowFactory) AddRegisterProcess(name string) {
	pm, exist := allProcess.Load(name)
	if !exist {
		panic(fmt.Sprintf("process [%s] not registered", name))
	}
	ff.processes = append(ff.processes, pm.(*ProcessMeta))
}

func (ff *FlowFactory) AddProcess(name string, conf *ProcessConfig) *ProcessMeta {
	used := &ProcessConfig{}
	if defaultProcessConfig != nil {
		used = defaultProcessConfig
	}
	if conf != nil {
		used = conf
	}
	pm := ProcessMeta{
		processName: name,
		conf:        used,
		steps:       make(map[string]*StepMeta),
	}

	pm.register()
	ff.processes = append(ff.processes, &pm)

	return &pm
}

func (wf *Workflow) addProcess(meta *ProcessMeta) *FlowProcess {
	pcsCtx := Context{
		name:   meta.processName,
		scopes: []string{ProcessCtx},
		table:  sync.Map{},
	}

	process := FlowProcess{
		ProcessMeta: meta,
		Context:     &pcsCtx,
		id:          generateId(),
		flowId:      wf.id,
		flowSteps:   make(map[string]*FlowStep),
		pcsScope:    map[string]*Context{ProcessCtx: &pcsCtx},
		pause:       sync.WaitGroup{},
		needRun:     sync.WaitGroup{},
	}
	pcsCtx.scopeCtxs = process.pcsScope
	pcsCtx.parents = append(pcsCtx.parents, wf.context)

	for _, stepMeta := range meta.orderStepMeta() {
		process.addStep(stepMeta)
	}

	process.needRun.Add(len(process.flowSteps))

	return &process
}

// Done function will block util all process done.
func (wf *Workflow) Done() map[string]*Feature {
	features := wf.Async()
	for _, feature := range features {
		feature.Done()
	}
	return features
}

// Async function asynchronous execute process of workflow and return immediately.
func (wf *Workflow) Async() map[string]*Feature {
	if wf.features != nil {
		return wf.features
	}
	wf.lock.Lock()
	defer wf.lock.Unlock()
	// DCL
	if wf.features != nil {
		return wf.features
	}
	features := make(map[string]*Feature, len(wf.processMap))
	for name, process := range wf.processMap {
		features[name] = process.flow()
	}
	wf.features = features
	return features
}

func (wf *Workflow) SkipFinishedStep(name string, result any) error {
	count := 0
	for processName := range wf.processMap {
		if err := wf.SkipProcessStep(processName, name, result); err == nil {
			count += 1
		}
	}

	if count > 1 {
		return fmt.Errorf("duplicate step [%s] found in workflow", name)
	}
	if count == 0 {
		return fmt.Errorf("step [%s] not found in workflow", name)
	}
	return nil
}

func (wf *Workflow) SkipProcessStep(processName, stepName string, result any) error {
	process, exist := wf.processMap[processName]
	if !exist {
		return fmt.Errorf("prcoess [%s] not found in workflow", processName)
	}
	step, exist := process.flowSteps[stepName]
	if !exist {
		return fmt.Errorf("step [%s] not found in process", stepName)
	}
	step.setStepResult(stepName, result)
	return nil
}

func (wf *Workflow) GetFeatures() map[string]*Feature {
	return wf.features
}

func (wf *Workflow) Pause() {
	for _, process := range wf.processMap {
		process.pauses()
	}
}

func (wf *Workflow) Resume() {
	for _, process := range wf.processMap {
		process.resume()
	}
}

func (wf *Workflow) PauseProcess(name string) {
	process := wf.processMap[name]
	process.pauses()
}

func (wf *Workflow) ResumeProcess(name string) {
	process := wf.processMap[name]
	process.resume()
}
