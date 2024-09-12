package light_flow

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var dispatcher = &eventDispatcher{init: sync.Once{}, capacity: 64, handlerNum: 0, maxHandler: 16, timeoutSec: 300}

var discardDelay int64 = 60
var waitEventTimeout = 30 * time.Second

var handlerRegistry = handlerRegister{
	handlers: make(map[eventStage][]func(FlexEvent) (keepOn bool)),
	discards: make(map[eventStage][]func(FlexEvent) (keepOn bool)),
	unLog:    make(map[eventStage]bool),
}

var (
	conditionName = map[eventType]string{}
	callbackName  = map[eventType]string{}
	persistName   = map[eventType]string{}
	suspendName   = map[eventType]string{}
	recoverName   = map[eventType]string{}
	eventNames    = map[eventStage]map[eventType]string{
		InCondition: conditionName,
		InCallback:  callbackName,
		InPersist:   persistName,
		InSuspend:   suspendName,
		InRecover:   recoverName,
	}
)

/*************************************************************
 * Event Bus Status
 *************************************************************/

const (
	busIdle int64 = iota
	busExpand
)

/*************************************************************
 * Handler Status
 *************************************************************/

const (
	idleH int64 = iota
	runningH
	discardH
	expiredH
	unattachedH
)

type HandlerRegister interface {
	Handle(stage eventStage, handles ...func(event FlexEvent) (keepOn bool)) HandlerRegister
	Discard(stage eventStage, discards ...func(event FlexEvent) (keepOn bool)) HandlerRegister
	DisableLog(stages ...eventStage) HandlerRegister
	Capacity(capacity int) HandlerRegister
	MaxHandler(num int) HandlerRegister
	EventTimeoutSec(sec int) HandlerRegister
}

type FlexEvent interface {
	basicEventI
	Flow() FlowInfo // never return nil
	Proc() ProcInfo // if event is not a proc event or step event, return nil
	Step() StepInfo // if event is not a step event, return nil
}

type basicEventI interface {
	exception
	EventID() string
	EventName() string
	Stage() eventStage
	Severity() eventSeverity
	Scope() eventScope
	Timestamp() time.Time
	Extra(index extraIndex) any
}

type exception interface {
	Error() error
	StackTrace() []byte // only panic event has stack trace
}

type FlowInfo interface {
	flowRuntime
}

type ProcInfo interface {
	procRuntime
	Get(key string) (value any, exist bool)
}

type StepInfo interface {
	stepRuntime
	Get(key string) (value any, exist bool)
}

type eventDispatcher struct {
	sync.RWMutex
	init       sync.Once
	timeoutSec int64
	handlers   []*handlerLifecycle
	handlerNum int64
	maxHandler int64
	busStatus  int64
	capacity   int
	discarding int64
	eventBus   chan FlexEvent
	quit       chan empty
}

type handlerLifecycle struct {
	*eventDispatcher
	status     int64
	latest     int64
	expiration int64
}

type flexEvent struct {
	eventSeverity
	eventStage
	eventID   string
	timestamp time.Time
	err       error
	stack     []byte
	extra     []any
}

type handlerRegister struct {
	handlers map[eventStage][]func(FlexEvent) (keepOn bool)
	discards map[eventStage][]func(FlexEvent) (keepOn bool)
	unLog    map[eventStage]bool
}

type eventHandler struct {
	handle  func(event FlexEvent) (keepOn bool)
	target  *set[string]
	exclude *set[string]
}

func HandlerRegistry() HandlerRegister {
	return &handlerRegistry
}

func (ed *eventDispatcher) send(event FlexEvent) {
	ed.init.Do(ed.start)
	select {
	case ed.eventBus <- event:
		return
	default:
	}
	if ed.addHandler(event) {
		return
	}
	go handlerRegistry.discard(event)
}

func (ed *eventDispatcher) start() {
	ed.eventBus = make(chan FlexEvent, ed.capacity)
	ed.quit = make(chan empty, 1)
	ed.boostHandler()
}

func (ed *eventDispatcher) boostHandler() {
	boost := handlerLifecycle{
		eventDispatcher: ed,
		latest:          1,
		status:          runningH,
		expiration:      time.Now().Unix() + ed.timeoutSec,
	}
	go boost.loop(nil)
	atomic.AddInt64(&ed.handlerNum, 1)
	ed.Lock()
	defer ed.Unlock()
	ed.handlers = append(ed.handlers, &boost)
}

func (ed *eventDispatcher) exit() {
	ed.quit <- empty{}
}

func (ed *eventDispatcher) addHandler(event FlexEvent) bool {
	handler := ed.findHandler()
	if handler != nil {
		atomic.StoreInt64(&handler.expiration, time.Now().Unix()+ed.timeoutSec)
		atomic.StoreInt64(&handler.status, runningH)
		go handler.loop(event)
		return true
	}
	return false
}

func (ed *eventDispatcher) findHandler() *handlerLifecycle {
	if atomic.AddInt64(&ed.handlerNum, 1) > ed.maxHandler {
		atomic.AddInt64(&ed.handlerNum, -1)
		now := time.Now().Unix()
		ed.RLock()
		// Replace the timeout handler with a new one
		start := rand.Intn(len(ed.handlers))
		for i := 0; i < len(ed.handlers); i++ {
			handler := ed.handlers[(start+i)%len(ed.handlers)]
			if atomic.LoadInt64(&handler.expiration) <= now && atomic.CompareAndSwapInt64(&handler.status, runningH, expiredH) {
				atomic.AddInt64(&handler.latest, 1)
				ed.RUnlock()
				return handler
			}
			println("here")
		}
		ed.RUnlock()
		return nil
	}
	if len(ed.handlers) >= int(ed.handlerNum) {
		ed.RLock()
		for _, handler := range ed.handlers {
			// reuse discard handler
			if atomic.CompareAndSwapInt64(&handler.status, unattachedH, idleH) {
				atomic.AddInt64(&handler.latest, 1)
				ed.RUnlock()
				return handler
			}
		}
		ed.RUnlock()
	}
	handler := new(handlerLifecycle)
	handler.eventDispatcher = ed
	ed.Lock()
	defer ed.Unlock()
	ed.handlers = append(ed.handlers, handler)
	return handler
}

func (el *handlerLifecycle) loop(bind FlexEvent) {
	if bind != nil {
		handlerRegistry.handle(bind)
	}
	index := atomic.LoadInt64(&el.latest)
	defer func() {
		if r := recover(); r == nil {
			return
		}
		el.Lock()
		defer el.Unlock()
		if index != atomic.LoadInt64(&el.latest) {
			return
		}
		atomic.StoreInt64(&el.status, unattachedH)
		atomic.AddInt64(&el.handlerNum, -1)
	}()
	for {
		select {
		case event := <-el.eventBus:
			handlerRegistry.handle(event)
		case <-el.quit:
			el.quit <- empty{}
			return
		default:
			if el.tryUnattached() {
				return
			}
		}
		// handler timeout and is replaced by the new
		if index != atomic.LoadInt64(&el.latest) {
			return
		}
		atomic.StoreInt64(&el.expiration, time.Now().Unix()+el.timeoutSec)
	}
}

func (el *handlerLifecycle) tryUnattached() bool {
	// Prevent from marking as expired
	if !atomic.CompareAndSwapInt64(&el.status, runningH, discardH) {
		return false
	}
	timer := time.NewTimer(waitEventTimeout)
	select {
	case event := <-el.eventBus:
		handlerRegistry.handle(event)
		atomic.CompareAndSwapInt64(&el.status, discardH, runningH)
		return false
	case <-el.quit:
		el.quit <- empty{}
		return true
	case <-timer.C:
	}
	discarding := atomic.LoadInt64(&el.discarding)
	// Gradually remove the excess handlers.
	if time.Now().Unix()-discarding >= discardDelay {
		if atomic.CompareAndSwapInt64(&el.discarding, discarding, time.Now().Unix()) {
			atomic.StoreInt64(&el.status, unattachedH)
			atomic.AddInt64(&el.handlerNum, -1)
			return true
		}
	}
	atomic.CompareAndSwapInt64(&el.status, discardH, runningH)
	return false
}

func (h *handlerRegister) Handle(stage eventStage, handlers ...func(event FlexEvent) (keepOn bool)) HandlerRegister {
	h.handlers[stage] = append(h.handlers[stage], handlers...)
	return h
}

func (h *handlerRegister) Discard(stage eventStage, discards ...func(event FlexEvent) (keepOn bool)) HandlerRegister {
	h.discards[stage] = append(h.discards[stage], discards...)
	return h
}

func (h *handlerRegister) DisableLog(stages ...eventStage) HandlerRegister {
	for _, stage := range stages {
		h.unLog[stage] = true
	}
	return h
}

func (h *handlerRegister) Capacity(capacity int) HandlerRegister {
	if capacity <= 0 {
		panic("The capacity must be greater than 0")
	}
	dispatcher.capacity = capacity
	return h
}

func (h *handlerRegister) MaxHandler(num int) HandlerRegister {
	dispatcher.maxHandler = int64(num)
	return h
}

func (h *handlerRegister) EventTimeoutSec(sec int) HandlerRegister {
	dispatcher.timeoutSec = int64(sec)
	return h
}

func (h *handlerRegister) handle(event FlexEvent) {
	defer func() {
		if r := recover(); r != nil {
			//todo log panic
		}
	}()
	if !h.unLog[event.Stage()] {
		// log it
	}
	for _, handler := range h.handlers[event.Stage()] {
		if !handler(event) {
			return
		}
	}
}

func (h *handlerRegister) discard(event FlexEvent) {
	defer func() {
		if r := recover(); r != nil {
			//todo log panic
		}
	}()
	if !h.unLog[event.Stage()] {
		// log it
	}
	for _, handler := range h.discards[event.Stage()] {
		if !handler(event) {
			return
		}
	}
}
