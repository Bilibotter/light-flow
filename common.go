package light_flow

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

var (
	visibleAll = Status(0)
	resultMark = Status(1 << 62)
)

// these constants are used to indicate the position of the process
var (
	End     = &StatusEnum{0b1, "End"}
	Head    = &StatusEnum{0b1 << 1, "Head"}
	HasNext = &StatusEnum{0b1 << 2, "HasNext"}
	Merged  = &StatusEnum{0b1 << 3, "Merged"}
)

// these variable are used to indicate the status of the unit
var (
	Pending      = &StatusEnum{0, "Pending"}
	Running      = &StatusEnum{0b1, "Running"}
	Pause        = &StatusEnum{0b1 << 1, "Pause"}
	Success      = &StatusEnum{0b1 << 15, "Success"}
	NormalMask   = &StatusEnum{0b1<<16 - 1, "NormalMask"}
	Cancel       = &StatusEnum{0b1 << 16, "Cancel"}
	Timeout      = &StatusEnum{0b1 << 17, "Timeout"}
	Panic        = &StatusEnum{0b1 << 18, "Panic"}
	Error        = &StatusEnum{0b1 << 19, "Error"}
	Stop         = &StatusEnum{0b1 << 20, "Stop"}
	CallbackFail = &StatusEnum{0b1 << 21, "CallbackFail"}
	Failed       = &StatusEnum{0b1 << 31, "Failed"}
	// AbnormalMask An abnormal step status will cause the cancellation of dependent unexecuted steps.
	AbnormalMask = &StatusEnum{NormalMask.flag << 16, "AbnormalMask"}
)

var (
	normal   = []*StatusEnum{Pending, Running, Pause, Success}
	abnormal = []*StatusEnum{Cancel, Timeout, Panic, Error, Stop, CallbackFail, Failed}
)

type Status int64

type StatusI interface {
	Contain(enum *StatusEnum) bool
	Success() bool
	Exceptions() []string
}

type BasicInfoI interface {
	StatusI
	addr() *Status
	GetId() string
	GetName() string
}

type Context interface {
	ContextName() string
	Get(key string) (value any, exist bool)
	GetAll(key string) map[string]any
	GetResult(key string) (value any, exist bool)
	Set(key string, value any)
}

type StatusEnum struct {
	flag Status
	msg  string
}

type basicInfo struct {
	*Status
	Id   string
	Name string
}

type Handler[T BasicInfoI] struct {
	filter []*callback[T]
}

type callback[T BasicInfoI] struct {
	// If must is false, the process will continue to execute
	// even if the processor fails
	must bool
	flag string
	// If keepOn is false, the next processors will not be executed
	// If execute success, info.Exceptions() will be nil
	run func(info T) (keepOn bool, err error)
}

type visibleContext struct {
	adjacencyTable
	*visitor // step visitor will update when mergeConfig, so using pointer
	parent   *visibleContext
}

type visitor struct {
	roster   map[int32]string // index to name
	priority map[string]int32 // specify the  key to index
	index    int32
	visible  Status
}

type adjacencyTable struct {
	lock  *sync.RWMutex
	nodes map[string]*node
}

type node struct {
	Index   int32  // used to identify a node
	Visible Status // combine current node connected node's index, corresponding bit will be set to 1
	Value   any
	Next    *node
}

type Future struct {
	*Status
	*basicInfo
	finish *sync.WaitGroup
}

func emptyStatus() *Status {
	return createStatus(0)
}

func createStatus(i int64) *Status {
	s := Status(i)
	return &s
}

func toStepName(value any) string {
	var result string
	switch value.(type) {
	case func(ctx Context) (any, error):
		result = GetFuncName(value)
	case string:
		result = value.(string)
	default:
		panic("value must be func(ctx Context) (any, error) or string")
	}
	return result
}

// Success return true if finish running and success
func (s *Status) Success() bool {
	return s.Contain(Success) && s.Normal()
}

// Normal return true if not exception occur
func (s *Status) Normal() bool {
	return s.load()&AbnormalMask.flag == 0
}

func (s *Status) Contain(enum *StatusEnum) bool {
	// can't use s.load()&enum.flag != 0, because enum.flag may be 0
	return s.load()&enum.flag == enum.flag
}

func (s *Status) contain(flag Status) bool {
	return s.load()&flag == flag
}

// Exceptions return contain exception's message
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

func (s *Status) Append(enum *StatusEnum) bool {
	for current := s.load(); !current.Contain(enum); current = s.load() {
		if s.cas(current, current|enum.flag) {
			return true
		}
	}
	return false
}

func (s *Status) append(flag Status) bool {
	for current := s.load(); !current.contain(flag); current = s.load() {
		if s.cas(current, current|flag) {
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

func (bi *basicInfo) addr() *Status {
	return bi.Status
}

func (bi *basicInfo) GetId() string {
	return bi.Id
}

func (cc *Handler[T]) clone() Handler[T] {
	config := Handler[T]{}
	config.filter = cc.copyFilters()
	return config
}

func (cc *Handler[T]) copyFilters() []*callback[T] {
	result := make([]*callback[T], 0, len(cc.filter))
	for _, filter := range cc.filter {
		result = append(result, filter)
	}
	return result
}

func (cc *Handler[T]) addCallback(flag string, must bool, run func(info T) (bool, error)) *callback[T] {
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

func (cc *Handler[T]) combine(chain *Handler[T]) {
	for _, filter := range chain.filter {
		cc.filter = append(cc.filter, filter)
	}
}

// don't want to raise error not deal hint, so return string
func (cc *Handler[T]) handle(flag string, info T) string {
	for _, filter := range cc.filter {
		keepOn, err, panicStack := filter.call(flag, info)
		if filter.must {
			if len(panicStack) != 0 || err != nil {
				info.addr().Append(CallbackFail)
				info.addr().Append(Failed)
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
	s := CreateSetBySliceFunc(name, func(value string) string { return value })
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
	s := CreateSetBySliceFunc(name, func(value string) string { return value })
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
			panic("CallbackFail is only valid StatusEnum for Before")
		}
	}
	old := c.run
	f := func(info T) (bool, error) {
		for _, match := range status {
			if info.Contain(match) {
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
			if info.Contain(match) {
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
			// this will lead to turn process status to panic
			panicStack = fmt.Sprintf("panic: %v\n\n%s", r, string(debug.Stack()))
		}

	}()

	keepOn, err = c.run(info)
	return
}

// Set method sets the value associated with the given key in own context.
func (t *adjacencyTable) set(num int32, visible Status, key string, value any) {
	t.lock.Lock()
	defer t.lock.Unlock()
	n := &node{
		Index:   num,
		Value:   value,
		Visible: visible.load(),
		Next:    t.nodes[key],
	}
	t.nodes[key] = n
}

func (t *adjacencyTable) find(num int32, key string) (any, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	head, exist := t.nodes[key]
	if !exist {
		return nil, false
	}
	return head.searchByIndex(num)
}

// Get method retrieves the value associated with the given key from the context path.
// The method first checks the priority context, then own context, finally parents context.
// Returns the value associated with the key (if found) and a boolean indicating its presence.
func (t *adjacencyTable) get(visible Status, key string) (any, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if head, exist := t.nodes[key]; exist {
		if value, find := head.search(visible); find {
			return value, find
		}
	}
	return nil, false
}

func (t *adjacencyTable) getAll(visible Status, key string) map[int32]any {
	t.lock.RLock()
	defer t.lock.RUnlock()
	head, exist := t.nodes[key]
	if !exist {
		return nil
	}
	all := make(map[int32]any)
	for head != nil {
		if visible.contain(head.Visible) {
			all[head.Index] = head.Value
		}
		head = head.Next
	}
	return all
}

func (n *node) searchByIndex(index int32) (any, bool) {
	if n.Index == index {
		return n.Value, true
	}
	if n.Next == nil {
		return nil, false
	}
	return n.Next.searchByIndex(index)
}

func (n *node) search(visible Status) (any, bool) {
	if visible.contain(n.Visible) {
		return n.Value, true
	}
	if n.Next == nil {
		return nil, false
	}
	return n.Next.search(visible)
}

func (vc *visibleContext) ContextName() string {
	return vc.roster[vc.index]
}

func (vc *visibleContext) Get(key string) (value any, exist bool) {
	if index, match := vc.priority[key]; match {
		return vc.find(index, key)
	}
	if value, exist = vc.get(vc.visible, key); exist {
		return
	}
	if vc.parent != nil {
		return vc.parent.Get(key)
	}
	return nil, false
}

func (vc *visibleContext) GetAll(key string) map[string]any {
	find := vc.getAll(vc.visible, key)
	result := make(map[string]any, len(find))
	for num, value := range find {
		name, ok := vc.roster[num]
		if !ok {
			continue
		}
		// if set a value more than once, only the last value will be returned
		if _, exist := result[name]; exist {
			continue
		}
		result[name] = value
	}
	if vc.parent != nil {
		for k, v := range vc.parent.GetAll(key) {
			if _, exist := result[k]; !exist {
				result[k] = v
			}
		}
	}
	return result
}

func (vc *visibleContext) GetResult(key string) (value any, exist bool) {
	vc.lock.RLock()
	defer vc.lock.RUnlock()
	head, find := vc.nodes[key]
	if !find {
		return nil, false
	}
	return head.search(resultMark)
}

func (vc *visibleContext) setResult(key string, value any) {
	vc.lock.Lock()
	defer vc.lock.Unlock()
	// set num and path to -1, so Get method will skip result
	head := &node{
		Index:   -1,
		Value:   value,
		Visible: resultMark,
		Next:    vc.nodes[key],
	}
	vc.nodes[key] = head
}

func (vc *visibleContext) Set(key string, value any) {
	vc.set(vc.index, vc.visible, key, value)
}

// Done method waits for the corresponding process to complete.
func (f *Future) Done() {
	f.finish.Wait()
}
