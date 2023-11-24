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
	Futures() []*Future
	Fails() []*Future
}

type FlowController interface {
	Controller
	ResultI
	Done() []*Future
	ListProcess() []string
	ProcessController(name string) Controller
}

type Configuration struct {
	flowConfig
	processConfig
}

type FlowMeta struct {
	flowConfig
	visitor
	init         sync.Once
	flowName     string
	noUseDefault bool
	processes    []*ProcessMeta
}

type flowConfig struct {
	*callbackChain[*FlowInfo]
}

type runFlow struct {
	*FlowMeta
	*basicInfo
	*visibleContext
	processes []*runProcess
	futures   []*Future
	lock      sync.Mutex
	finish    sync.WaitGroup
}

type FlowInfo struct {
	*basicInfo
	*visibleContext
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
		flowConfig:    flowConfig{},
		processConfig: newProcessConfig(),
	}
	return defaultConfig
}

func RegisterFlow(name string) *FlowMeta {
	flow := FlowMeta{
		visitor: visitor{
			index:   int32(visibleAll),
			visible: visibleAll,
			roster:  map[int32]string{int32(visibleAll): name},
		},
		flowConfig: flowConfig{&callbackChain[*FlowInfo]{}},
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
	flow := factory.(*FlowMeta).buildRunFlow(input)
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
	flow := factory.(*FlowMeta).buildRunFlow(input)
	flow.Done()
	return flow
}

func BuildRunFlow(name string, input map[string]any) *runFlow {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("flow factory [%s] not found", name))
	}
	return factory.(*FlowMeta).buildRunFlow(input)
}

func (fc *flowConfig) BeforeFlow(must bool, callback func(*FlowInfo) (keepOn bool, err error)) *callback[*FlowInfo] {
	if fc.callbackChain == nil {
		fc.callbackChain = &callbackChain[*FlowInfo]{}
	}
	return fc.addCallback(Before, must, callback)
}

func (fc *flowConfig) AfterFlow(must bool, callback func(*FlowInfo) (keepOn bool, err error)) *callback[*FlowInfo] {
	if fc.callbackChain == nil {
		fc.callbackChain = &callbackChain[*FlowInfo]{}
	}
	return fc.addCallback(After, must, callback)
}

func (fc *flowConfig) merge(merged *flowConfig) *flowConfig {
	CopyPropertiesSkipNotEmpty(merged, fc)
	if merged.callbackChain != nil {
		fc.filters = append(merged.copyChain(), fc.filters...)
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
		if defaultConfig == nil {
			return
		}
		fm.flowConfig.merge(&defaultConfig.flowConfig)
	})
}

func (fm *FlowMeta) NotUseDefault() *FlowMeta {
	fm.noUseDefault = true
	return fm
}

func (fm *FlowMeta) buildRunFlow(input map[string]any) *runFlow {
	table := adjacencyTable{
		lock:  &sync.RWMutex{},
		nodes: make(map[string]*node, len(input)),
	}
	rf := runFlow{
		visibleContext: &visibleContext{
			adjacencyTable: table,
			visitor:        &fm.visitor,
		},
		basicInfo: &basicInfo{
			Status: emptyStatus(),
			Name:   fm.flowName,
			Id:     generateId(),
		},
		FlowMeta:  fm,
		lock:      sync.Mutex{},
		processes: make([]*runProcess, 0, len(fm.processes)),
		finish:    sync.WaitGroup{},
	}

	for k, v := range input {
		rf.Set(k, v)
	}

	for _, processMeta := range fm.processes {
		process := rf.buildRunProcess(processMeta)
		rf.processes = append(rf.processes, process)
	}

	return &rf
}

func (fm *FlowMeta) TakeProcess(name string) {
	pm, exist := allProcess.Load(name)
	if !exist {
		panic(fmt.Sprintf("process [%s] not registered", name))
	}
	fm.processes = append(fm.processes, pm.(*ProcessMeta))
}

func (fm *FlowMeta) Process(name string) *ProcessMeta {
	pm := ProcessMeta{
		visitor: visitor{
			visible: 0,
			index:   0,
			roster:  map[int32]string{0: name},
		},
		processConfig: processConfig{
			stepChain: &callbackChain[*StepInfo]{},
			procChain: &callbackChain[*ProcessInfo]{},
		},
		processName: name,
		init:        sync.Once{},
		steps:       make(map[string]*StepMeta),
	}

	pm.register()
	fm.processes = append(fm.processes, &pm)
	return &pm
}

func (rf *runFlow) buildRunProcess(meta *ProcessMeta) *runProcess {
	table := adjacencyTable{
		lock:  &sync.RWMutex{},
		nodes: map[string]*node{},
	}
	process := runProcess{
		visibleContext: &visibleContext{
			parent:         rf.visibleContext,
			visitor:        &meta.visitor,
			adjacencyTable: table,
		},
		Status:      emptyStatus(),
		ProcessMeta: meta,
		id:          generateId(),
		flowId:      rf.Id,
		flowSteps:   make(map[string]*runStep),
		pause:       sync.WaitGroup{},
		running:     sync.WaitGroup{},
		finish:      sync.WaitGroup{},
	}

	stepTable := adjacencyTable{
		lock:  &sync.RWMutex{},
		nodes: map[string]*node{},
	}
	for _, stepMeta := range meta.sortedStep() {
		step := process.buildRunStep(stepMeta)
		step.adjacencyTable = stepTable
		process.flowSteps[stepMeta.stepName] = step
	}
	process.running.Add(len(process.flowSteps))

	return &process
}

// Done function will block util all process done.
func (rf *runFlow) Done() []*Future {
	futures := rf.Flow()
	for _, future := range futures {
		// process finish running and callback
		future.Done()
	}
	// workflow finish running and callback
	rf.finish.Wait()
	return futures
}

// Flow function asynchronous execute process of workflow and return immediately.
func (rf *runFlow) Flow() []*Future {
	if rf.futures != nil {
		return rf.futures
	}
	// avoid duplicate call
	rf.lock.Lock()
	defer rf.lock.Unlock()
	// DCL
	if rf.futures != nil {
		return rf.futures
	}
	rf.initialize()
	info := &FlowInfo{
		basicInfo: &basicInfo{
			Status: rf.Status,
			Id:     rf.Id,
			Name:   rf.flowName,
		},
		visibleContext: rf.visibleContext,
	}
	if rf.flowConfig.callbackChain != nil {
		rf.process(Before, info)
	}
	futures := make([]*Future, 0, len(rf.processes))
	for _, process := range rf.processes {
		futures = append(futures, process.flow())
	}
	rf.futures = futures
	rf.finish.Add(1)
	if rf.flowConfig.callbackChain == nil {
		return futures
	}

	go func() {
		for _, future := range futures {
			future.Done()
		}
		for _, process := range rf.processes {
			rf.combine(process.Status)
		}
		if rf.Normal() {
			rf.Append(Success)
		}
		rf.process(After, info)
		rf.finish.Done()
	}()

	return futures
}

func (rf *runFlow) Fails() []*Future {
	futures := make([]*Future, 0)
	for _, future := range rf.futures {
		if len(future.Exceptions()) != 0 {
			futures = append(futures, future)
		}
	}
	return futures
}

func (rf *runFlow) Futures() []*Future {
	return rf.futures
}

func (rf *runFlow) ListProcess() []string {
	processes := make([]string, 0, len(rf.processes))
	for _, process := range rf.processes {
		processes = append(processes, process.processName)
	}
	return processes
}

func (rf *runFlow) ProcessController(name string) Controller {
	for _, process := range rf.processes {
		if process.processName == name {
			return process
		}
	}
	return nil
}

func (rf *runFlow) Pause() {
	for _, process := range rf.processes {
		process.Pause()
	}
}

func (rf *runFlow) Resume() {
	for _, process := range rf.processes {
		process.Resume()
	}
}

func (rf *runFlow) Stop() {
	for _, process := range rf.processes {
		process.Stop()
	}
}
