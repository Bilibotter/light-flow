package light_flow

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var testStage = InCallback

var letGo int64

func waitCurrent(i int) {
	if atomic.LoadInt64(&current) == int64(i) {
		return
	}
	start := time.Now()
	duration := 600 * time.Millisecond
	for {
		if atomic.LoadInt64(&current) == int64(i) {
			return
		}
		if atomic.LoadInt64(&current) == int64(i) {
			return
		}
		if time.Since(start) > duration {
			panic(fmt.Sprintf("current is not %d, current is %d\n", i, atomic.LoadInt64(&current)))
		}
	}
}

func resetEventEnv() {
	dispatcher.exit()
	resetCurrent()
	resetRegistry()
	atomic.StoreInt64(&discardDelay, 60)
	waitEventTimeout = 30 * time.Second
	dispatcher = new(eventDispatcher)
	dispatcher.init = sync.Once{}
	dispatcher.capacity = 64
	dispatcher.handlerNum = 0
	dispatcher.maxHandler = 16
	dispatcher.timeoutSec = 300
}

func resetRegistry() {
	handlerRegistry = handlerRegister{
		handlers: make(map[eventStage][]func(FlexEvent) (keepOn bool)),
		discards: make(map[eventStage][]func(FlexEvent) (keepOn bool)),
		unLog:    make(map[eventStage]bool),
	}
}

type eventImpl struct {
	name string
}

func (e *eventImpl) Error() error {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) StackTrace() []byte {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) EventID() string {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) EventName() string {
	return e.name
}

func (e *eventImpl) Stage() eventStage {
	return testStage
}

func (e *eventImpl) Severity() eventSeverity {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Scope() eventScope {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Timestamp() time.Time {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Extra(index extraIndex) any {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Flow() FlowInfo {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Proc() ProcInfo {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Step() StepInfo {
	//TODO implement me
	panic("implement me")
}

func sleepEvent(t *testing.T) func(event FlexEvent) bool {
	return func(event FlexEvent) bool {
		t.Logf("Event[ %s ] start", event.EventName())
		time.Sleep(10 * time.Millisecond)
		t.Logf("Event[ %s ] end", event.EventName())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func untilLetGo(t *testing.T) func(event FlexEvent) bool {
	return func(event FlexEvent) bool {
		t.Logf("start Event[ %s ]", event.EventName())
		atomic.AddInt64(&current, 1)
		now := time.Now()
		for tmp := atomic.LoadInt64(&letGo); tmp == 0; tmp = atomic.LoadInt64(&letGo) {
			if time.Since(now) > 600*time.Millisecond {
				t.Errorf("Event[ %s ] wait letGo timeout", event.EventName())
				return true
			}
		}
		t.Logf("end Event[ %s ]", event.EventName())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func dropHint(t *testing.T) func(event FlexEvent) bool {
	return func(event FlexEvent) bool {
		t.Logf("Event[ %s ] discard", event.EventName())
		return true
	}
}

func noDelay(_ FlexEvent) bool {
	atomic.AddInt64(&current, 1)
	return true
}

func TestSend(t *testing.T) {
	defer resetEventEnv()
	HandlerRegistry().Handle(testStage, sleepEvent(t))
	dispatcher.send(&eventImpl{"TestSend"})
	waitCurrent(1)
}

func TestSingleHandler(t *testing.T) {
	defer resetEventEnv()
	atomic.StoreInt64(&letGo, 0)
	HandlerRegistry().Handle(testStage, untilLetGo(t)).MaxHandler(1).Capacity(64).Discard(testStage, dropHint(t))
	for i := 0; i < 32; i++ {
		go dispatcher.send(&eventImpl{"TestSingleHandler" + strconv.Itoa(i)})
	}
	waitCurrent(1)
	if atomic.LoadInt64(&dispatcher.handlerNum) != 1 {
		t.Errorf("handler num should be 1, but got %d", atomic.LoadInt64(&dispatcher.handlerNum))
	}
	atomic.StoreInt64(&letGo, 1)
	waitCurrent(64)
	if atomic.LoadInt64(&dispatcher.handlerNum) != 1 {
		t.Errorf("handler num should be 1, but got %d", atomic.LoadInt64(&dispatcher.handlerNum))
	}
	return
}

func TestHandlerExpired(t *testing.T) {
	defer resetEventEnv()
	atomic.StoreInt64(&letGo, 0)
	HandlerRegistry().Handle(testStage, untilLetGo(t)).Discard(testStage, dropHint(t)).
		MaxHandler(12).Capacity(8).EventTimeoutSec(-1)
	atomic.StoreInt64(&discardDelay, 3600)
	for i := 0; i < 20; i++ {
		go dispatcher.send(&eventImpl{"TestHandlerExpired" + strconv.Itoa(i)})
	}
	waitCurrent(12)
	for i := 0; i < 12; i++ {
		go dispatcher.send(&eventImpl{"TestHandlerExpired" + strconv.Itoa(i+32)})
	}
	waitCurrent(24)
	if atomic.LoadInt64(&dispatcher.handlerNum) != 12 {
		t.Errorf("handler num should be 12, but got %d", atomic.LoadInt64(&dispatcher.handlerNum))
	}
	atomic.StoreInt64(&letGo, 1)
	waitCurrent(64)
}

func TestHandlerAdd(t *testing.T) {
	defer resetEventEnv()
	atomic.StoreInt64(&letGo, 0)
	HandlerRegistry().Handle(testStage, untilLetGo(t)).MaxHandler(64).Capacity(8).Discard(testStage, dropHint(t))
	for i := 0; i < 32; i++ {
		go dispatcher.send(&eventImpl{"TestHandlerAdd" + strconv.Itoa(i)})
	}
	waitCurrent(24)
	if atomic.LoadInt64(&dispatcher.handlerNum) != 24 {
		t.Errorf("handler num should be 24, but got %d, capacity=%d, length=%d", atomic.LoadInt64(&dispatcher.handlerNum), len(dispatcher.eventBus), len(dispatcher.handlers))
	}
	atomic.StoreInt64(&letGo, 1)
	waitCurrent(64)
	t.Logf("Handler num=%d", atomic.LoadInt64(&dispatcher.handlerNum))
	return
}

func TestDiscardHandlerReuse(t *testing.T) {
	defer resetEventEnv()
	atomic.StoreInt64(&letGo, 0)
	HandlerRegistry().Handle(testStage, untilLetGo(t)).Discard(testStage, dropHint(t)).
		MaxHandler(64).Capacity(8)
	atomic.StoreInt64(&discardDelay, 0)
	waitEventTimeout = 1 * time.Millisecond
	for i := 0; i < 32; i++ {
		go dispatcher.send(&eventImpl{"TestDiscardHandlerReuse" + strconv.Itoa(i)})
	}
	waitCurrent(24)
	if atomic.LoadInt64(&dispatcher.handlerNum) != 24 {
		t.Errorf("handler num should be 24, but got %d, capacity=%d, length=%d", atomic.LoadInt64(&dispatcher.handlerNum), len(dispatcher.eventBus), len(dispatcher.handlers))
	}
	atomic.StoreInt64(&letGo, 1)
	waitCurrent(64)
	t.Logf("Handler num=%d", atomic.LoadInt64(&dispatcher.handlerNum))
	start := time.Now()
	pass := false
	for time.Since(start) < 1000*time.Millisecond {
		if atomic.LoadInt64(&dispatcher.handlerNum) == 0 {
			for _, info := range dispatcher.handlers {
				if atomic.LoadInt64(&info.status) != unattachedH {
					t.Errorf("handler status should be unattachedH, but got %d", atomic.LoadInt64(&info.status))
				}
			}
			t.Logf("Discard all event handlers")
			pass = true
			break
		}
	}
	if !pass {
		t.Errorf("handler num should be 0, but got %d, capacity=%d, length=%d", atomic.LoadInt64(&dispatcher.handlerNum), len(dispatcher.eventBus), len(dispatcher.handlers))
		return
	}

	waitEventTimeout = 1 * time.Hour
	resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	atomic.StoreInt64(&discardDelay, 3600)
	for i := 0; i < 32; i++ {
		go dispatcher.send(&eventImpl{"TestHandlerDiscard" + strconv.Itoa(i)})
	}
	waitCurrent(24)
	if atomic.LoadInt64(&dispatcher.handlerNum) != 24 {
		t.Errorf("handler num should be 24, but got %d, capacity=%d, length=%d", atomic.LoadInt64(&dispatcher.handlerNum), len(dispatcher.eventBus), len(dispatcher.handlers))
	}
	atomic.StoreInt64(&letGo, 1)
	waitCurrent(64)
	if len(dispatcher.handlers) != 24 {
		t.Errorf("handler num should be 24, but got %d, capacity=%d, length=%d", atomic.LoadInt64(&dispatcher.handlerNum), len(dispatcher.eventBus), len(dispatcher.handlers))
	}
	t.Logf("Handler info num=%d", len(dispatcher.handlers))
	return
}

func TestHandlerDiscard(t *testing.T) {
	defer resetEventEnv()
	atomic.StoreInt64(&letGo, 0)
	HandlerRegistry().Handle(testStage, untilLetGo(t)).Discard(testStage, dropHint(t)).
		MaxHandler(64).Capacity(8)
	atomic.StoreInt64(&discardDelay, 0)
	waitEventTimeout = 1 * time.Millisecond
	for i := 0; i < 32; i++ {
		go dispatcher.send(&eventImpl{"TestHandlerDiscard" + strconv.Itoa(i)})
	}
	waitCurrent(24)
	if atomic.LoadInt64(&dispatcher.handlerNum) != 24 {
		t.Errorf("handler num should be 24, but got %d, capacity=%d, length=%d", atomic.LoadInt64(&dispatcher.handlerNum), len(dispatcher.eventBus), len(dispatcher.handlers))
	}
	atomic.StoreInt64(&letGo, 1)
	waitCurrent(64)
	t.Logf("Handler num=%d", atomic.LoadInt64(&dispatcher.handlerNum))
	start := time.Now()
	for time.Since(start) < 1000*time.Millisecond {
		if atomic.LoadInt64(&dispatcher.handlerNum) == 0 {
			for _, info := range dispatcher.handlers {
				if atomic.LoadInt64(&info.status) != unattachedH {
					t.Errorf("handler status should be unattachedH, but got %d", atomic.LoadInt64(&info.status))
				}
			}
			t.Logf("Discard all event handlers")
			return
		}
	}
	t.Errorf("handler num should be 0, but got %d, capacity=%d, length=%d", atomic.LoadInt64(&dispatcher.handlerNum), len(dispatcher.eventBus), len(dispatcher.handlers))
	return
}

func TestMultipleEvent(t *testing.T) {
	defer resetEventEnv()
	HandlerRegistry().
		Handle(testStage, noDelay).Discard(testStage, noDelay).
		MaxHandler(32).Capacity(128)
	for i := 0; i < 4000; i++ {
		go dispatcher.send(&eventImpl{})
	}
	waitCurrent(4000)
}
