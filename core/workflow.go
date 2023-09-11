package core

import (
	"sync"
)

type Workflow struct {
	procedureMap map[string]*Procedure
	context      *Context
	features     map[string]*Feature
	pause        sync.WaitGroup
	finish       sync.WaitGroup
	lock         sync.Mutex
}

func NewWorkflow(input map[string]any) *Workflow {
	context := Context{
		scope:         ProcedureCtx,
		scopeContexts: make(map[string]*Context),
		table:         sync.Map{},
		priority:      make(map[string]any),
	}
	for k, v := range input {
		context.table.Store(k, v)
	}

	context.scopeContexts[ProcedureCtx] = &context

	flow := Workflow{
		lock:         sync.Mutex{},
		context:      &context,
		procedureMap: make(map[string]*Procedure),
		pause:        sync.WaitGroup{},
		finish:       sync.WaitGroup{},
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
	features := make(map[string]*Feature, len(wf.procedureMap))
	for name, procedure := range wf.procedureMap {
		features[name] = procedure.run()
	}
	wf.features = features
	return features
}

func (wf *Workflow) AddProcedure(name string, conf *ProduceConfig) *Procedure {
	procedureCtx := Context{
		scope:         ProcedureCtx,
		scopeContexts: make(map[string]*Context),
		table:         sync.Map{},
	}
	procedureCtx.scopeContexts[ProcedureCtx] = &procedureCtx

	procedure := Procedure{
		name:              name,
		stepMap:           make(map[string]*Step),
		procedureContexts: procedureCtx.scopeContexts,
		context:           &procedureCtx,
		pause:             sync.WaitGroup{},
		running:           sync.WaitGroup{},
		conf:              conf,
	}

	wf.procedureMap[procedure.name] = &procedure
	wf.finish.Add(1)
	procedure.context.parents = append(procedure.context.parents, wf.context)

	return &procedure
}
