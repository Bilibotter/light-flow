package light_flow

import (
	"fmt"
	"math/bits"
	"sync"
	"sync/atomic"
	"time"
)

const (
	publicIndex   uint64 = 0
	resultIndex   uint64 = 62
	internalIndex uint64 = 63
	fullAccess    uint64 = 1<<62 - 1
	resultMark    uint64 = 1 << resultIndex
	internalMark  uint64 = 1 << 63
)

// these constants are used to indicate the position of the process
var (
	endE     = &StatusEnum{0b1, "end"}
	headE    = &StatusEnum{0b1 << 1, "head"}
	hasNextE = &StatusEnum{0b1 << 2, "hasNext"}
	mergedE  = &StatusEnum{0b1 << 3, "merged"}
)

// these variable are used to indicate the state of the unit
var (
	normal       = []*StatusEnum{Pending, Running, Pause, Success}
	Pending      = &StatusEnum{0, "Pending"}
	Running      = &StatusEnum{0b1, "Running"}
	Pause        = &StatusEnum{0b1 << 1, "Pause"}
	skipped      = &StatusEnum{0b1 << 2, "skip"}
	executed     = &StatusEnum{0b1 << 3, "executed"}
	Recovering   = &StatusEnum{0b1 << 4, "Recovering"}
	Suspend      = &StatusEnum{0b1 << 5, "Suspend"}
	Success      = &StatusEnum{0b1 << 15, "Success"}
	NormalMask   = &StatusEnum{0b1<<16 - 1, "NormalMask"}
	abnormal     = []*StatusEnum{Cancel, Timeout, Panic, Error, Stop, CallbackFail, Failed}
	Cancel       = &StatusEnum{0b1<<31 | 0b1<<16, "Cancel"}
	Timeout      = &StatusEnum{0b1<<31 | 0b1<<17, "Timeout"}
	Panic        = &StatusEnum{0b1<<31 | 0b1<<18, "Panic"}
	Error        = &StatusEnum{0b1<<31 | 0b1<<19, "Error"}
	Stop         = &StatusEnum{0b1<<31 | 0b1<<20, "Stop"}
	CallbackFail = &StatusEnum{0b1<<31 | 0b1<<21, "CallbackFail"}
	Failed       = &StatusEnum{0b1 << 31, "Failed"}
	// An abnormal step state will cause the cancellation of dependent unexecuted steps.
	AbnormalMask = &StatusEnum{NormalMask.flag << 16, "AbnormalMask"}
)

type state int64

type FlowController interface {
	flowRuntime
	Process(name string) (process ProcController, exist bool)
	Done() FinishedWorkFlow
	Resume() FlowController
	Pause() FlowController
	Stop() FlowController
}

type WorkFlow interface {
	flowRuntime
	proto
}

type FinishedWorkFlow interface {
	flowRuntime
	Recover() (FinishedWorkFlow, error)
	Exceptions() []FinishedProcess
}

type flowRuntime interface {
	runtimeI
	// HasAny checks if any of the steps within a given workflow are in any of the specified states.
	HasAny(...*StatusEnum) bool
}

type Process interface {
	procCtx
	procRuntime
	proto
}

type FinishedProcess interface {
	procRuntime
	Exceptions() []FinishedStep
}

type ProcController interface {
	procRuntime
	Stop() ProcController
	Pause() ProcController
	Resume() ProcController
	Step(name string) (step StepController, exist bool)
}

type procCtx interface {
	context
	resManagerI
	GetByStepName(stepName, key string) (value any, exist bool)
}

type procRuntime interface {
	runtimeI
	FlowID() string
	// HasAny checks if any of the steps within a given process are in any of the specified states.
	HasAny(...*StatusEnum) bool
}

type Step interface {
	stepCtx
	stepRuntime
	proto
}

type FinishedStep interface {
	stepRuntime
	GetResult() any
}

type StepController interface {
	stepRuntime
}

type stepCtx interface {
	context
	resManagerI
	SetCondition(value any, targets ...string) // the targets contain names of the evaluators to be matched
	EndValues(key string) map[string]any
	Result(key string) (value any, exist bool)
}

type stepRuntime interface {
	runtimeI
	FlowID() string
	ProcessID() string
	Dependents() (stepNames []string)
	Err() error
}

type runtimeI interface {
	statusI
	nameI
	identifierI
	periodI
}

type resManagerI interface {
	Attach(resName string, initParam any) (Resource, error)
	Acquire(resName string) (Resource, bool)
}

type statusI interface {
	ExplainStatus() []string
	Has(enum ...*StatusEnum) bool
	Success() bool
	append(enum *StatusEnum) (updated bool)
	add(flag state) (updated bool)
}

type nameI interface {
	Name() string
}

type identifierI interface {
	ID() string
}

type periodI interface {
	StartTime() time.Time
	EndTime() time.Time
	CostTime() time.Duration
}

type context interface {
	Get(key string) (value any, exist bool)
	Set(key string, value any)
}

type routing interface {
	nodeInfo
	Prefer(key string) (index uint64, exist bool)
	Visible() uint64
	Access() uint64
}

type nodeInfo interface {
	ToIndex(name string) (uint64, bool)
	ToName(index uint64) string
	nodeName() string
}

type StatusEnum struct {
	flag state
	msg  string
}

type simpleContext struct {
	sync.RWMutex
	table    map[string]any
	internal map[string]any
	ctxName  string
}

type dependentContext struct {
	*linkedTable
	routing
	prev context
}

type linkedTable struct {
	sync.RWMutex
	nodes map[string]*node
}

type node struct {
	Path  uint64 // combine current node connected node's index, corresponding bit will be set to 1
	Value any
	Next  *node
}

type outcome struct {
	Result  interface{}
	Named   []evalValue
	Unnamed []evalValue
}

type nodeRouter struct {
	toName   map[uint64]string // index to name
	toIndex  map[string]uint64 // name to toIndex
	priority map[string]uint64 // specify the  key to index
	index    uint64
	nodePath uint64
}

type evalValue struct {
	Matches *set[string]
	Value   interface{}
}

func emptyStatus() *state {
	return createStatus(0)
}

func createStatus(i int64) *state {
	s := state(i)
	return &s
}

func toStepName(value any) string {
	switch result := value.(type) {
	case func(ctx Step) (any, error):
		return getFuncName(value)
	case string:
		return result
	default:
		panic("depend must be func(ctx context) (any, error) or string")
	}
}

// Success return true if finish running and success
func (s *state) Success() bool {
	return s.Has(Success) && s.Normal()
}

// Normal return true if not exception occur
func (s *state) Normal() bool {
	return s.load()&AbnormalMask.flag == 0
}

func (s *state) Has(enum ...*StatusEnum) bool {
	flag := state(0)
	for _, e := range enum {
		flag |= e.flag
	}
	// can't use s.load()&enum.flag != 0, because enum.flag may be 0
	return s.load()&flag == flag
}

// Exceptions return contain exception's message
func (s *state) Exceptions() []string {
	if s.Normal() {
		return nil
	}
	return s.ExplainStatus()
}

// ExplainStatus function explains the state represented by the provided bitmask.
// The function checks the state against predefined abnormal and normal flags,
// and returns a slice of strings containing the names of the matching flags.
// Parameter state is the bitmask representing the state.
// The returned slice contains the names of the matching flags in the layer they were found.
// If abnormal flags are found, normal flags will be ignored.
func (s *state) ExplainStatus() []string {
	compress := make([]string, 0)

	for _, enum := range abnormal {
		if s.Has(enum) {
			compress = append(compress, enum.Message())
		}
	}
	if len(compress) > 0 {
		return compress
	}

	if s.Has(Success) {
		return []string{Success.Message()}
	}

	for _, enum := range normal {
		if s.Has(enum) {
			compress = append(compress, enum.Message())
		}
	}

	return compress
}

// clear function pops a state bit from the specified address.
// The function checks if the specified state bit exists in the current value.
// If it exists, it removes the state bit, and returns true indicating successful removal of the state bit.
// Otherwise, it returns false.
func (s *state) clear(enum *StatusEnum) bool {
	for current := s.load(); enum.flag&current != 0; {
		if s.cas(current, current^enum.flag) {
			return true
		}
	}
	return false
}

func (s *state) append(enum *StatusEnum) (updated bool) {
	return s.add(enum.flag)
}

func (s *state) add(flag state) (updated bool) {
	for current := s.load(); current.load()&flag != flag; current = s.load() {
		if s.cas(current, current|flag) {
			return true
		}
	}
	return
}

func (s *state) load() state {
	return state(atomic.LoadInt64((*int64)(s)))
}

func (s *state) cas(old, new state) bool {
	return atomic.CompareAndSwapInt64((*int64)(s), int64(old), int64(new))
}

func (s *StatusEnum) Message() string {
	return s.msg
}

func (sc *simpleContext) Get(key string) (value any, exist bool) {
	value, exist = sc.table[key]
	return
}

func (sc *simpleContext) Set(key string, value any) {
	sc.table[key] = value
}

func (sc *simpleContext) setInternal(key string, value any) {
	sc.Lock()
	defer sc.Unlock()
	sc.internal[key] = value
}

func (sc *simpleContext) getInternal(key string) (value any, exist bool) {
	sc.RLock()
	defer sc.RUnlock()
	value, exist = sc.internal[key]
	return
}

func (ctx *dependentContext) Get(key string) (value any, exist bool) {
	if index, match := ctx.Prefer(key); match {
		return ctx.matchByIndex(index, key)
	}
	if value, exist = ctx.get(ctx.Access(), key); exist {
		return
	}
	if ctx.prev != nil {
		return ctx.prev.Get(key)
	}
	return nil, false
}

func (ctx *dependentContext) Set(key string, value any) {
	ctx.set(ctx.Visible(), key, value)
}

func (ctx *dependentContext) getInternal(key string) (value any, exist bool) {
	return ctx.matchByIndex(internalIndex, key)
}

func (ctx *dependentContext) setInternal(key string, value any) {
	ctx.set(internalMark, key, value)
}

func (ctx *dependentContext) EndValues(key string) map[string]any {
	ctx.RLock()
	defer ctx.RUnlock()
	current, find := ctx.nodes[key]
	if !find {
		return nil
	}
	m := make(map[string]any)
	exist := uint64(0)
	for current.Next != nil {
		if ctx.Visible()&current.Path != current.Path || exist|current.Path == exist {
			current = current.Next
			continue
		}
		length := bits.Len64(current.Path)
		name := ctx.ToName(uint64(length) - 1)
		m[name] = current.Value
		exist |= current.Path
		current = current.Next
	}
	return m
}

func (ctx *dependentContext) Result(key string) (value any, exist bool) {
	index, valid := ctx.ToIndex(key)
	if !valid {
		return nil, false
	}
	access := uint64(1 << index)
	if ctx.Access()&access != access {
		return nil, false
	}
	ptr, find := ctx.getOutCome(key)
	defer ctx.RUnlock()
	if !find {
		return nil, false
	}
	return ptr.Result, true
}

func (ctx *dependentContext) setResult(key string, value any) {
	ptr := ctx.setOutcomeIfAbsent(key)
	defer ctx.Unlock()
	ptr.Result = value
}

func (ctx *dependentContext) getCondition(key string) (named, unnamed []evalValue, exist bool) {
	ptr, find := ctx.getOutCome(key)
	defer ctx.RUnlock()
	if !find {
		return nil, nil, false
	}
	return ptr.Named, ptr.Unnamed, true
}

func (ctx *dependentContext) SetCondition(value any, targets ...string) {
	ptr := ctx.setOutcomeIfAbsent(ctx.nodeName())
	defer ctx.Unlock()
	switch value.(type) {
	case int8, int16, int32, int64, int:
		value = toInt64(value)
	case uint8, uint16, uint32, uint64, uint:
		value = toUint64(value)
	case float32, float64:
		value = toFloat64(value)
	}
	insert := evalValue{Value: value}
	if len(targets) != 0 {
		insert.Matches = createSetBySliceFunc(targets, func(value string) string { return value })
	}
	ptr.Unnamed = append(ptr.Unnamed, insert)
}

func (ctx *dependentContext) getOutCome(name string) (*outcome, bool) {
	// make sure panic will never occur,
	// otherwise lock will never be released and will block permanently
	ctx.RLock()
	find, ok := ctx.nodes[name]
	if !ok {
		return nil, false
	}
	wrap, ok := find.matchByHighest(resultMark)
	if !ok {
		return nil, false
	}
	return wrap.(*outcome), true
}

func (ctx *dependentContext) setOutcomeIfAbsent(key string) *outcome {
	ctx.Lock()
	if find, exist := ctx.nodes[key]; exist {
		wrap, ok := find.matchByHighest(resultMark)
		if ok {
			unwrap := wrap.(*outcome)
			return unwrap
		}
	}
	ptr := &outcome{}
	head := &node{
		Value: ptr,
		Path:  resultMark,
		Next:  ctx.nodes[key],
	}
	ctx.nodes[key] = head
	return ptr
}

func (ctx *dependentContext) GetByStepName(stepName, key string) (value any, exist bool) {
	ctx.RLock()
	defer ctx.RUnlock()
	index, ok := ctx.ToIndex(stepName)
	if !ok {
		panic(fmt.Sprintf("Step[ %s ] not found.", stepName))
	}
	return ctx.matchByIndex(index, key)
}

// Get method retrieves the value associated with the given key from the context path.
// The method first checks the priority context, then own context, finally parents context.
// Returns the value associated with the key (if found) and a boolean indicating its presence.
func (t *linkedTable) get(path uint64, key string) (any, bool) {
	t.RLock()
	defer t.RUnlock()
	if head, exist := t.nodes[key]; exist {
		if value, find := head.search(path); find {
			return value, find
		}
	}
	return nil, false
}

// set method sets the value associated with the given key in own context.
func (t *linkedTable) set(path uint64, key string, value any) {
	t.Lock()
	defer t.Unlock()
	n := &node{
		Value: value,
		Path:  path,
		Next:  t.nodes[key],
	}
	t.nodes[key] = n
}

func (t *linkedTable) matchByIndex(index uint64, key string) (any, bool) {
	t.RLock()
	defer t.RUnlock()
	head, exist := t.nodes[key]
	if !exist {
		return nil, false
	}
	if index == resultMark {
		return head.matchByHighest(0)
	}
	return head.matchByHighest(1 << index)
}

// matchByHighest will exact search node.
// In most cases, the position of the highest bit is the index.
func (n *node) matchByHighest(highest uint64) (any, bool) {
	// Check if the highest set bit in "highest" is equal to the highest set bit in "n.Path".
	// If the condition is true, the value of node is set by the step corresponding to highest.
	if bits.Len64(highest) == bits.Len64(n.Path) {
		return n.Value, true
	}
	if n.Next == nil {
		return nil, false
	}
	return n.Next.matchByHighest(highest)
}

func (n *node) search(path uint64) (any, bool) {
	if path&n.Path == n.Path {
		return n.Value, true
	}
	if n.Next == nil {
		return nil, false
	}
	return n.Next.search(path)
}

func (router *nodeRouter) Prefer(key string) (index uint64, exist bool) {
	index, exist = router.priority[key]
	return
}

func (router *nodeRouter) Visible() uint64 {
	if router.nodePath == fullAccess {
		return publicIndex
	}
	return router.nodePath
}

func (router *nodeRouter) Access() uint64 {
	return router.nodePath
}

func (router *nodeRouter) ToIndex(name string) (index uint64, exist bool) {
	index, exist = router.toIndex[name]
	return
}

func (router *nodeRouter) ToName(index uint64) string {
	return router.toName[index]
}

func (router *nodeRouter) nodeName() string {
	return router.ToName(router.index)
}
