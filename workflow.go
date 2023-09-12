package light_flow

import (
	"sync"
)

type Workflow struct {
	processMap map[string]*Process
	context    *Context
	features   map[string]*Feature
	pause      sync.WaitGroup
	finish     sync.WaitGroup
	lock       sync.Mutex
}

func NewWorkflow(input map[string]any) *Workflow {
	context := Context{
		scope:         ProcessCtx,
		scopeContexts: make(map[string]*Context),
		table:         sync.Map{},
		priority:      make(map[string]any),
	}
	for k, v := range input {
		context.table.Store(k, v)
	}

	context.scopeContexts[ProcessCtx] = &context

	flow := Workflow{
		lock:       sync.Mutex{},
		context:    &context,
		processMap: make(map[string]*Process),
		pause:      sync.WaitGroup{},
		finish:     sync.WaitGroup{},
	}

	return &flow
}

func (wf *Workflow) WaitToDone() map[string]*Feature {
	features := wf.AsyncFlow()
	for _, feature := range features {
		feature.WaitToDone()
	}
	return features
}

func (wf *Workflow) AsyncFlow() map[string]*Feature {
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

func (wf *Workflow) AddProcess(name string, conf *ProcessConfig) *Process {
	processCtx := Context{
		scope:         ProcessCtx,
		scopeContexts: make(map[string]*Context),
		table:         sync.Map{},
	}
	processCtx.scopeContexts[ProcessCtx] = &processCtx

	process := Process{
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
