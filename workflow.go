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
	compress  state
	processes map[string]*runProcess
	start     time.Time
	end       time.Time
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

func AsyncFlow(name string, input map[string]any) FlowController {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("Flow[%s] not found", name))
	}
	flow := factory.(*FlowMeta).buildRunFlow(input)
	go flow.Done()
	return flow
}

func DoneFlow(name string, input map[string]any) FinishedWorkFlow {
	factory, ok := allFlows.Load(name)
	if !ok {
		panic(fmt.Sprintf("Flow[%s] not found", name))
	}
	flow := factory.(*FlowMeta).buildRunFlow(input)
	return flow.Done()
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

func (rf *runFlow) HasAny(enum ...*StatusEnum) bool {
	return rf.compress.Has(enum...)
}

func (rf *runFlow) Process(name string) (ProcController, bool) {
	res, ok := rf.processes[name]
	if ok {
		return res, true
	}
	return nil, false
}

func (rf *runFlow) Exceptions() []FinishedProcess {
	res := make([]FinishedProcess, 0)
	for _, process := range rf.processes {
		if !process.state.Normal() {
			res = append(res, process)
		}
	}
	return res
}

func (rf *runFlow) ID() string {
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
	for _, process := range rf.processes {
		rf.compress.add(process.compress.load())
	}
	rf.compress.add(rf.load())
	if rf.compress.Normal() {
		rf.append(Success)
	} else {
		rf.append(Failed)
	}
	if rf.shallRecover() {
		rf.saveCheckpoints()
	}
	rf.advertise(After)
	rf.compress.add(rf.load())
	rf.finish.Done()
}

// Done function will block util all process done.
func (rf *runFlow) Done() FinishedWorkFlow {
	if !rf.EndTime().IsZero() {
		return rf
	}
	// Asynchronous scheduling may call Done twice,
	// so a lock is used to ensure that it only executes once.
	rf.lock.Lock()
	defer rf.lock.Unlock()
	// DCL
	if !rf.EndTime().IsZero() {
		return rf
	}
	rf.start = time.Now().UTC()
	rf.initialize()
	// execute before flow callback
	rf.advertise(Before)
	wg := make([]*sync.WaitGroup, 0, len(rf.processes))
	for _, process := range rf.processes {
		if rf.Has(CallbackFail) {
			process.append(Cancel)
		}
		wg = append(wg, process.schedule())
	}
	rf.finish.Add(1)
	for _, w := range wg {
		w.Wait()
	}
	// execute callback and recover after all processes are done
	rf.finalize()
	rf.end = time.Now().UTC()
	return rf
}

func (rf *runFlow) advertise(flag string) {
	if !rf.flowNotUseDefault && defaultConfig != nil {
		defaultConfig.handle(flag, rf)
	}
	rf.handle(flag, rf)
}

// Fails returns a slice of fail result Future pointers.
// If all process success, return an empty slice.
//func (rf *runFlow) Fails() []*Future {
//	futures := make([]*Future, 0)
//	for _, future := range rf.futures {
//		if len(future.Exceptions()) != 0 {
//			futures = append(futures, future)
//		}
//	}
//	return futures
//}

// Futures returns the slice of future objects.
// future can get the result and state of the process.
//func (rf *runFlow) Futures() []*Future {
//	return rf.futures
//}

func (rf *runFlow) ListProcess() []string {
	processes := make([]string, 0, len(rf.processes))
	for _, process := range rf.processes {
		processes = append(processes, process.name)
	}
	return processes
}

func (rf *runFlow) Pause() FlowController {
	rf.append(Pause)
	for _, process := range rf.processes {
		process.Pause()
	}
	return rf
}

// Resume resumes the execution of the workflow.
func (rf *runFlow) Resume() FlowController {
	for _, process := range rf.processes {
		process.Resume()
	}
	return rf
}

// Stop stops the execution of the workflow.
func (rf *runFlow) Stop() FlowController {
	for _, process := range rf.processes {
		process.Stop()
	}
	return rf
}
