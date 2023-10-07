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

var (
	defaultConfig *Configuration
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

type Configuration struct {
	*FlowConfig
	*ProcessConfig
}

type FlowMeta struct {
	*FlowConfig
	init         sync.Once
	name         string
	noUseDefault bool
	processes    []*ProcessMeta
}

type FlowConfig struct {
	flowCallback *CallbackChain[*FlowInfo]
}

type RunFlow struct {
	*FlowMeta
	id         string
	processMap map[string]*RunProcess
	context    *Context
	features   map[string]*Feature
	lock       sync.Mutex
	status     int64
	finish     sync.WaitGroup
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
	generateId = method
}

func SetDefaultConfig(config *Configuration) {
	defaultConfig = config
}

func RegisterFlow(name string) *FlowMeta {
	flow := FlowMeta{
		FlowConfig: &FlowConfig{flowCallback: &CallbackChain[*FlowInfo]{}},
		name:       name,
		init:       sync.Once{},
	}
	flow.register()
	return &flow
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
	flow := factory.(*FlowMeta).BuildRunFlow(input)
	flow.Flow()
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
	flow := factory.(*FlowMeta).BuildRunFlow(input)
	return flow.Done()
}

func BuildRunFlow(name string, input map[string]any) *RunFlow {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("flow factory [%s] not found", name))
	}
	return factory.(*FlowMeta).BuildRunFlow(input)
}

func (c *Configuration) AddBeforeStep(must bool, callback func(*StepInfo) (keepOn bool)) *Callback[*StepInfo] {
	if c.ProcessConfig == nil {
		c.ProcessConfig = &ProcessConfig{}
	}
	return c.ProcessConfig.AddBeforeStep(must, callback)
}

func (c *Configuration) AddAfterStep(must bool, callback func(*StepInfo) (keepOn bool)) *Callback[*StepInfo] {
	if c.ProcessConfig == nil {
		c.ProcessConfig = &ProcessConfig{}
	}
	return c.ProcessConfig.AddAfterStep(must, callback)
}

func (c *Configuration) AddBeforeProcess(must bool, callback func(*ProcessInfo) (keepOn bool)) *Callback[*ProcessInfo] {
	if c.ProcessConfig == nil {
		c.ProcessConfig = &ProcessConfig{}
	}
	return c.ProcessConfig.AddBeforeProcess(must, callback)
}

func (c *Configuration) AddAfterProcess(must bool, callback func(*ProcessInfo) (keepOn bool)) *Callback[*ProcessInfo] {
	if c.ProcessConfig == nil {
		c.ProcessConfig = &ProcessConfig{}
	}
	return c.ProcessConfig.AddAfterProcess(must, callback)
}

func (c *Configuration) AddBeforeFlow(must bool, callback func(*FlowInfo) (keepOn bool)) *Callback[*FlowInfo] {
	if c.FlowConfig == nil {
		c.FlowConfig = &FlowConfig{}
	}
	return c.FlowConfig.AddBeforeFlow(must, callback)
}

func (c *Configuration) AddAfterFlow(must bool, callback func(*FlowInfo) (keepOn bool)) *Callback[*FlowInfo] {
	if c.FlowConfig == nil {
		c.FlowConfig = &FlowConfig{}
	}
	return c.FlowConfig.AddAfterFlow(must, callback)
}

func (fc *FlowConfig) AddBeforeFlow(must bool, callback func(*FlowInfo) (keepOn bool)) *Callback[*FlowInfo] {
	if fc.flowCallback == nil {
		fc.flowCallback = &CallbackChain[*FlowInfo]{}
	}
	return fc.flowCallback.Add(Before, must, callback)
}

func (fc *FlowConfig) AddAfterFlow(must bool, callback func(*FlowInfo) (keepOn bool)) *Callback[*FlowInfo] {
	if fc.flowCallback == nil {
		fc.flowCallback = &CallbackChain[*FlowInfo]{}
	}
	return fc.flowCallback.Add(After, must, callback)
}

func (fc *FlowConfig) merge(merged *FlowConfig) *FlowConfig {
	CopyPropertiesSkipNotEmpty(merged, fc)
	if merged.flowCallback != nil {
		fc.flowCallback.filters = append(merged.flowCallback.CopyChain(), fc.flowCallback.filters...)
		fc.flowCallback.sort()
	}
	return fc
}

func (fm *FlowMeta) register() *FlowMeta {
	if len(fm.name) == 0 {
		panic("can't register flow factory with empty stepName")
	}

	_, load := allFlows.LoadOrStore(fm.name, fm)
	if load {
		panic(fmt.Sprintf("register duplicate flow factory named [%s]", fm.name))
	}

	return fm
}

func (fm *FlowMeta) initialize() {
	fm.init.Do(func() {
		if fm.noUseDefault {
			return
		}
		if defaultConfig == nil || defaultConfig.FlowConfig == nil {
			return
		}
		if fm.FlowConfig == nil {
			fm.FlowConfig = defaultConfig.FlowConfig
			return
		}
		fm.FlowConfig.merge(defaultConfig.FlowConfig)
	})
}

func (fm *FlowMeta) NotUseDefault() *FlowMeta {
	fm.noUseDefault = true
	return fm
}

func (fm *FlowMeta) BuildRunFlow(input map[string]any) *RunFlow {
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
		FlowMeta:   fm,
		id:         generateId(),
		lock:       sync.Mutex{},
		context:    &context,
		processMap: make(map[string]*RunProcess),
		finish:     sync.WaitGroup{},
	}

	for _, processMeta := range fm.processes {
		process := wf.buildRunProcess(processMeta)
		wf.processMap[process.processName] = process
	}

	return &wf
}

func (fm *FlowMeta) AddRegisterProcess(name string) {
	pm, exist := allProcess.Load(name)
	if !exist {
		panic(fmt.Sprintf("process [%s] not registered", name))
	}
	fm.processes = append(fm.processes, pm.(*ProcessMeta))
}

func (fm *FlowMeta) AddProcess(name string, conf *ProcessConfig) *ProcessMeta {
	if conf != nil {
		conf.notUseDefault = true
	}
	if conf == nil {
		conf = &ProcessConfig{}
	}

	pm := ProcessMeta{
		ProcessConfig: conf,
		processName:   name,
		init:          sync.Once{},
		steps:         make(map[string]*StepMeta),
	}

	pm.register()
	fm.processes = append(fm.processes, &pm)

	return &pm
}

func (wf *RunFlow) buildRunProcess(meta *ProcessMeta) *RunProcess {
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
		finish:      sync.WaitGroup{},
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
	features := wf.Flow()
	for _, feature := range features {
		// process finish running and callback
		feature.Done()
	}
	// workflow finish running and callback
	wf.finish.Wait()
	return features
}

// Flow function asynchronous execute process of workflow and return immediately.
func (wf *RunFlow) Flow() map[string]*Feature {
	if wf.features != nil {
		return wf.features
	}
	// avoid duplicate call
	wf.lock.Lock()
	defer wf.lock.Unlock()
	// DCL
	if wf.features != nil {
		return wf.features
	}
	wf.initialize()
	info := &FlowInfo{
		BasicInfo: &BasicInfo{
			Id:     wf.id,
			Name:   wf.name,
			status: &wf.status,
		},
		Ctx: wf.context,
	}
	if wf.FlowConfig != nil && wf.FlowConfig.flowCallback != nil {
		wf.FlowConfig.flowCallback.process(Before, info)
	}
	features := make(map[string]*Feature, len(wf.processMap))
	for name, process := range wf.processMap {
		features[name] = process.flow()
	}
	wf.features = features
	if wf.FlowConfig != nil && wf.FlowConfig.flowCallback != nil {
		go func() {
			for _, feature := range features {
				feature.Done()
			}
			wf.flowCallback.process(After, info)
			wf.finish.Done()
		}()
	}
	wf.finish.Add(1)
	return features
}

func (wf *RunFlow) SkipFinishedStep(name string, result any) error {
	count := 0
	for processName := range wf.processMap {
		if err := wf.skipProcessStep(processName, name, result); err == nil {
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

func (wf *RunFlow) skipProcessStep(processName, stepName string, result any) error {
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
