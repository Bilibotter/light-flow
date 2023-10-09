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
	InternalPrefix = "::"
	WorkflowCtx    = "Flow::"
	ProcessCtx     = "Proc::"
)

// these constants are used to indicate the position of the process
const (
	End     = int64(0b1)
	Start   = int64(0b1 << 1)
	HasNext = int64(0b1 << 2)
	Merged  = int64(0b1 << 3)
)

// these variable are used to indicate the status of the unit
var (
	Pending    = &StatusEnum{0, "Pending"}
	Running    = &StatusEnum{0b1, "Running"}
	Pause      = &StatusEnum{0b1 << 1, "Pause"}
	Success    = &StatusEnum{0b1 << 15, "Success"}
	NormalMask = &StatusEnum{0b1<<16 - 1, "NormalMask"}
	Cancel     = &StatusEnum{0b1 << 16, "Cancel"}
	Timeout    = &StatusEnum{0b1 << 17, "StepTimeout"}
	Panic      = &StatusEnum{0b1 << 18, "Panic"}
	Error      = &StatusEnum{0b1 << 19, "Error"}
	Stop       = &StatusEnum{0b1 << 20, "Stop"}
	Failed     = &StatusEnum{0b1 << 31, "Failed"}
	// AbnormalMask An abnormal step status will cause the cancellation of dependent unexecuted steps.
	AbnormalMask = &StatusEnum{NormalMask.flag << 16, "AbnormalMask"}
)

var (
	normal   = []*StatusEnum{Pending, Running, Pause, Success}
	abnormal = []*StatusEnum{Cancel, Timeout, Panic, Error, Stop, Failed}
)

type Status int64

type StatusI interface {
	Contain(enum *StatusEnum) bool
	Success() bool
	Exceptions() []string
}

type InfoI interface {
	StatusI
	addr() *Status
	GetId() string
	GetName() string
}

type StatusEnum struct {
	flag Status
	msg  string
}

type BasicInfo struct {
	*Status
	Id   string
	Name string
}

type CallbackChain[T InfoI] struct {
	filters []*Callback[T]
}

type Callback[T InfoI] struct {
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
	*Status
	*BasicInfo
	finish *sync.WaitGroup
}

// appendStatus function appends a status bit to the specified address.
// The function checks if the specified update bit is already present in the current value.
// If it is not present, it appends the update bit,
// and returns true indicating successful appending of the update bit.
// Otherwise, it returns false.
func appendStatus(addr *int64, update int64) (change bool) {
	for current := atomic.LoadInt64(addr); current&update != update; {
		if atomic.CompareAndSwapInt64(addr, current, current|update) {
			return true
		}
		current = atomic.LoadInt64(addr)
	}
	return false
}

func emptyStatus() *Status {
	s := Status(0)
	return &s
}

func contains(addr *int64, status int64) bool {
	return atomic.LoadInt64(addr)&status == status
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

func (s *Status) Success() bool {
	return s.Contain(Success) && s.Normal()
}

func (s *Status) Normal() bool {
	return s.load()&AbnormalMask.flag == 0
}

func (s *Status) Contain(enum *StatusEnum) bool {
	// can't use s.load()&enum.flag != 0, because enum.flag may be 0
	return s.load()&enum.flag == enum.flag
}

func (s *Status) Exceptions() []string {
	if s.Normal() {
		return nil
	}
	return s.ExplainStatus()
}

// ExplainStatus function explains the status represented by the provided bitmask.
// The function checks the status against predefined abnormal and normal flags,
// and returns a slice of strings containing the names of the matching flags.
// Parameter status is the bitmask representing the status.
// The returned slice contains the names of the matching flags in the layer they were found.
// If abnormal flags are found, normal flags will be ignored.
func (s *Status) ExplainStatus() []string {
	compress := make([]string, 0)

	for _, enum := range abnormal {
		if s.Contain(enum) {
			compress = append(compress, enum.Message())
		}
	}
	if len(compress) > 0 {
		return compress
	}

	if s.Contain(Success) {
		return []string{Success.Message()}
	}

	for _, enum := range normal {
		if s.Contain(enum) {
			compress = append(compress, enum.Message())
		}
	}

	return compress
}

func (s *Status) combine(status *Status) {
	for current := s.load(); current|status.load() != current; current = s.load() {
		if s.cas(current, current|status.load()) {
			return
		}
	}
}

// Pop function pops a status bit from the specified address.
// The function checks if the specified status bit exists in the current value.
// If it exists, it removes the status bit, and returns true indicating successful removal of the status bit.
// Otherwise, it returns false.
func (s *Status) Pop(enum *StatusEnum) bool {
	for current := s.load(); enum.flag&current != 0; {
		if s.cas(current, current^enum.flag) {
			return true
		}
	}
	return false
}

func (s *Status) AppendStatus(enum *StatusEnum) bool {
	for current := s.load(); !current.Contain(enum); current = s.load() {
		if s.cas(current, current|enum.flag) {
			return true
		}
	}
	return false
}

func (s *Status) load() Status {
	return Status(atomic.LoadInt64((*int64)(s)))
}

func (s *Status) cas(old, new Status) bool {
	return atomic.CompareAndSwapInt64((*int64)(s), int64(old), int64(new))
}

func (s *StatusEnum) Contained(explain []string) bool {
	for _, value := range explain {
		if value == s.msg {
			return true
		}
	}
	return false
}

func (s *StatusEnum) Message() string {
	return s.msg
}

func (bi *BasicInfo) GetName() string {
	return bi.Name
}

func (bi *BasicInfo) addr() *Status {
	return bi.Status
}

func (bi *BasicInfo) GetId() string {
	return bi.Id
}

func (cc *CallbackChain[T]) CopyChain() []*Callback[T] {
	result := make([]*Callback[T], 0, len(cc.filters))
	for _, filter := range cc.filters {
		result = append(result, filter)
	}
	return result
}

func (cc *CallbackChain[T]) AddCallback(flag string, must bool, run func(info T) bool) *Callback[T] {
	callback := &Callback[T]{
		must: must,
		flag: flag,
		run:  run,
	}

	cc.filters = append(cc.filters, callback)
	cc.maintain()
	return callback
}

func (cc *CallbackChain[T]) maintain() {
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
		keepOn, panicStack = filter.call(flag, info)
		// len(panicStack) != 0 when callback that must be executed encounters panic
		if len(panicStack) != 0 {
			info.addr().AppendStatus(Panic)
			return
		}
		if !keepOn {
			return
		}
	}

	return
}

func (c *Callback[T]) NotFor(name ...string) *Callback[T] {
	s := CreateFromSliceFunc(name, func(value string) string { return value })
	old := c.run
	f := func(info T) bool {
		if s.Contains(info.GetName()) {
			return true
		}
		return old(info)
	}

	c.run = f
	return c
}

func (c *Callback[T]) OnlyFor(name ...string) *Callback[T] {
	s := CreateFromSliceFunc(name, func(value string) string { return value })
	old := c.run
	f := func(info T) bool {
		if !s.Contains(info.GetName()) {
			return true
		}
		return old(info)
	}

	c.run = f
	return c
}

func (c *Callback[T]) When(status ...*StatusEnum) *Callback[T] {
	old := c.run
	f := func(info T) bool {
		for _, match := range status {
			if info.Contain(match) {
				return old(info)
			}
		}
		return true
	}
	c.run = f
	return c
}

func (c *Callback[T]) Exclude(status ...*StatusEnum) *Callback[T] {
	old := c.run
	f := func(info T) bool {
		for _, match := range status {
			if info.Contain(match) {
				return true
			}
		}
		return old(info)
	}
	c.run = f
	return c
}

func (c *Callback[T]) call(flag string, info T) (keepOn bool, panicStack string) {
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

func (ctx *Context) Name() string {
	return ctx.name
}

// Set method sets the value associated with the given key in own context.
func (ctx *Context) Set(key string, value any) {
	ctx.table.Store(key, value)
}

// Get method retrieves the value associated with the given key from the context path.
// The method first checks the priority context, then own context, finally parents context.
// Returns the value associated with the key (if found) and a boolean indicating its presence.
func (ctx *Context) Get(key string) (any, bool) {
	s := NewRoutineUnsafeSet[string]()
	return ctx.search(key, s)
}

func (ctx *Context) search(key string, prev *Set[string]) (any, bool) {
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

func (ctx *Context) GetAll(key string) map[string]any {
	saved := make(map[string]any)
	ctx.getAll(key, saved)
	return saved
}

func (ctx *Context) getAll(key string, saved map[string]any) {
	if _, visited := saved[key]; visited {
		return
	}

	if value, exist := ctx.table.Load(key); exist {
		saved[ctx.Name()] = value
	}

	for _, parent := range ctx.parents {
		parent.getAll(key, saved)
	}
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
	return ctx.scopeCtxs[name].search(key, NewRoutineUnsafeSet[string]())
}

// Done method waits for the corresponding process to complete.
func (f *Feature) Done() {
	f.finish.Wait()
}
