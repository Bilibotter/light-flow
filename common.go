package light_flow

import (
	"fmt"
	"runtime/debug"
	"sort"
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
	End     = int64(0b1)
	Start   = int64(0b1 << 1)
	HasNext = int64(0b1 << 2)
	Merged  = int64(0b1 << 3)
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
	Stop       = int64(0b1 << 20)
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
		Stop:    "Stop",
		Failed:  "Failed",
	}
)

type Status interface {
	Success() bool
	Exceptions() []string
}

type Info interface {
	Status
	addr() *int64
	GetId() string
	GetName() string
	GetStatus() int64
}

type BasicInfo struct {
	Id     string
	Name   string
	status *int64
}

type CallbackChain[T Info] struct {
	filters []*Callback[T]
}

type Callback[T Info] struct {
	// If must is false, the process will continue to execute
	// even if the processor fails
	must bool
	flag string
	// If keepOn is false, the next processors will not be executed
	// If execute success, info.Exceptions() will be nil
	run func(info T) (keepOn bool)
}

type Context struct {
	name      string
	table     sync.Map // current node's context
	parents   []*Context
	scopes    []string // todo use scope to provide more connect and isolate
	scopeCtxs map[string]*Context
	priority  map[string]string
}

type Feature struct {
	running *sync.WaitGroup
	status  *int64
	finish  *sync.WaitGroup
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

func Contains(status int64, matched int64) bool {
	return status&matched == matched
}

func contains(addr *int64, status int64) bool {
	return atomic.LoadInt64(addr)&status == status
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
// The returned slice contains the names of the matching flags in the layer they were found.
// If abnormal flags are found, normal flags will be ignored.
func ExplainStatus(status int64) []string {
	compress := make([]string, 0)

	for flag, name := range abnormal {
		if flag&status == flag {
			compress = append(compress, name)
		}
	}
	if len(compress) > 0 {
		return compress
	}

	if status&Success == Success {
		return []string{"Success"}
	}

	for flag, name := range normal {
		if flag&status == flag {
			compress = append(compress, name)
		}
	}

	return compress
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

func (bi *BasicInfo) GetName() string {
	return bi.Name
}

func (bi *BasicInfo) GetStatus() int64 {
	return atomic.LoadInt64(bi.status)
}

func (bi *BasicInfo) addr() *int64 {
	return bi.status
}

func (bi *BasicInfo) GetId() string {
	return bi.Id
}

func (bi *BasicInfo) Success() bool {
	return atomic.LoadInt64(bi.status)&Success == Success && IsStatusNormal(atomic.LoadInt64(bi.status))
}

func (bi *BasicInfo) Exceptions() []string {
	if bi.Success() {
		return nil
	}
	return ExplainStatus(atomic.LoadInt64(bi.status))
}

func (cc *CallbackChain[T]) CopyChain() []*Callback[T] {
	result := make([]*Callback[T], 0, len(cc.filters))
	for _, filter := range cc.filters {
		result = append(result, filter)
	}
	return result
}

func (cc *CallbackChain[T]) Add(flag string, must bool, run func(info T) bool) *Callback[T] {
	callback := &Callback[T]{
		must: must,
		flag: flag,
		run:  run,
	}

	cc.filters = append(cc.filters, callback)
	cc.sort()
	return callback
}

func (cc *CallbackChain[T]) sort() {
	sort.SliceStable(cc.filters, func(i, j int) bool {
		if cc.filters[i].must == cc.filters[j].must {
			return i < j
		}
		return cc.filters[i].must
	})
}

func (cc *CallbackChain[T]) process(flag string, info T) (panicStack string) {
	var keepOn bool
	for _, filter := range cc.filters {
		keepOn, panicStack = filter.invoke(flag, info)
		// len(panicStack) != 0 when callback that must be executed encounters panic
		if len(panicStack) != 0 {
			AppendStatus(info.addr(), Panic)
			return
		}
		if !keepOn {
			return
		}
	}

	return
}

func (c *Callback[T]) OnlyFor(name ...string) *Callback[T] {
	s := CreateFromSliceFunc(name, func(value string) string { return value })
	f := func(info T) bool {
		if !s.Contains(info.GetName()) {
			return true
		}
		return c.run(info)
	}

	c.run = f
	return c
}

func (c *Callback[T]) When(status ...int64) *Callback[T] {
	f := func(info T) bool {
		for _, match := range status {
			if Contains(info.GetStatus(), match) {
				return c.run(info)
			}
		}
		return true
	}
	c.run = f
	return c
}

func (c *Callback[T]) invoke(flag string, info T) (keepOn bool, panicStack string) {
	if c.flag != flag {
		return true, ""
	}

	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if !c.must {
			return
		}

		// this will lead to turn process status to panic
		panicStack = fmt.Sprintf("panic: %v\n\n%s", r, string(debug.Stack()))
	}()

	keepOn = c.run(info)
	return
}

// Set method sets the value associated with the given key in own context.
func (ctx *Context) Set(key string, value any) {
	ctx.table.Store(key, value)
}

// Get method retrieves the value associated with the given key from the context path.
// The method first checks the priority context, then own context, finally parents context.
// Returns the value associated with the key (if found) and a boolean indicating its presence.
func (ctx *Context) Get(key string) (any, bool) {
	s := NewRoutineUnsafeSet()
	return ctx.search(key, s)
}

func (ctx *Context) search(key string, prev *Set) (any, bool) {
	if prev.Contains(ctx.name) {
		return nil, false
	}
	prev.Add(ctx.name)
	if target, find := ctx.getByPriority(key); find {
		return target, true
	}

	if used, exist := ctx.table.Load(key); exist {
		return used, true
	}

	for _, parent := range ctx.parents {
		used, exist := parent.search(key, prev)
		if exist {
			return used, exist
		}
	}

	return nil, false
}

// Exposed method exposes a key-value pair to the scope,
// so that units within the scope (steps in the process) can access it.
func (ctx *Context) Exposed(key string, value any) {
	ctx.scopeCtxs[ProcessCtx].Set(key, value)
}

// GetStepResult method retrieves the result of a step's execution.
// Each time a step is executed,
// its execution result is saved in the context of the process.
// Use this method to retrieve the execution result of a step.
func (ctx *Context) GetStepResult(name string) (any, bool) {
	key := InternalPrefix + name
	return ctx.scopeCtxs[ProcessCtx].Get(key)
}

func (ctx *Context) setStepResult(name string, value any) {
	key := InternalPrefix + name
	ctx.scopeCtxs[ProcessCtx].Set(key, value)
}

func (ctx *Context) getByPriority(key string) (any, bool) {
	name, exist := ctx.priority[key]
	if !exist {
		return nil, false
	}
	return ctx.scopeCtxs[name].search(key, NewRoutineUnsafeSet())
}

// Done method waits for the corresponding process to complete.
func (f *Feature) Done() {
	f.running.Wait()
	f.finish.Wait()
}

func (f *Feature) Success() bool {
	return *f.status&Success == Success && IsStatusNormal(atomic.LoadInt64(f.status))
}

// See common.ExplainStatus for details.
func (f *Feature) ExplainStatus() []string {
	return ExplainStatus(atomic.LoadInt64(f.status))
}
