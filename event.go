package light_flow

import (
	"sync"
	"sync/atomic"
	"time"
)

var eventBus = eventDispatcher{init: sync.Once{}, maxBus: 64, maxHandler: 16, timeoutSec: 300}

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
 * event bus status
 *************************************************************/

const (
	busIdle int64 = iota
	busExpand
)

/*************************************************************
 * handler status
 *************************************************************/

const (
	idleH int64 = iota
	runningH
	discardH
	expiredH
	unattachedH
)

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

type flexEvent struct {
	eventSeverity
	eventStage
	eventID   string
	timestamp time.Time
	err       error
	stack     []byte
	extra     []any
}

type eventDispatcher struct {
	sync.WaitGroup
	sync.RWMutex
	init       sync.Once
	timeoutSec int64
	handlers   []*eventHandler
	handlerNum int64
	maxHandler int64
	busStatus  int64
	maxBus     int
	using      int64
	discarding int64
	eventBus0  chan FlexEvent
	eventBus1  chan FlexEvent
}

type eventHandler struct {
	status     int64
	latest     int64
	expiration int64
}

func (es *eventDispatcher) send(event FlexEvent) {
	es.init.Do(es.start)
	master := es.eventBus0
	if atomic.LoadInt64(&es.using) == 1 {
		master = es.eventBus1
	}
	select {
	case master <- event:
		return
	default:
	}
	handlers := atomic.LoadInt64(&es.handlerNum)
	if handlers <= int64(len(master)) && handlers < es.maxHandler {
		go es.addHandler(event)
		return
	}
	go es.expand(event)
}

func (es *eventDispatcher) start() {
	es.eventBus0 = make(chan FlexEvent, 8)
	es.eventBus1 = make(chan FlexEvent, 2)
}

func (es *eventDispatcher) addHandler(event FlexEvent) {
	handler := es.findHandler()
	if handler != nil {
		atomic.AddInt64(&handler.latest, 1)
		atomic.StoreInt64(&handler.expiration, time.Now().Unix()+es.timeoutSec)
		atomic.StoreInt64(&handler.status, runningH)
		go handler.loop(event)
		return
	}
	es.expand(event)
}

func (es *eventDispatcher) findHandler() *eventHandler {
	if atomic.AddInt64(&es.handlerNum, 1) > es.maxHandler {
		atomic.AddInt64(&es.handlerNum, -1)
		now := time.Now().Unix()
		es.RLock()
		defer es.RUnlock()
		// Replace the timeout handler with a new one
		for _, handler := range es.handlers {
			if atomic.LoadInt64(&handler.expiration) > now && atomic.CompareAndSwapInt64(&handler.status, runningH, expiredH) {
				return handler
			}
		}
		return nil
	}
	length := int64(len(es.handlers))
	if length <= (es.handlerNum + es.handlerNum>>2) {
		es.RLock()
		defer es.RUnlock()
		for _, handler := range es.handlers {
			// reuse discard handler
			if atomic.CompareAndSwapInt64(&handler.status, unattachedH, idleH) {
				return handler
			}
		}
	}
	handler := new(eventHandler)
	es.Lock()
	defer es.Unlock()
	es.handlers = append(es.handlers, handler)
	return handler
}

func (eh *eventHandler) loop(bind FlexEvent) {
	// todo deal with bind event
	index := atomic.LoadInt64(&eh.latest)
	var event FlexEvent
	for {
		select {
		case event = <-eventBus.eventBus0:
			//todo handler event
		case event = <-eventBus.eventBus1:
			//todo handler event
		default:
			if eh.tryDiscard() {
				return
			}
		}
		// handler timeout and is replaced by the new
		if index != atomic.LoadInt64(&eh.latest) {
			return
		}
		atomic.StoreInt64(&eh.expiration, time.Now().Unix()+eventBus.timeoutSec)
	}
}

func (eh *eventHandler) tryDiscard() bool {
	if !atomic.CompareAndSwapInt64(&eh.status, runningH, discardH) {
		return false
	}
	var event FlexEvent
	timer := time.NewTimer(30 * time.Second)
	select {
	case event = <-eventBus.eventBus0:
		//todo handler event
		return false
	case event = <-eventBus.eventBus1:
		//todo handler event
		return false
	case <-timer.C:
	}
	discarding := atomic.LoadInt64(&eventBus.discarding)
	// Gradually remove the excess handlers.
	if time.Now().Unix()-discarding > 30 {
		return atomic.CompareAndSwapInt64(&eventBus.discarding, discarding, time.Now().Unix())
	}
	return false
}

func (es *eventDispatcher) expand(bind FlexEvent) {
	if es.tryDiscardEvent(bind) {
		return
	}
	if atomic.LoadInt64(&es.busStatus) == busExpand || !atomic.CompareAndSwapInt64(&es.busStatus, busIdle, busExpand) {
		es.Wait()
		es.send(bind)
		return
	}

	capacity := es.handlerNum + es.handlerNum/2
	slave := es.eventBus1
	if es.using == 1 {
		slave = es.eventBus0
	}
	if len(slave) != 0 {
		capacity = es.handlerNum + es.handlerNum
	}

	if len(slave) >= int(capacity) {
		atomic.StoreInt64(&es.using, ^es.using)
		atomic.StoreInt64(&es.busStatus, busIdle)
		return
	}

	newMaster := make(chan FlexEvent, capacity)
	keepOn := true
	for {
		select {
		case e := <-slave:
			newMaster <- e
		default:
			keepOn = false
		}
		if !keepOn {
			break
		}
	}

	atomic.StoreInt64(&es.using, ^es.using)
	atomic.StoreInt64(&es.busStatus, busIdle)

	// DCL
	for {
		select {
		case e := <-slave:
			newMaster <- e
		default:
			keepOn = false
		}
		if !keepOn {
			break
		}
	}
}

func (es *eventDispatcher) tryDiscardEvent(event FlexEvent) bool {
	if es.maxBus <= 0 {
		return false
	}
	if len(es.eventBus0) >= es.maxBus || len(es.eventBus1) >= es.maxBus {
		//todo discard event
		return true
	}
	return false
}
