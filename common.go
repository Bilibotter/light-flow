package light_flow

import (
	"fmt"
	"math/bits"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

const (
	publicKey   int64 = 0
	resultIndex int64 = 62
	visibleAll  int64 = 1<<62 - 1
	resultMark  int64 = 1 << resultIndex
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
	Pending      = &StatusEnum{0, "Pending"}
	Running      = &StatusEnum{0b1, "Running"}
	Pause        = &StatusEnum{0b1 << 1, "Pause"}
	skipped      = &StatusEnum{0b1 << 2, "skip"}
	Success      = &StatusEnum{0b1 << 15, "Success"}
	NormalMask   = &StatusEnum{0b1<<16 - 1, "NormalMask"}
	Cancel       = &StatusEnum{0b1 << 16, "Cancel"}
	Timeout      = &StatusEnum{0b1 << 17, "Timeout"}
	Panic        = &StatusEnum{0b1 << 18, "Panic"}
	Error        = &StatusEnum{0b1 << 19, "Error"}
	Stop         = &StatusEnum{0b1 << 20, "Stop"}
	CallbackFail = &StatusEnum{0b1 << 21, "CallbackFail"}
	Failed       = &StatusEnum{0b1 << 31, "Failed"}
	// AbnormalMask An abnormal step state will cause the cancellation of dependent unexecuted steps.
	AbnormalMask = &StatusEnum{NormalMask.flag << 16, "AbnormalMask"}
)

var (
	normal   = []*StatusEnum{Pending, Running, Pause, Success}
	abnormal = []*StatusEnum{Cancel, Timeout, Panic, Error, Stop, CallbackFail, Failed}
)

type state int64

type statusI interface {
	Has(enum *StatusEnum) bool
	Success() bool
	Exceptions() []string
}

type basicInfoI interface {
	statusI
	addr() *state
	GetId() string
	GetName() string
}

type context interface {
	ContextName() string
	Get(key string) (value any, exist bool)
	getByName(name, key string)
	Set(key string, value any)
}

type StepCtx interface {
	context
	GetEndValues(key string) map[string]any
	GetResult(key string) (value any, exist bool)
	setResult(key string, value any)
	SetCond(value any)
	getCond(key string) (named, unnamed evalValues, exist bool)
}

type procCtx interface {
	context
	GetByStepName(stepName, key string) (value any, exist bool)
}

type StatusEnum struct {
	flag state
	msg  string
}

type basicInfo struct {
	*state
	Id   string
	Name string
}

type handler[T basicInfoI] struct {
	filter []*callback[T]
}

type callback[T basicInfoI] struct {
	// If must is false, the process will continue to execute
	// even if the processor fails
	must bool
	flag string
	// If keepOn is false, the next processors will not be executed
	// If execute success, info.Exceptions() will be nil
	run func(info T) (keepOn bool, err error)
}

type simpleContext struct {
	m    map[string]any
	name string
}

type visibleContext struct {
	*linkedTable
	*accessInfo // step accessInfo will update when mergeConfig, so using pointer
	prev        context
	next        context
}

type accessInfo struct {
	names    map[int64]string // index to name
	indexes  map[string]int64 // name to indexes
	priority map[string]int64 // specify the  key to index
	index    int64
	passing  int64
}

type linkedTable struct {
	lock  sync.RWMutex
	nodes map[string]*node
}

type node struct {
	Passing int64 // combine current node connected node's index, corresponding bit will be set to 1
	Value   any
	Next    *node
}

type Future struct {
	*state
	*basicInfo
	finish *sync.WaitGroup
}

type outcome struct {
	Result  interface{}
	named   evalValues
	unnamed evalValues
}

type evalValues []*evalValue

type evalValue struct {
	matches *set[string]
	value   interface{}
}

func emptyStatus() *state {
	return createStatus(0)
}

func createStatus(i int64) *state {
	s := state(i)
	return &s
}

func toStepName(value any) string {
	var result string
	switch value.(type) {
	case func(ctx StepCtx) (any, error):
		result = getFuncName(value)
	case string:
		result = value.(string)
	default:
		panic("value must be func(ctx context) (any, error) or string")
	}
	return result
}

// Success return true if finish running and success
func (s *state) Success() bool {
	return s.Has(Success) && s.Normal()
}

// Normal return true if not exception occur
func (s *state) Normal() bool {
	return s.load()&AbnormalMask.flag == 0
}

func (s *state) Has(enum *StatusEnum) bool {
	// can't use s.load()&enum.flag != 0, because enum.flag may be 0
	return s.load()&enum.flag == enum.flag
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

func (s *state) set(enum *StatusEnum) bool {
	return s.add(enum.flag)
}

func (s *state) add(flag state) bool {
	for current := s.load(); current.load()&flag != flag; current = s.load() {
		if s.cas(current, current|flag) {
			return true
		}
	}
	return false
}

func (s *state) load() state {
	return state(atomic.LoadInt64((*int64)(s)))
}

func (s *state) cas(old, new state) bool {
	return atomic.CompareAndSwapInt64((*int64)(s), int64(old), int64(new))
}

func (s *StatusEnum) Contained(explain ...string) bool {
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

func (bi *basicInfo) GetName() string {
	return bi.Name
}

func (bi *basicInfo) addr() *state {
	return bi.state
}

func (bi *basicInfo) GetId() string {
	return bi.Id
}

func (cc *handler[T]) clone() handler[T] {
	config := handler[T]{}
	config.filter = cc.copyFilters()
	return config
}

func (cc *handler[T]) copyFilters() []*callback[T] {
	result := make([]*callback[T], 0, len(cc.filter))
	for _, filter := range cc.filter {
		result = append(result, filter)
	}
	return result
}

func (cc *handler[T]) addCallback(flag string, must bool, run func(info T) (bool, error)) *callback[T] {
	if len(cc.filter) > 0 {
		last := cc.filter[len(cc.filter)-1]
		// essential callback depend on non-essential callback is against Dependency Inversion Principle
		if last.flag == flag && last.must != must && last.must != true {
			panic("essential callback should before non-essential callback")
		}
	}

	filter := &callback[T]{
		must: must,
		flag: flag,
		run:  run,
	}

	cc.filter = append(cc.filter, filter)
	return filter
}

func (cc *handler[T]) combine(chain *handler[T]) {
	for _, filter := range chain.filter {
		cc.filter = append(cc.filter, filter)
	}
}

// don't want to raise error not deal hint, so return string
func (cc *handler[T]) handle(flag string, info T) string {
	for _, filter := range cc.filter {
		keepOn, err, panicStack := filter.call(flag, info)
		if filter.must {
			if len(panicStack) != 0 || err != nil {
				info.addr().set(CallbackFail)
				info.addr().set(Failed)
				return panicStack
			}
		}

		if !keepOn {
			return panicStack
		}
	}

	return ""
}

func (c *callback[T]) NotFor(name ...string) *callback[T] {
	s := createSetBySliceFunc(name, func(value string) string { return value })
	old := c.run
	f := func(info T) (bool, error) {
		if s.Contains(info.GetName()) {
			return true, nil
		}
		return old(info)
	}

	c.run = f
	return c
}

func (c *callback[T]) OnlyFor(name ...string) *callback[T] {
	s := createSetBySliceFunc(name, func(value string) string { return value })
	old := c.run
	f := func(info T) (bool, error) {
		if !s.Contains(info.GetName()) {
			return true, nil
		}
		return old(info)
	}

	c.run = f
	return c
}

func (c *callback[T]) When(status ...*StatusEnum) *callback[T] {
	if c.flag == Before {
		if len(status) > 1 || status[0] != CallbackFail {
			panic("CallbackFail is only valid statusEnum for Before")
		}
	}
	old := c.run
	f := func(info T) (bool, error) {
		for _, match := range status {
			if info.Has(match) {
				return old(info)
			}
		}
		return true, nil
	}
	c.run = f
	return c
}

func (c *callback[T]) Exclude(status ...*StatusEnum) *callback[T] {
	old := c.run
	f := func(info T) (bool, error) {
		for _, match := range status {
			if info.Has(match) {
				return true, nil
			}
		}
		return old(info)
	}
	c.run = f
	return c
}

func (c *callback[T]) call(flag string, info T) (keepOn bool, err error, panicStack string) {
	if c.flag != flag {
		return true, nil, ""
	}

	defer func() {
		r := recover()
		if r != nil {
			// this will lead to turn process state to panic
			panicStack = fmt.Sprintf("panic: %v\n\n%s", r, string(debug.Stack()))
		}

	}()

	keepOn, err = c.run(info)
	return
}

// set method sets the value associated with the given key in own context.
func (t *linkedTable) set(passing int64, key string, value any) {
	t.lock.Lock()
	defer t.lock.Unlock()
	n := &node{
		Value:   value,
		Passing: passing,
		Next:    t.nodes[key],
	}
	t.nodes[key] = n
}

func (t *linkedTable) matchByIndex(index int64, key string) (any, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	head, exist := t.nodes[key]
	if !exist {
		return nil, false
	}
	return head.matchByHighest(1 << index)
}

// Get method retrieves the value associated with the given key from the context path.
// The method first checks the priority context, then own context, finally parents context.
// Returns the value associated with the key (if found) and a boolean indicating its presence.
func (t *linkedTable) get(passing int64, key string) (any, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if head, exist := t.nodes[key]; exist {
		if value, find := head.search(passing); find {
			return value, find
		}
	}
	return nil, false
}

// matchByHighest will exact search node.
// In most cases, the position of the highest bit is the index.
func (n *node) matchByHighest(highest int64) (any, bool) {
	// Check if the highest set bit in "highest" is equal to the highest set bit in "n.Passing".
	// If the condition is true, the value of node is set by the step corresponding to highest.
	if bits.Len64(uint64(highest)) == bits.Len64(uint64(n.Passing)) {
		return n.Value, true
	}
	if n.Next == nil {
		return nil, false
	}
	return n.Next.matchByHighest(highest)
}

func (n *node) search(passing int64) (any, bool) {
	if passing&n.Passing == n.Passing {
		return n.Value, true
	}
	if n.Next == nil {
		return nil, false
	}
	return n.Next.search(passing)
}

func (vc *visibleContext) ContextName() string {
	return vc.names[vc.index]
}

func (vc *visibleContext) Get(key string) (value any, exist bool) {
	if index, match := vc.priority[key]; match {
		return vc.matchByIndex(index, key)
	}
	if value, exist = vc.get(vc.passing, key); exist {
		return
	}
	if vc.prev != nil {
		return vc.prev.Get(key)
	}
	return nil, false
}

func (vc *visibleContext) getByName(name, key string) {
	if vc.indexes == nil {
		panic("This method is not supported.")
	}
	index, ok := vc.indexes[name]
	if !ok {
		panic("Invalid node name.")
	}
	vc.matchByIndex(index, key)
}

func (vc *visibleContext) GetResult(key string) (value any, exist bool) {
	ptr, find := vc.getOutCome(key)
	defer vc.lock.RUnlock()
	if !find {
		return nil, false
	}
	return ptr.Result, true
}

func (vc *visibleContext) getCond(key string) (named, unnamed evalValues, exist bool) {
	ptr, find := vc.getOutCome(key)
	defer vc.lock.RUnlock()
	if !find {
		return nil, nil, false
	}
	return ptr.named, ptr.unnamed, true
}

func (vc *visibleContext) GetEndValues(key string) map[string]any {
	if vc.names == nil {
		panic("This method is not supported.")
	}
	vc.lock.RLock()
	defer vc.lock.RUnlock()
	current, find := vc.nodes[key]
	if !find {
		return nil
	}
	m := make(map[string]any)
	exist := int64(0)
	for current.Next != nil {
		if vc.passing&current.Passing != current.Passing || exist|current.Passing == exist {
			current = current.Next
			continue
		}
		length := bits.Len64(uint64(current.Passing))
		name := vc.names[int64(length)-1]
		m[name] = current.Value
		exist |= current.Passing
		current = current.Next
	}
	return m
}

func (vc *visibleContext) GetByStepName(stepName, key string) (value any, exist bool) {
	if vc.indexes == nil {
		panic("This method is not supported.")
	}
	vc.lock.RLock()
	defer vc.lock.RUnlock()
	index, ok := vc.indexes[stepName]
	if !ok {
		panic(fmt.Sprintf("Step[%s] not found.", stepName))
	}
	return vc.matchByIndex(index, key)
}

func (vc *visibleContext) setResult(key string, value any) {
	ptr := vc.setOutcomeIfNotExist(key)
	defer vc.lock.Unlock()
	ptr.Result = value
}

func (vc *visibleContext) setCond(name string, value any) {
	ptr := vc.setOutcomeIfNotExist(name)
	defer vc.lock.Unlock()
	switch value.(type) {
	case int8, int16, int32, int64, int:
		value = toInt64(value)
	case uint8, uint16, uint32, uint64, uint:
		value = toUint64(value)
	case float32, float64:
		value = toFloat64(value)
	}
	ptr.unnamed = append(ptr.unnamed, &evalValue{value: value})
}

func (vc *visibleContext) getOutCome(name string) (*outcome, bool) {
	// make sure panic will never occur,
	// otherwise lock will never be released and will block permanently
	vc.lock.RLock()
	find, ok := vc.nodes[name]
	if !ok {
		return nil, false
	}
	wrap, ok := find.matchByHighest(resultMark)
	if !ok {
		return nil, false
	}
	return wrap.(*outcome), true
}

func (vc *visibleContext) setOutcomeIfNotExist(key string) *outcome {
	vc.lock.Lock()
	if find, exist := vc.nodes[key]; exist {
		wrap, ok := find.matchByHighest(resultMark)
		if ok {
			unwrap := wrap.(*outcome)
			return unwrap
		}
	}
	ptr := &outcome{}
	head := &node{
		Value:   ptr,
		Passing: resultMark,
		Next:    vc.nodes[key],
	}
	vc.nodes[key] = head
	return ptr
}

func (vc *visibleContext) Set(key string, value any) {
	if vc.passing == visibleAll {
		vc.set(publicKey, key, value)
		return
	}
	vc.set(vc.passing, key, value)
}

func (sc *simpleContext) ContextName() string {
	return sc.name
}

func (sc *simpleContext) Get(key string) (value any, exist bool) {
	value, exist = sc.m[key]
	return
}

func (sc *simpleContext) getByName(name, key string) {
	panic("This method is not supported.")
}

func (sc *simpleContext) Set(key string, value any) {
	sc.m[key] = value
}

// Done method waits for the corresponding process to complete.
func (f *Future) Done() {
	f.finish.Wait()
}
