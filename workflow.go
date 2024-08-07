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

type FlowMeta struct {
	flowConfig
	flowCallback
	init      sync.Once
	name      string
	processes []*ProcessMeta
}

type runFlow struct {
	*state
	*FlowMeta
	*simpleContext
	id           string
	compress     state
	runProcesses map[string]*runProcess
	start        time.Time
	end          time.Time
	lock         sync.Mutex
	finish       sync.WaitGroup
}

func init() {
	generateId = func() string {
		return uuid.NewString()
	}
}

func SetIdGenerator(method func() string) {
	generateId = method
}

func DefaultConfig() FlowConfig {
	return defaultConfig
}

func RegisterFlow(name string) *FlowMeta {
	flow := FlowMeta{
		name:         name,
		init:         sync.Once{},
		flowCallback: buildFlowCallback(flowScope),
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

func (fm *FlowMeta) buildRunFlow(input map[string]any) *runFlow {
	ctx := simpleContext{
		table:    input,
		internal: make(map[string]any),
		ctxName:  fm.name,
	}
	rf := runFlow{
		state:         emptyStatus(),
		FlowMeta:      fm,
		simpleContext: &ctx,
		id:            generateId(),
		lock:          sync.Mutex{},
		runProcesses:  make(map[string]*runProcess, len(fm.processes)),
		finish:        sync.WaitGroup{},
	}

	for k, v := range input {
		rf.Set(k, v)
	}

	for _, processMeta := range fm.processes {
		process := rf.buildRunProcess(processMeta)
		rf.runProcesses[processMeta.name] = process
	}

	return &rf
}

func (fm *FlowMeta) Process(name string) *ProcessMeta {
	pm := ProcessMeta{
		nodeRouter: nodeRouter{
			nodePath: fullAccess,
			index:    0,
			toName:   map[uint64]string{0: name},
			toIndex:  map[string]uint64{name: 0},
		},
		procCallback: buildProcCallback(procScope),
		belong:       fm,
		name:         name,
		init:         sync.Once{},
		steps:        make(map[string]*StepMeta),
	}
	pm.register()
	fm.processes = append(fm.processes, &pm)
	return &pm
}

func (rf *runFlow) HasAny(enum ...*StatusEnum) bool {
	return rf.compress.Has(enum...)
}

func (rf *runFlow) Process(name string) (ProcController, bool) {
	res, ok := rf.runProcesses[name]
	if ok {
		return res, true
	}
	return nil, false
}

func (rf *runFlow) Exceptions() []FinishedProcess {
	res := make([]FinishedProcess, 0)
	for _, process := range rf.runProcesses {
		if !process.state.Normal() {
			res = append(res, process)
		}
	}
	return res
}

func (rf *runFlow) Recover() (FinishedWorkFlow, error) {
	if !rf.Has(Failed) {
		return nil, fmt.Errorf("workflow[%s] is't failed, can't recover", rf.name)
	}
	return RecoverFlow(rf.id)
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
		runSteps:    make(map[string]*runStep),
		pause:       sync.WaitGroup{},
		running:     sync.WaitGroup{},
		finish:      sync.WaitGroup{},
	}

	for _, stepMeta := range meta.sortedSteps() {
		step := process.buildRunStep(stepMeta)
		process.runSteps[stepMeta.name] = step
	}
	process.running.Add(len(process.runSteps))

	return &process
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
	rf.initialize()
	rf.start = time.Now().UTC()
	// execute before flow callback
	rf.advertise(beforeF)
	wg := make([]*sync.WaitGroup, 0, len(rf.runProcesses))
	for _, process := range rf.runProcesses {
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
	return rf
}

func (rf *runFlow) initialize() {
	rf.init.Do(func() {
		copyProperties(defaultConfig, &rf.flowConfig, true)
		copyProperties(&defaultConfig.processConfig, &rf.processConfig, true)
	})
}

func (rf *runFlow) advertise(flag uint64) {
	if !rf.disableDefault {
		if !defaultCallback.flowFilter(flag, rf) {
			return
		}
	}
	rf.flowFilter(flag, rf)
}

func (rf *runFlow) finalize() {
	for _, process := range rf.runProcesses {
		rf.compress.add(process.compress.load())
	}
	rf.compress.add(rf.load())
	if rf.compress.Normal() {
		rf.compress.append(Success)
		rf.append(Success)
	} else {
		rf.compress.append(Failed)
		rf.append(Failed)
	}
	rf.end = time.Now().UTC()
	rf.advertise(afterF)
	rf.compress.add(rf.load())
	if !rf.Normal() && rf.isRecoverable() {
		rf.saveCheckpoints()
	}
	rf.finish.Done()
}

func (rf *runFlow) ListProcess() []string {
	processes := make([]string, 0, len(rf.runProcesses))
	for _, process := range rf.runProcesses {
		processes = append(processes, process.name)
	}
	return processes
}

func (rf *runFlow) Pause() FlowController {
	rf.append(Pause)
	for _, process := range rf.runProcesses {
		process.Pause()
	}
	return rf
}

// Resume resumes the execution of the workflow.
func (rf *runFlow) Resume() FlowController {
	for _, process := range rf.runProcesses {
		process.Resume()
	}
	return rf
}

// Stop stops the execution of the workflow.
func (rf *runFlow) Stop() FlowController {
	for _, process := range rf.runProcesses {
		process.Stop()
	}
	return rf
}
