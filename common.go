package light_flow

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// these constants are used to indicate the scope of the context
const (
	InternalPrefix = "~"
	WorkflowCtx    = "~workflow~"
	ProcessCtx     = "~process~"
)

// these constants are used to indicate the position of the process
const (
	NoUse = int64(-1)
	End   = int64(0)
	Start = int64(1)
)

// these constants are used to indicate the status of the process
const (
	Pending    = int64(0)
	Running    = int64(0b1)
	Pause      = int64(0b1 << 1)
	Success    = int64(0b1 << 15)
	NormalMask = int64(0b1<<16 - 1)
	Cancel     = int64(0b1 << 16)
	Timeout    = int64(0b1 << 17)
	Panic      = int64(0b1 << 18)
	Error      = int64(0b1 << 19)
	Failed     = int64(0b1 << 31)
	// abnormal produce status will cause the cancellation of unexecuted steps during execution
	// abnormal step status will cause the cancellation of unexecuted steps that depend on it
	AbnormalMask = NormalMask << 16
)

var (
	normal = map[int64]string{
		Pending: "Pending",
		Running: "Running",
		Pause:   "Pause",
		Success: "Success",
	}

	abnormal = map[int64]string{
		Cancel:  "Cancel",
		Timeout: "Timeout",
		Panic:   "Panic",
		Error:   "Error",
		Failed:  "Failed",
	}
)

type Context struct {
	name          string
	scope         string
	scopeContexts map[string]*Context
	parents       []*Context
	table         sync.Map // current node's context
	priority      map[string]any
}

type Feature struct {
	running *sync.WaitGroup
	status  *int64
	finish  *int64
}

// PopStatus function pops a status bit from the specified address.
// The function checks if the specified status bit exists in the current value.
// If it exists, it removes the status bit, and returns true indicating successful removal of the status bit.
// Otherwise, it returns false.
func PopStatus(addr *int64, status int64) (change bool) {
	for current := atomic.LoadInt64(addr); status&current == status; {
		if atomic.CompareAndSwapInt64(addr, current, current^status) {
			return true
		}
	}
	return false
}

// AppendStatus function appends a status bit to the specified address.
// The function checks if the specified update bit is already present in the current value.
// If it is not present, it appends the update bit,
// and returns true indicating successful appending of the update bit.
// Otherwise, it returns false.
func AppendStatus(addr *int64, update int64) (change bool) {
	for current := atomic.LoadInt64(addr); current&update != update; {
		if atomic.CompareAndSwapInt64(addr, current, current|update) {
			return true
		}
		current = atomic.LoadInt64(addr)
	}
	return false
}

func IsStatusNormal(status int64) bool {
	if status&AbnormalMask == 0 {
		return true
	}
	return false
}

// ExplainStatus function explains the status represented by the provided bitmask.
// The function checks the status against predefined abnormal and normal flags,
// and returns a slice of strings containing the names of the matching flags.
// Parameter status is the bitmask representing the status.
// The returned slice contains the names of the matching flags in the order they were found.
// If abnormal flags are found, normal flags will be ignored.
func ExplainStatus(status int64) []string {
	contains := make([]string, 0)

	for flag, name := range abnormal {
		if flag&status == flag {
			contains = append(contains, name)
		}
	}
	if len(contains) > 0 {
		return contains
	}

	if status&Success == Success {
		return []string{"Success"}
	}

	for flag, name := range normal {
		if flag&status == flag {
			contains = append(contains, name)
		}
	}

	return contains
}

func toStepName(value any) string {
	var result string
	switch value.(type) {
	case func(ctx *Context) (any, error):
		result = GetFuncName(value)
	case string:
		result = value.(string)
	default:
		panic("value must be func(ctx *Context) (any, error) or string")
	}
	return result
}

// Set method sets the value associated with the given key in own context.
func (ctx *Context) Set(key string, value any) {
	ctx.table.Store(key, value)
}

// Get method retrieves the value associated with the given key from the context path.
// The method first checks the priority context, then own context, finally parents context.
// Returns the value associated with the key (if found) and a boolean indicating its presence.
func (ctx *Context) Get(key string) (any, bool) {
	if used, exist := ctx.getCtxByPriority(key); exist {
		return used.Get(key)
	}

	if used, exist := ctx.table.Load(key); exist {
		return used, true
	}

	for _, parent := range ctx.parents {
		used, exist := parent.Get(key)
		if exist {
			return used, exist
		}
	}

	return nil, false
}

// Exposed method exposed key-value to scope,
// so that unit in scope(step in process) can get it.
func (ctx *Context) Exposed(key string, value any) {
	ctx.scopeContexts[ctx.scope].Set(key, value)
}

// GetStepResult method retrieves the step execute result.
// Each time a step is executed,
// the execution results are saved in the context of the process.
// So you could get the step execute result by this method.
func (ctx *Context) GetStepResult(name string) (any, bool) {
	key := InternalPrefix + name
	return ctx.Get(key)
}

func (ctx *Context) setStepResult(name string, value any) {
	key := InternalPrefix + name
	ctx.scopeContexts[ctx.scope].Set(key, value)
}

// checkPriority check if priority key can correspond to an existing step
// If not it will panic.
func (ctx *Context) checkPriority() {
	for key, value := range ctx.priority {
		// check if priority's value can correspond to an existing step
		ctx.getCtxByPriority(key)
		if _, find := ctx.backTrackSearch(toStepName(value)); !find {
			panic("priority value must be a step that can be back tracking by the current step")
		}
	}
}

func (ctx *Context) getCtxByPriority(key string) (*Context, bool) {
	value, exist := ctx.priority[key]
	if !exist {
		return nil, false
	}
	name := toStepName(value)
	used, exist := ctx.scopeContexts[name]
	if !exist {
		panic(fmt.Sprintf("can't find step named %s", name))
	}
	return used, true
}

func (ctx *Context) backTrackSearch(parentName string) (*Context, bool) {
	if len(ctx.parents) == 0 {
		return nil, false
	}
	for _, parent := range ctx.parents {
		if parent.name == parentName {
			return parent, true
		}
		if candidate, find := parent.backTrackSearch(parentName); find {
			return candidate, true
		}
	}
	return nil, false
}

// Done method wait for the corresponding process to complete.
func (f *Feature) Done() {
	f.running.Wait()
	// wait goroutine to change status
	for finish := atomic.LoadInt64(f.finish); finish == 0; finish = atomic.LoadInt64(f.finish) {
	}
}

func (f *Feature) Success() bool {
	return *f.status&Success == Success && IsStatusNormal(atomic.LoadInt64(f.status))
}

// See common.ExplainStatus
func (f *Feature) ExplainStatus() []string {
	return ExplainStatus(atomic.LoadInt64(f.status))
}
