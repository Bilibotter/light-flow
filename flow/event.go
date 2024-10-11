package flow

/*
 *
 *   Copyright (c) 2024 Bilibotter
 *
 *   This software is licensed under the MIT License.
 *   For more details, see the LICENSE file or visit:
 *   https://opensource.org/licenses/MIT
 *
 *   Project: https://github.com/Bilibotter/light-flow
 *
 */

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var dispatcher = &eventDispatcher{init: sync.Once{}, capacity: 64, handlerNum: 0, maxHandler: 16, timeoutSec: 300}

var discardDelay int64 = 60
var waitEventTimeout = 30 * time.Second
var handlerRegistry = newEventRegister()

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
	Handle(stage EventStage, handles ...func(event FlexEvent) (keepOn bool)) HandlerRegister
	Discard(stage EventStage, discards ...func(event FlexEvent) (keepOn bool)) HandlerRegister
	DisableLog(stages ...EventStage) HandlerRegister
	Capacity(capacity int) HandlerRegister
	MaxHandler(num int) HandlerRegister
	EventTimeoutSec(sec int) HandlerRegister
	Clear()
}

type FlexEvent interface {
	basicEventI
	flowInfo
	procInfo
	Get(key string) (any, bool) // Get value from context, only available for step and process.
}

type basicEventI interface {
	errorResult
	ID() string
	Name() string
	EventID() string
	Stage() EventStage
	Level() EventLevel
	Layer() EventLayer
	Timestamp() time.Time
	DetailsMap() map[string]string
	// Details fetches the value of a key from the event's details map.
	// If the key does not exist, an empty string is returned.
	Details(key string) string
}

type errorResult interface {
	error
	Panic() any
	StackTrace() []byte // only panic event has stack trace
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
	error
	proto
	EventStage
	EventLayer
	EventLevel
	eventID   string
	timestamp time.Time
	object    any // panic object
	stack     []byte
	details   map[string]string
}

type handlerRegister struct {
	handlers map[EventStage][]func(FlexEvent) (keepOn bool)
	discards map[EventStage][]func(FlexEvent) (keepOn bool)
	logs     map[EventStage]func(FlexEvent)
	unLog    map[EventStage]bool
}

func EventHandler() HandlerRegister {
	return handlerRegistry
}

func errorEvent[T proto](runtime T, stage EventStage, err error) *flexEvent {
	event := newEvent[T](runtime, stage)
	event.error = err
	event.EventLevel = ErrorLevel
	return event
}

func panicEvent[T proto](runtime T, stage EventStage, object any, stack []byte) *flexEvent {
	event := newEvent[T](runtime, stage)
	event.object = object
	event.stack = stack
	event.EventLevel = PanicLevel
	return event
}

func newEvent[T proto](runtime T, stage EventStage) *flexEvent {
	event := &flexEvent{
		proto:      runtime,
		eventID:    generateId(),
		timestamp:  time.Now(),
		EventStage: stage,
		EventLevel: ErrorLevel,
	}
	switch any(runtime).(type) {
	case *runFlow:
		event.EventLayer = FlowLayer
	case *runProcess:
		event.EventLayer = ProcLayer
	case *runStep:
		event.EventLayer = StepLayer
	}
	return event
}

func newEventRegister() *handlerRegister {
	return &handlerRegister{
		handlers: make(map[EventStage][]func(FlexEvent) (keepOn bool)),
		discards: make(map[EventStage][]func(FlexEvent) (keepOn bool)),
		unLog:    make(map[EventStage]bool),
		logs: map[EventStage]func(event FlexEvent){
			InCallback: commonLog(callbackOrder),
			InSuspend:  commonLog(suspendOrder),
			InRecover:  commonLog(recoverOrder),
			InResource: commonLog(resourceOrder),
			InPersist:  commonLog(persistOrder),
			inEvent:    commonLog(eventOrder),
		},
	}
}

func (ed *eventDispatcher) send(event FlexEvent) {
	ed.init.Do(ed.start)
	select {
	case ed.eventBus <- event:
		if len(ed.eventBus)/4 > int(atomic.LoadInt64(&ed.handlerNum)) {
			ed.addHandler(nil)
		}
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
		}
		ed.RUnlock()
		return nil
	}
	if len(ed.handlers) < int(ed.handlerNum) {
		handler := new(handlerLifecycle)
		handler.eventDispatcher = ed
		ed.Lock()
		ed.handlers = append(ed.handlers, handler)
		ed.Unlock()
		return handler
	}
	ed.RLock()
	defer ed.RUnlock()
	for _, handler := range ed.handlers {
		// reuse discard handler
		if atomic.CompareAndSwapInt64(&handler.status, unattachedH, idleH) {
			atomic.AddInt64(&handler.latest, 1)
			return handler
		}
	}
	return nil
}

func (el *handlerLifecycle) loop(bind FlexEvent) {
	if bind != nil {
		handlerRegistry.handle(bind)
	}
	index := atomic.LoadInt64(&el.latest)
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
			current := atomic.AddInt64(&el.handlerNum, -1)
			if current != 0 {
				return true
			}
			// Need to ensure that at least one handler is preserved
			atomic.StoreInt64(&el.status, runningH)
			atomic.AddInt64(&el.handlerNum, 1)
			return false
		}
	}
	atomic.CompareAndSwapInt64(&el.status, discardH, runningH)
	return false
}

func (h *handlerRegister) Handle(stage EventStage, handlers ...func(event FlexEvent) (keepOn bool)) HandlerRegister {
	if len(handlers) == 0 {
		panic("No handler is given")
	}
	h.handlers[stage] = append(h.handlers[stage], handlers...)
	return h
}

func (h *handlerRegister) Discard(stage EventStage, discards ...func(event FlexEvent) (keepOn bool)) HandlerRegister {
	if len(discards) == 0 {
		panic("No discard is given")
	}
	h.discards[stage] = append(h.discards[stage], discards...)
	return h
}

func (h *handlerRegister) DisableLog(stages ...EventStage) HandlerRegister {
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

func (h *handlerRegister) Clear() {
	h.handlers = map[EventStage][]func(FlexEvent) (keepOn bool){}
	h.discards = map[EventStage][]func(FlexEvent) (keepOn bool){}
	h.unLog = map[EventStage]bool{}
}

func (h *handlerRegister) handle(event FlexEvent) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf(handlePanicLog, event.Stage(), event.Layer(), event.Name(), event.ID(), r, stack())
		}
	}()
	if !h.unLog[event.Stage()] {
		h.logs[event.Stage()](event)
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
			logger.Errorf(discardPanicLog, event.Stage(), event.Layer(), event.Name(), event.ID(), r, stack())
		}
	}()
	for _, handler := range h.discards[event.Stage()] {
		if !handler(event) {
			return
		}
	}
}

func (f *flexEvent) Error() string {
	if f.error != nil {
		return f.error.Error()
	}
	return ""
}

func (f *flexEvent) Panic() any {
	return f.object
}

func (f *flexEvent) StackTrace() []byte {
	return f.stack
}

func (f *flexEvent) EventID() string {
	return f.eventID
}

func (f *flexEvent) Stage() EventStage {
	return f.EventStage
}

func (f *flexEvent) Level() EventLevel {
	return f.EventLevel
}

func (f *flexEvent) Layer() EventLayer {
	return f.EventLayer
}

func (f *flexEvent) Timestamp() time.Time {
	return f.timestamp
}

func (f *flexEvent) DetailsMap() map[string]string {
	return f.details
}

func (f *flexEvent) Details(key string) string {
	return f.details[key]
}

func (f *flexEvent) FlowID() string {
	switch f.proto.(type) {
	case *runFlow:
		return f.proto.(*runFlow).ID()
	case *runProcess:
		return f.proto.(*runProcess).FlowID()
	case *runStep:
		return f.proto.(*runStep).FlowID()
	default:
		return ""
	}
}

func (f *flexEvent) FlowName() string {
	switch f.proto.(type) {
	case *runFlow:
		return f.proto.(*runFlow).Name()
	case *runProcess:
		return f.proto.(*runProcess).FlowName()
	case *runStep:
		return f.proto.(*runStep).FlowName()
	default:
		return ""
	}
}

func (f *flexEvent) ProcessID() string {
	switch f.proto.(type) {
	case *runProcess:
		return f.proto.(*runProcess).Name()
	case *runStep:
		return f.proto.(*runStep).ProcessName()
	default:
		return ""
	}
}

func (f *flexEvent) ProcessName() string {
	switch f.proto.(type) {
	case *runProcess:
		return f.proto.(*runProcess).ID()
	case *runStep:
		return f.proto.(*runStep).ProcessID()
	default:
		return ""
	}
}

func (f *flexEvent) Get(key string) (any, bool) {
	switch f.proto.(type) {
	case *runProcess:
		return f.proto.(*runProcess).Get(key)
	case *runStep:
		return f.proto.(*runStep).Get(key)
	default:
		return nil, false
	}
}

func (f *flexEvent) write(key, value string) {
	if f.details == nil {
		f.details = make(map[string]string, 1)
	}
	f.details[key] = value
}
