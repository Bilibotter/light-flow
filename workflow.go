package light_flow

import (
	"github.com/google/uuid"
	"sync"
)

var (
	generateId func() string
)

type Workflow struct {
	id         string
	processMap map[string]*Process
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

// NewWorkflow function creates a workflow and use input as global context.
func NewWorkflow[T any](input map[string]T) *Workflow {
	context := Context{
		scope:         WorkflowCtx,
		scopeContexts: make(map[string]*Context),
		table:         sync.Map{},
		priority:      make(map[string]any),
	}
	for k, v := range input {
		context.table.Store(k, v)
	}

	context.scopeContexts[WorkflowCtx] = &context

	flow := Workflow{
		id:         generateId(),
		lock:       sync.Mutex{},
		context:    &context,
		processMap: make(map[string]*Process),
		finish:     sync.WaitGroup{},
	}

	return &flow
}

func SetIdGenerator(method func() string) {
	generateId = method
}

// Done function will block util all process done.
func (wf *Workflow) Done() map[string]*Feature {
	features := wf.Flow()
	for _, feature := range features {
		feature.Done()
	}
	return features
}

// Flow function asynchronous execute process of workflow and return immediately.
func (wf *Workflow) Flow() map[string]*Feature {
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
		features[name] = process.schedule()
	}
	wf.features = features
	return features
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

func (wf *Workflow) AddProcess(name string, conf *ProcessConfig) *Process {
	processCtx := Context{
		scope:         ProcessCtx,
		scopeContexts: make(map[string]*Context),
		table:         sync.Map{},
	}
	processCtx.scopeContexts[ProcessCtx] = &processCtx
	if conf == nil {
		conf = defaultProcessConfig
	}
	process := Process{
		id:              generateId(),
		flowId:          wf.id,
		name:            name,
		stepMap:         make(map[string]*Step),
		processContexts: processCtx.scopeContexts,
		context:         &processCtx,
		pause:           sync.WaitGroup{},
		running:         sync.WaitGroup{},
		conf:            conf,
	}

	wf.processMap[process.name] = &process
	wf.finish.Add(1)
	process.context.parents = append(process.context.parents, wf.context)

	return &process
}
