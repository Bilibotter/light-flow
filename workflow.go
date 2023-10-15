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

type ResultI interface {
	BasicInfoI
	Features() map[string]*Feature
	FailFeatures() map[string]*Feature
}

type FlowController interface {
	Controller
	ResultI
	Done() map[string]*Feature
	ListProcess() []string
	ProcessController(name string) Controller
}

type Configuration struct {
	*FlowConfig
	*processConfig
}

type FlowMeta struct {
	*FlowConfig
	init         sync.Once
	flowName     string
	noUseDefault bool
	processes    []*ProcessMeta
}

type FlowConfig struct {
	*CallbackChain[*FlowInfo]
}

type RunFlow struct {
	*FlowMeta
	*BasicInfo
	*Context
	processMap map[string]*RunProcess
	features   map[string]*Feature
	lock       sync.Mutex
	finish     sync.WaitGroup
}

type FlowInfo struct {
	*BasicInfo
	*Context
}

func init() {
	generateId = func() string {
		return uuid.NewString()
	}
}

func SetIdGenerator(method func() string) {
	generateId = method
}

func CreateDefaultConfig() *Configuration {
	defaultConfig = &Configuration{
		FlowConfig:    &FlowConfig{},
		processConfig: NewProcessConfig(),
	}
	return defaultConfig
}

func RegisterFlow(name string) *FlowMeta {
	flow := FlowMeta{
		FlowConfig: &FlowConfig{&CallbackChain[*FlowInfo]{}},
		flowName:   name,
		init:       sync.Once{},
	}
	flow.register()
	return &flow
}

func AsyncArgs(name string, args ...any) FlowController {
	input := make(map[string]any, len(args))
	for _, arg := range args {
		input[GetStructName(arg)] = arg
	}
	return AsyncFlow(name, input)
}

func AsyncFlow(name string, input map[string]any) FlowController {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("flow factory [%s] not found", name))
	}
	flow := factory.(*FlowMeta).BuildRunFlow(input)
	flow.Flow()
	return flow
}

func DoneArgs(name string, args ...any) ResultI {
	input := make(map[string]any, len(args))
	for _, arg := range args {
		input[GetStructName(arg)] = arg
	}
	return DoneFlow(name, input)
}

func DoneFlow(name string, input map[string]any) ResultI {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("flow factory [%s] not found", name))
	}
	flow := factory.(*FlowMeta).BuildRunFlow(input)
	flow.Done()
	return flow
}

func BuildRunFlow(name string, input map[string]any) *RunFlow {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("flow factory [%s] not found", name))
	}
	return factory.(*FlowMeta).BuildRunFlow(input)
}

func (fc *FlowConfig) AddBeforeFlow(must bool, callback func(*FlowInfo) (keepOn bool, err error)) *Callback[*FlowInfo] {
	if fc.CallbackChain == nil {
		fc.CallbackChain = &CallbackChain[*FlowInfo]{}
	}
	return fc.AddCallback(Before, must, callback)
}

func (fc *FlowConfig) AddAfterFlow(must bool, callback func(*FlowInfo) (keepOn bool, err error)) *Callback[*FlowInfo] {
	if fc.CallbackChain == nil {
		fc.CallbackChain = &CallbackChain[*FlowInfo]{}
	}
	return fc.AddCallback(After, must, callback)
}

func (fc *FlowConfig) merge(merged *FlowConfig) *FlowConfig {
	CopyPropertiesSkipNotEmpty(merged, fc)
	if merged.CallbackChain != nil {
		fc.filters = append(merged.CopyChain(), fc.filters...)
		fc.maintain()
	}
	return fc
}

func (fm *FlowMeta) register() *FlowMeta {
	if len(fm.flowName) == 0 {
		panic("can't register flow factory with empty stepName")
	}

	_, load := allFlows.LoadOrStore(fm.flowName, fm)
	if load {
		panic(fmt.Sprintf("register duplicate flow factory named [%s]", fm.flowName))
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
		name:      WorkflowCtx + fm.flowName,
		scopes:    []string{WorkflowCtx},
		scopeCtxs: make(map[string]*Context),
		table:     &sync.Map{},
		priority:  make(map[string]string),
	}

	for k, v := range input {
		context.table.Store(k, v)
	}

	rf := RunFlow{
		BasicInfo: &BasicInfo{
			Status: emptyStatus(),
			Name:   fm.flowName,
			Id:     generateId(),
		},
		FlowMeta:   fm,
		lock:       sync.Mutex{},
		Context:    &context,
		processMap: make(map[string]*RunProcess),
		finish:     sync.WaitGroup{},
	}

	for _, processMeta := range fm.processes {
		process := rf.buildRunProcess(processMeta)
		rf.processMap[process.processName] = process
	}

	return &rf
}

func (fm *FlowMeta) AddRegisterProcess(name string) {
	pm, exist := allProcess.Load(name)
	if !exist {
		panic(fmt.Sprintf("process [%s] not registered", name))
	}
	fm.processes = append(fm.processes, pm.(*ProcessMeta))
}

func (fm *FlowMeta) AddProcess(name string) *ProcessMeta {
	return fm.AddProcessWithConf(name, nil)
}

func (fm *FlowMeta) AddProcessWithConf(name string, conf *processConfig) *ProcessMeta {
	if conf != nil {
		conf.notUseDefault = true
	}
	if conf == nil {
		conf = NewProcessConfig()
	}

	pm := ProcessMeta{
		processConfig: conf,
		processName:   name,
		init:          sync.Once{},
		steps:         make(map[string]*StepMeta),
	}

	pm.register()
	fm.processes = append(fm.processes, &pm)

	return &pm
}

func (rf *RunFlow) buildRunProcess(meta *ProcessMeta) *RunProcess {
	pcsCtx := Context{
		name:   ProcessCtx + meta.processName,
		scopes: []string{ProcessCtx},
		table:  &sync.Map{},
	}

	process := RunProcess{
		Status:      emptyStatus(),
		ProcessMeta: meta,
		Context:     &pcsCtx,
		id:          generateId(),
		flowId:      rf.Id,
		flowSteps:   make(map[string]*RunStep),
		pcsScope:    map[string]*Context{ProcessCtx: &pcsCtx},
		pause:       sync.WaitGroup{},
		running:     sync.WaitGroup{},
		finish:      sync.WaitGroup{},
	}
	pcsCtx.scopeCtxs = process.pcsScope
	pcsCtx.parents = append(pcsCtx.parents, rf.Context)

	for _, stepMeta := range meta.sortedStepMeta() {
		process.buildRunStep(stepMeta)
	}

	process.running.Add(len(process.flowSteps))

	return &process
}

// Done function will block util all process done.
func (rf *RunFlow) Done() map[string]*Feature {
	features := rf.Flow()
	for _, feature := range features {
		// process finish running and callback
		feature.Done()
	}
	// workflow finish running and callback
	rf.finish.Wait()
	return features
}

// Flow function asynchronous execute process of workflow and return immediately.
func (rf *RunFlow) Flow() map[string]*Feature {
	if rf.features != nil {
		return rf.features
	}
	// avoid duplicate call
	rf.lock.Lock()
	defer rf.lock.Unlock()
	// DCL
	if rf.features != nil {
		return rf.features
	}
	rf.initialize()
	info := &FlowInfo{
		BasicInfo: &BasicInfo{
			Status: rf.Status,
			Id:     rf.Id,
			Name:   rf.flowName,
		},
		Context: rf.Context,
	}
	if rf.FlowConfig != nil && rf.FlowConfig.CallbackChain != nil {
		rf.process(Before, info)
	}
	features := make(map[string]*Feature, len(rf.processMap))
	for name, process := range rf.processMap {
		features[name] = process.flow()
	}
	rf.features = features
	rf.finish.Add(1)
	if rf.FlowConfig == nil || rf.FlowConfig.CallbackChain == nil {
		return features
	}

	go func() {
		for _, feature := range features {
			feature.Done()
		}
		for _, process := range rf.processMap {
			rf.combine(process.Status)
		}
		if rf.Normal() {
			rf.Append(Success)
		}
		rf.process(After, info)
		rf.finish.Done()
	}()

	return features
}

func (rf *RunFlow) SkipFinishedStep(name string, result any) error {
	count := 0
	for processName := range rf.processMap {
		if err := rf.skipProcessStep(processName, name, result); err == nil {
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

func (rf *RunFlow) skipProcessStep(processName, stepName string, result any) error {
	process, exist := rf.processMap[processName]
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

func (rf *RunFlow) FailFeatures() map[string]*Feature {
	features := make(map[string]*Feature)
	for name, feature := range rf.features {
		if len(feature.Exceptions()) != 0 {
			features[name] = feature
		}
	}
	return features
}

func (rf *RunFlow) Features() map[string]*Feature {
	return rf.features
}

func (rf *RunFlow) ListProcess() []string {
	processes := make([]string, 0, len(rf.processMap))
	for name := range rf.processMap {
		processes = append(processes, name)
	}
	return processes
}

func (rf *RunFlow) ProcessController(name string) Controller {
	process, exist := rf.processMap[name]
	if !exist {
		return nil
	}
	return process
}

func (rf *RunFlow) Pause() {
	for _, process := range rf.processMap {
		process.Pause()
	}
}

func (rf *RunFlow) Resume() {
	for _, process := range rf.processMap {
		process.Resume()
	}
}

func (rf *RunFlow) Stop() {
	for _, process := range rf.processMap {
		process.Stop()
	}
}
