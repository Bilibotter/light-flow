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

type Controller interface {
	Resume()
	Pause()
	Stop()
}

type WorkFlowCtrl interface {
	Controller
	Done() map[string]*Feature
	GetFeatures() map[string]*Feature
	ListProcess() []string
	GetProcessController(name string) Controller
}

type FlowMeta struct {
	name         string
	processes    []*ProcessMeta
	flowCallback []*Callback[*FlowInfo]
}

type RunFlow struct {
	id         string
	processMap map[string]*RunProcess
	context    *Context
	features   map[string]*Feature
	finish     sync.WaitGroup
	lock       sync.Mutex
}

type FlowInfo struct {
	*BasicInfo
	Ctx *Context
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

func RegisterFlow(name string) *FlowMeta {
	flow := FlowMeta{name: name}
	flow.register()
	return &flow
}

func (ff *FlowMeta) register() *FlowMeta {
	if len(ff.name) == 0 {
		panic("can't register flow factory with empty stepName")
	}

	_, load := allFlows.LoadOrStore(ff.name, ff)
	if load {
		panic(fmt.Sprintf("register duplicate flow factory named [%s]", ff.name))
	}

	return ff
}

func AsyncArgs(name string, args ...any) WorkFlowCtrl {
	input := make(map[string]any, len(args))
	for _, arg := range args {
		input[GetStructName(arg)] = arg
	}
	return AsyncFlow(name, input)
}

func AsyncFlow(name string, input map[string]any) WorkFlowCtrl {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("flow factory [%s] not found", name))
	}
	flow := factory.(*FlowMeta).BuildWorkflow(input)
	flow.Async()
	return flow
}

func DoneArgs(name string, args ...any) map[string]*Feature {
	input := make(map[string]any, len(args))
	for _, arg := range args {
		input[GetStructName(arg)] = arg
	}
	return DoneFlow(name, input)
}

func DoneFlow(name string, input map[string]any) map[string]*Feature {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("flow factory [%s] not found", name))
	}
	flow := factory.(*FlowMeta).BuildWorkflow(input)
	return flow.Done()
}

func BuildWorkflow(name string, input map[string]any) *RunFlow {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("flow factory [%s] not found", name))
	}
	return factory.(*FlowMeta).BuildWorkflow(input)
}

func (ff *FlowMeta) AddBeforeFlow(must bool, callback func(*FlowInfo) (keepOn bool)) *Callback[*FlowInfo] {
	return nil
}

func (ff *FlowMeta) BuildWorkflow(input map[string]any) *RunFlow {
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

	wf := RunFlow{
		id:         generateId(),
		lock:       sync.Mutex{},
		context:    &context,
		processMap: make(map[string]*RunProcess),
		finish:     sync.WaitGroup{},
	}

	for _, processMeta := range ff.processes {
		process := wf.addProcess(processMeta)
		wf.processMap[process.processName] = process
	}

	return &wf
}

func (ff *FlowMeta) AddRegisterProcess(name string) {
	pm, exist := allProcess.Load(name)
	if !exist {
		panic(fmt.Sprintf("process [%s] not registered", name))
	}
	ff.processes = append(ff.processes, pm.(*ProcessMeta))
}

func (ff *FlowMeta) AddProcess(name string, conf *ProcessConfig) *ProcessMeta {
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

func (wf *RunFlow) addProcess(meta *ProcessMeta) *RunProcess {
	pcsCtx := Context{
		name:   meta.processName,
		scopes: []string{ProcessCtx},
		table:  sync.Map{},
	}

	process := RunProcess{
		ProcessMeta: meta,
		Context:     &pcsCtx,
		id:          generateId(),
		flowId:      wf.id,
		flowSteps:   make(map[string]*RunStep),
		pcsScope:    map[string]*Context{ProcessCtx: &pcsCtx},
		pause:       sync.WaitGroup{},
		needRun:     sync.WaitGroup{},
	}
	pcsCtx.scopeCtxs = process.pcsScope
	pcsCtx.parents = append(pcsCtx.parents, wf.context)

	for _, stepMeta := range meta.sortedStepMeta() {
		process.addStep(stepMeta)
	}

	process.needRun.Add(len(process.flowSteps))

	return &process
}

// Done function will block util all process done.
func (wf *RunFlow) Done() map[string]*Feature {
	features := wf.Async()
	for _, feature := range features {
		feature.Done()
	}
	return features
}

// Async function asynchronous execute process of workflow and return immediately.
func (wf *RunFlow) Async() map[string]*Feature {
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

func (wf *RunFlow) SkipFinishedStep(name string, result any) error {
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

func (wf *RunFlow) SkipProcessStep(processName, stepName string, result any) error {
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

func (wf *RunFlow) GetFeatures() map[string]*Feature {
	return wf.features
}

func (wf *RunFlow) ListProcess() []string {
	processes := make([]string, 0, len(wf.processMap))
	for name := range wf.processMap {
		processes = append(processes, name)
	}
	return processes
}

func (wf *RunFlow) GetProcessController(name string) Controller {
	process, exist := wf.processMap[name]
	if !exist {
		return nil
	}
	return process
}

func (wf *RunFlow) Pause() {
	for _, process := range wf.processMap {
		process.Pause()
	}
}

func (wf *RunFlow) Resume() {
	for _, process := range wf.processMap {
		process.Resume()
	}
}

func (wf *RunFlow) Stop() {
	for _, process := range wf.processMap {
		process.Stop()
	}
}
