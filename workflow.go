package light_flow

import (
	"fmt"
	"github.com/google/uuid"
	"sync"
	"time"
)

var (
	generateId func() string
)

var (
	allFlows   sync.Map
	allProcess sync.Map
)

var (
	defaultConfig *FlowConfig
)

type controller interface {
	Resume()
	Pause()
	Stop()
}

type resultI interface {
	basicInfoI
	Futures() []*Future
	Fails() []*Future
}

type FlowController interface {
	controller
	resultI
	Done() []*Future
	ListProcess() []string
	ProcessController(name string) controller
}

type FlowConfig struct {
	handler[WorkFlow] `flow:"skip"`
	ProcessConfig     `flow:"skip"`
	enableRecover     int8
}

type FlowMeta struct {
	FlowConfig
	init              sync.Once
	name              string
	flowNotUseDefault bool
	processes         []*ProcessMeta
}

type runFlow struct {
	*state
	*FlowMeta
	*simpleContext
	id        string
	processes map[string]*runProcess
	start     time.Time
	end       time.Time
	futures   []*Future
	lock      sync.Mutex
	finish    sync.WaitGroup
}

func init() {
	generateId = func() string {
		return uuid.NewString()
	}
}

func SetIdGenerator(method func() string) {
	generateId = method
}

func CreateDefaultConfig() *FlowConfig {
	defaultConfig = &FlowConfig{
		ProcessConfig: newProcessConfig(),
	}
	return defaultConfig
}

func RegisterFlow(name string) *FlowMeta {
	flow := FlowMeta{
		name: name,
		init: sync.Once{},
	}
	flow.register()
	return &flow
}

//func AsyncArgs(name string, args ...any) FlowController {
//	input := make(map[string]any, len(args))
//	for _, arg := range args {
//		input[getStructName(arg)] = arg
//	}
//	return AsyncFlow(name, input)
//}

//func AsyncFlow(name string, input map[string]any) FlowController {
//	factory, ok := allFlows.Load(name)
//	if !ok {
//		panic(fmt.Sprintf("Flow[%s] not found", name))
//	}
//	flow := factory.(*FlowMeta).buildRunFlow(input)
//	flow.Flow()
//	return flow
//}
//
//func DoneArgs(name string, args ...any) resultI {
//	input := make(map[string]any, len(args))
//	for _, arg := range args {
//		input[getStructName(arg)] = arg
//	}
//	return DoneFlow(name, input)
//}

func DoneFlow(name string, input map[string]any) {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("Flow[%s] not found", name))
	}
	flow := factory.(*FlowMeta).buildRunFlow(input)
	flow.Done()
}

func (fc *FlowConfig) BeforeFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) *callback[WorkFlow] {
	return fc.addCallback(Before, must, callback)
}

func (fc *FlowConfig) AfterFlow(must bool, callback func(WorkFlow) (keepOn bool, err error)) *callback[WorkFlow] {
	return fc.addCallback(After, must, callback)
}

func (fm *FlowMeta) initialize() {
	fm.init.Do(func() {
		if fm.flowNotUseDefault || defaultConfig == nil {
			return
		}
		copyPropertiesSkipNotEmpty(defaultConfig, &fm.FlowConfig)
		copyPropertiesSkipNotEmpty(&defaultConfig.ProcessConfig, &fm.FlowConfig.ProcessConfig)
		for _, meta := range fm.processes {
			if meta.ProcNotUseDefault {
				continue
			}
			copyPropertiesSkipNotEmpty(&fm.ProcessConfig, &meta.ProcessConfig)
		}
	})
}

func (fm *FlowMeta) register() *FlowMeta {
	if len(fm.name) == 0 {
		panic("can't register flow with empty name")
	}

	_, load := allFlows.LoadOrStore(fm.name, fm)
	if load {
		panic(fmt.Sprintf("Flow[%s] alraedy exists", fm.name))
	}

	return fm
}

func (fm *FlowMeta) Name() string {
	return fm.name
}

func (fm *FlowMeta) NoUseDefault() *FlowMeta {
	fm.flowNotUseDefault = true
	return fm
}

func (fm *FlowMeta) EnableRecover() *FlowMeta {
	fm.enableRecover = 1
	return fm
}

func (fm *FlowMeta) DisableRecover() *FlowMeta {
	fm.enableRecover = -1
	return fm
}

func (fm *FlowMeta) buildRunFlow(input map[string]any) *runFlow {
	ctx := simpleContext{
		table:   input,
		ctxName: fm.name,
	}
	rf := runFlow{
		state:         emptyStatus(),
		FlowMeta:      fm,
		simpleContext: &ctx,
		id:            generateId(),
		lock:          sync.Mutex{},
		processes:     make(map[string]*runProcess, len(fm.processes)),
		finish:        sync.WaitGroup{},
	}

	for k, v := range input {
		rf.Set(k, v)
	}

	for _, processMeta := range fm.processes {
		process := rf.buildRunProcess(processMeta)
		rf.processes[processMeta.name] = process
	}

	return &rf
}

func (fm *FlowMeta) Process(name string) *ProcessMeta {
	pm := ProcessMeta{
		nodeRouter: nodeRouter{
			nodePath: fullAccess,
			index:    0,
			toName:   map[int64]string{0: name},
			toIndex:  map[string]int64{name: 0},
		},
		belong: fm,
		name:   name,
		init:   sync.Once{},
		steps:  make(map[string]*StepMeta),
	}

	pm.register()
	fm.processes = append(fm.processes, &pm)
	return &pm
}

func (rf *runFlow) Id() string {
	return rf.id
}

func (rf *runFlow) StartTime() time.Time {
	return rf.start
}

func (rf *runFlow) EndTime() time.Time {
	return rf.end
}

func (rf *runFlow) CostTime() time.Duration {
	if rf.end.IsZero() {
		return 0
	}
	return rf.end.Sub(rf.start)
}

func (rf *runFlow) buildRunProcess(meta *ProcessMeta) *runProcess {
	table := linkedTable{
		lock:  sync.RWMutex{},
		nodes: map[string]*node{},
	}
	process := runProcess{
		dependentContext: &dependentContext{
			prev:        rf.simpleContext,
			routing:     &meta.nodeRouter,
			linkedTable: &table,
		},
		state:       emptyStatus(),
		ProcessMeta: meta,
		belong:      rf,
		id:          generateId(),
		flowSteps:   make(map[string]*runStep),
		pause:       sync.WaitGroup{},
		running:     sync.WaitGroup{},
		finish:      sync.WaitGroup{},
	}

	for _, stepMeta := range meta.sortedSteps() {
		step := process.buildRunStep(stepMeta)
		process.flowSteps[stepMeta.name] = step
	}
	process.running.Add(len(process.flowSteps))

	return &process
}

func (rf *runFlow) finalize() {
	for _, future := range rf.futures {
		future.Done()
	}
	for _, process := range rf.processes {
		rf.add(process.state.load())
	}
	if rf.Normal() {
		rf.append(Success)
	}
	if rf.shallRecover() {
		rf.saveCheckpoints()
	}
	rf.advertise(After)
	rf.finish.Done()
}

// Done function will block util all process done.
func (rf *runFlow) Done() []*Future {
	rf.start = time.Now().UTC()
	futures := rf.Flow()
	for _, future := range futures {
		// process finish running and callback
		future.Done()
	}
	// workflow finish running and callback
	rf.finish.Wait()
	rf.end = time.Now().UTC()
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
	rf.advertise(Before)
	futures := make([]*Future, 0, len(rf.processes))
	for _, process := range rf.processes {
		if rf.Has(CallbackFail) {
			process.append(CallbackFail)
		}
		futures = append(futures, process.schedule())
	}
	rf.futures = futures
	rf.finish.Add(1)
	// execute callback and recover after all processes are done
	go rf.finalize()
	return futures
}

func (rf *runFlow) advertise(flag string) {
	if !rf.flowNotUseDefault && defaultConfig != nil {
		defaultConfig.handle(flag, rf)
	}
	rf.handle(flag, rf)
}

// Fails returns a slice of fail result Future pointers.
// If all process success, return an empty slice.
func (rf *runFlow) Fails() []*Future {
	futures := make([]*Future, 0)
	for _, future := range rf.futures {
		if len(future.Exceptions()) != 0 {
			futures = append(futures, future)
		}
	}
	return futures
}

// Futures returns the slice of future objects.
// future can get the result and state of the process.
func (rf *runFlow) Futures() []*Future {
	return rf.futures
}

func (rf *runFlow) ListProcess() []string {
	processes := make([]string, 0, len(rf.processes))
	for _, process := range rf.processes {
		processes = append(processes, process.name)
	}
	return processes
}

// ProcessController returns the controller of the specified process name.
// controller can stop and resume the process.
func (rf *runFlow) ProcessController(processName string) controller {
	for _, process := range rf.processes {
		if process.name == processName {
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

// Resume resumes the execution of the workflow.
func (rf *runFlow) Resume() {
	for _, process := range rf.processes {
		process.Resume()
	}
}

// Stop stops the execution of the workflow.
func (rf *runFlow) Stop() {
	for _, process := range rf.processes {
		process.Stop()
	}
}
