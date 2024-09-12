package test

import (
	flow "github.com/Bilibotter/light-flow"
	"sync/atomic"
	"testing"
)

func restEventEnv() {
	flow.HandlerRegistry().Clear()
}

func eventCheck(t *testing.T) func(event flow.FlexEvent) {
	return func(event flow.FlexEvent) {
		if event.EventID() == "" {
			t.Errorf("Expected event ID, got empty string")
		}
		if event.Name() == "" {
			t.Errorf("Expected event name, got empty string")
		}
		if event.ID() == "" {
			t.Errorf("Expected event ID, got empty string")
		}
		if event.Timestamp().IsZero() {
			t.Errorf("Expected timestamp, got zero value")
		}
		if event.Extra() == nil {
			t.Errorf("Expected extra, got nil")
		} else {
			t.Logf("Extra: %v\n", event.Extra())
		}
		if event.Scope() == flow.FlowScp {
			if event.Flow() == nil {
				t.Errorf("Expected flow info, got nil")
			}
		}
		if event.Scope() == flow.ProcScp {
			if event.Flow() == nil {
				t.Errorf("Expected flow info, got nil")
			}
			if event.Proc() == nil {
				t.Errorf("Expected proc info, got nil")
			}
		}
		if event.Scope() == flow.StepScp {
			if event.Flow() == nil {
				t.Errorf("Expected flow info, got nil")
			}
			if event.Proc() == nil {
				t.Errorf("Expected proc info, got nil")
			}
			if event.Step() == nil {
				t.Errorf("Expected step info, got nil")
			}
		}
	}
}

func callbackErrHandler(t *testing.T) func(event flow.FlexEvent) (keepOn bool) {
	f := eventCheck(t)
	return func(event flow.FlexEvent) (keepOn bool) {
		t.Logf("Event[ %s ] start", event.Name())
		atomic.AddInt64(&current, 1)
		f(event)
		if event.Error() == "" {
			t.Errorf("Expected error, got empty")
		}
		if event.Level() != flow.ErrorLevel {
			t.Errorf("Expected error event, got %s", event.Level())
		}
		if event.Stage() != flow.InCallback {
			t.Errorf("Expected stage, got %s", event.Stage())
		}
		t.Logf("Event[ %s ] end", event.Name())
		return true
	}
}

func callbackPanicHandler(t *testing.T) func(event flow.FlexEvent) (keepOn bool) {
	f := eventCheck(t)
	return func(event flow.FlexEvent) (keepOn bool) {
		t.Logf("Event[ %s ] start", event.Name())
		f(event)
		atomic.AddInt64(&current, 1)
		if event.Panic() == nil {
			t.Errorf("Expected panic, got nil")
		}
		if event.StackTrace() == nil {
			t.Errorf("Expected stack trace, got nil")
		}
		if event.Level() != flow.PanicLevel {
			t.Errorf("Expected panic event, got %s", event.Level())
		}
		if event.Stage() != flow.InCallback {
			t.Errorf("Expected stage, got %s", event.Stage())
		}
		t.Logf("Event[ %s ] end", event.Name())
		return true
	}
}

func TestCallbackErrorEvent(t *testing.T) {
	defer restEventEnv()
	defer resetCurrent()
	flow.HandlerRegistry().Handle(flow.InCallback, callbackErrHandler(t))

	wf := flow.RegisterFlow("TestCallbackErrorEvent")
	proc := wf.Process("TestCallbackErrorEvent")
	proc.AfterStep(true, Fx[flow.Step](t).Error().Callback())
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff := flow.DoneFlow("TestCallbackErrorEvent", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	resetCurrent()
	wf = flow.RegisterFlow("TestCallbackErrorEvent0")
	proc = wf.Process("TestCallbackErrorEvent0")
	proc.AfterProcess(true, Fx[flow.Process](t).Error().Callback())
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestCallbackErrorEvent0", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	resetCurrent()
	wf = flow.RegisterFlow("TestCallbackErrorEvent1")
	proc = wf.Process("TestCallbackErrorEvent1")
	wf.AfterFlow(true, Fx[flow.WorkFlow](t).Error().Callback())
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestCallbackErrorEvent1", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.CallbackFail)(any(ff).(flow.WorkFlow))
}

func TestCallbackPanicEvent(t *testing.T) {
	defer restEventEnv()
	defer resetCurrent()
	flow.HandlerRegistry().Handle(flow.InCallback, callbackPanicHandler(t))

	wf := flow.RegisterFlow("TestCallbackPanicEvent")
	proc := wf.Process("TestCallbackPanicEvent")
	proc.AfterStep(true, Fx[flow.Step](t).Panic().Callback())
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff := flow.DoneFlow("TestCallbackPanicEvent", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	resetCurrent()
	wf = flow.RegisterFlow("TestCallbackPanicEvent0")
	proc = wf.Process("TestCallbackPanicEvent0")
	proc.AfterProcess(true, Fx[flow.Process](t).Panic().Callback())
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestCallbackPanicEvent0", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	resetCurrent()
	wf = flow.RegisterFlow("TestCallbackPanicEvent1")
	proc = wf.Process("TestCallbackPanicEvent1")
	wf.AfterFlow(true, Fx[flow.WorkFlow](t).Panic().Callback())
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestCallbackPanicEvent1", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.CallbackFail)(any(ff).(flow.WorkFlow))
}
