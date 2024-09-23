package flow

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
	FlowPersist().OnInsert(func(_ WorkFlow) error {
		return nil
	})
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
	handlerRegistry = newEventRegister()
}

type eventImpl struct {
	name string
}

func (e *eventImpl) DetailsMap() map[string]string {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Details(key string) string {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) FlowID() string {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) FlowName() string {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) ProcessID() string {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) ProcessName() string {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Get(key string) (any, bool) {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Panic() any {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) EventID() string {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Level() EventLevel {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Fetch(key string) string {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Error() string {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) StackTrace() []byte {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) ID() string {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Name() string {
	return e.name
}

func (e *eventImpl) Stage() EventStage {
	return testStage
}

func (e *eventImpl) Severity() EventLevel {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Layer() EventLayer {
	//TODO implement me
	panic("implement me")
}

func (e *eventImpl) Timestamp() time.Time {
	//TODO implement me
	panic("implement me")
}

func sleepEvent(t *testing.T) func(event FlexEvent) bool {
	return func(event FlexEvent) bool {
		t.Logf("Event[ %s ] start", event.Name())
		time.Sleep(10 * time.Millisecond)
		t.Logf("Event[ %s ] end", event.Name())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func untilLetGo(t *testing.T) func(event FlexEvent) bool {
	return func(event FlexEvent) bool {
		t.Logf("start Event[ %s ]", event.Name())
		atomic.AddInt64(&current, 1)
		now := time.Now()
		for tmp := atomic.LoadInt64(&letGo); tmp == 0; tmp = atomic.LoadInt64(&letGo) {
			if time.Since(now) > 600*time.Millisecond {
				t.Errorf("Event[ %s ] wait letGo timeout", event.Name())
				return true
			}
		}
		t.Logf("end Event[ %s ]", event.Name())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func dropHint(t *testing.T) func(event FlexEvent) bool {
	return func(event FlexEvent) bool {
		t.Logf("Event[ %s ] discard", event.Name())
		return true
	}
}

func noDelay(_ FlexEvent) bool {
	atomic.AddInt64(&current, 1)
	return true
}

func TestSend(t *testing.T) {
	defer resetEventEnv()
	EventHandler().Handle(testStage, sleepEvent(t)).DisableLog(InCallback)
	dispatcher.send(&eventImpl{"TestSend"})
	waitCurrent(1)
}

func TestSingleHandler(t *testing.T) {
	defer resetEventEnv()
	atomic.StoreInt64(&letGo, 0)
	EventHandler().Handle(testStage, untilLetGo(t)).MaxHandler(1).Capacity(64).Discard(testStage, dropHint(t)).DisableLog(InCallback)
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
	EventHandler().Handle(testStage, untilLetGo(t)).Discard(testStage, dropHint(t)).
		MaxHandler(12).Capacity(8).EventTimeoutSec(-1).DisableLog(InCallback)
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
	EventHandler().Handle(testStage, untilLetGo(t)).MaxHandler(64).Capacity(8).Discard(testStage, dropHint(t)).DisableLog(InCallback)
	for i := 0; i < 32; i++ {
		go dispatcher.send(&eventImpl{"TestHandlerAdd" + strconv.Itoa(i)})
	}
	waitCurrent(24)
	if atomic.LoadInt64(&dispatcher.handlerNum) != 24 {
		if int(atomic.LoadInt64(&dispatcher.handlerNum))+len(dispatcher.eventBus) != 32 {
			t.Logf("Handler num=%d, capacity=%d, length=%d", atomic.LoadInt64(&dispatcher.handlerNum), len(dispatcher.eventBus), len(dispatcher.handlers))
		}
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
	EventHandler().Handle(testStage, untilLetGo(t)).Discard(testStage, dropHint(t)).
		MaxHandler(64).Capacity(8).DisableLog(InCallback)
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
			var goOn bool
			for _, info := range dispatcher.handlers {
				if atomic.LoadInt64(&info.status) != unattachedH && atomic.LoadInt64(&info.status) != runningH {
					goOn = true
					break
				}
			}
			if goOn {
				continue
			}
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
	EventHandler().Handle(testStage, untilLetGo(t)).Discard(testStage, dropHint(t)).
		MaxHandler(64).Capacity(8).DisableLog(InCallback)
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
		if atomic.LoadInt64(&dispatcher.handlerNum) == 1 {
			var goOn bool
			for _, info := range dispatcher.handlers {
				if atomic.LoadInt64(&info.status) != unattachedH && atomic.LoadInt64(&info.status) != runningH {
					goOn = true
					break
				}
			}
			if goOn {
				continue
			}
			t.Logf("Discard all event handlers")
			return
		}
	}
	for _, info := range dispatcher.handlers {
		if atomic.LoadInt64(&info.status) != unattachedH && atomic.LoadInt64(&info.status) != runningH {
			t.Errorf("handler status should be unattachedH, but got %d", atomic.LoadInt64(&info.status))
		}
	}
	t.Errorf("handler num should be 0, but got %d, capacity=%d, length=%d", atomic.LoadInt64(&dispatcher.handlerNum), len(dispatcher.eventBus), len(dispatcher.handlers))
	return
}

func TestMultipleEvent(t *testing.T) {
	defer resetEventEnv()
	EventHandler().
		Handle(testStage, noDelay).Discard(testStage, noDelay).
		MaxHandler(32).Capacity(128).DisableLog(InCallback)
	for i := 0; i < 4000; i++ {
		go dispatcher.send(&eventImpl{})
	}
	waitCurrent(4000)
}

func TestEventHandlerDiscardPanic(t *testing.T) {
	defer resetEventEnv()
	t.Logf("Dispather capacity=%d, maxHandler=%d", dispatcher.capacity, dispatcher.maxHandler)
	t.Logf("Dsipatcher handler num=%d, eventBus length=%d", dispatcher.handlerNum, len(dispatcher.eventBus))
	FlowPersist().OnInsert(func(flow WorkFlow) error {
		return fmt.Errorf("persist error")
	})
	canOpen := int64(0)
	EventHandler().
		Handle(InPersist, func(event FlexEvent) (keepOn bool) {
			t.Logf("receive event: %s", event.Name())
			for now := time.Now(); time.Since(now) < time.Second; {
				if atomic.LoadInt64(&canOpen) == 1 {
					atomic.AddInt64(&current, 1)
					return true
				}
			}
			t.Errorf("handler wait timeout")
			return true
		}).
		Discard(InPersist, func(event FlexEvent) (keepOn bool) {
			t.Logf("discard event: %s", event.Name())
			atomic.AddInt64(&current, 1)
			panic("handle panic")
		}).
		MaxHandler(1).Capacity(1)
	wf := RegisterFlow("TestEventHandlerDiscardPanic")
	proc := wf.Process("TestEventHandlerDiscardPanic")
	proc.CustomStep(func(ctx Step) (any, error) {
		atomic.AddInt64(&current, 1)
		return nil, nil
	}, "1")
	for i := 0; i < 4; i++ {
		t.Logf("TestEventHandlerDiscardPanic round %d", i)
		ff := DoneFlow("TestEventHandlerDiscardPanic", nil)
		if !ff.Success() {
			t.Errorf("TestEventHandlerDiscardPanic failed")
		}
		t.Logf("TestEventHandler DiscardPanic round %d done", i)
		t.Logf("Dispather capacity=%d, maxHandler=%d", dispatcher.capacity, dispatcher.maxHandler)
		t.Logf("Dsipatcher handler num=%d, eventBus length=%d", dispatcher.handlerNum, len(dispatcher.eventBus))
	}
	waitCurrent(6)
	atomic.AddInt64(&canOpen, 1)
	waitCurrent(8)
}

func TestEventHandlerPanic(t *testing.T) {
	defer resetEventEnv()
	FlowPersist().OnInsert(func(flow WorkFlow) error {
		return fmt.Errorf("persist error")
	})
	EventHandler().Handle(InPersist, func(event FlexEvent) (keepOn bool) {
		atomic.AddInt64(&current, 1)
		panic("handle panic")
	}).MaxHandler(1).Capacity(1)
	wf := RegisterFlow("TestEventHandlerPanic")
	proc := wf.Process("TestEventHandlerPanic")
	proc.CustomStep(func(ctx Step) (any, error) {
		atomic.AddInt64(&current, 1)
		return nil, nil
	}, "1")
	ff := DoneFlow("TestEventHandlerPanic", nil)
	waitCurrent(2)
	CheckResult(t, 2, Success)(any(ff).(WorkFlow))

	// test if handler will block when handle panic
	for i := 0; i < 4; i++ {
		resetCurrent()
		ff = DoneFlow("TestEventHandlerPanic", nil)
		waitCurrent(2)
		CheckResult(t, 2, Success)(any(ff).(WorkFlow))
	}
}
