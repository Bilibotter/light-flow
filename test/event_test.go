package test

import (
	"errors"
	"fmt"
	"github.com/Bilibotter/light-flow/flow"
	"sync/atomic"
	"testing"
)

type noSerializable struct {
	field1 string
	field2 string
}

type errorSave struct {
	persist                      *persisitImpl
	latestRecordError            bool
	latestRecordPanic            bool
	listPointsError              bool
	listPointsPanic              bool
	updateRecordError            bool
	updateRecordPanic            bool
	saveCheckpointAndRecordError bool
	saveCheckpointAndRecordPanic bool
}

func errorResource(_ flow.Resource) error {
	return fmt.Errorf("resource error")
}

func panicResource(_ flow.Resource) error {
	panic("resource panic")
}

func (e *errorSave) GetLatestRecord(rootUid string) (flow.RecoverRecord, error) {
	if e.latestRecordError {
		return nil, fmt.Errorf("save error")
	}
	if e.latestRecordPanic {
		panic("save panic")
	}
	return e.persist.GetLatestRecord(rootUid)
}

func (e *errorSave) ListCheckpoints(recoveryId string) ([]flow.CheckPoint, error) {
	if e.listPointsError {
		return nil, fmt.Errorf("save error")
	}
	if e.listPointsPanic {
		panic("save panic")
	}
	return e.persist.ListCheckpoints(recoveryId)
}

func (e *errorSave) UpdateRecordStatus(record flow.RecoverRecord) error {
	if e.updateRecordError {
		return fmt.Errorf("save error")
	}
	if e.updateRecordPanic {
		panic("save panic")
	}
	return e.persist.UpdateRecordStatus(record)
}

func (e *errorSave) SaveCheckpointAndRecord(checkpoint []flow.CheckPoint, record flow.RecoverRecord) error {
	if e.saveCheckpointAndRecordError {
		return fmt.Errorf("save error")
	}
	if e.saveCheckpointAndRecordPanic {
		panic("save panic")
	}
	return e.persist.SaveCheckpointAndRecord(checkpoint, record)
}

type errorEncryptor struct {
	encryptFailed bool
	encryptPanic  bool
	decryptFailed bool
	decryptPanic  bool
}

func (e *errorEncryptor) Encrypt(plainText string, _ []byte) (string, error) {
	if e.encryptFailed {
		return "", fmt.Errorf("encrypt error")
	}
	if e.encryptPanic {
		panic("encrypt panic")
	}
	return plainText, nil
}

func (e *errorEncryptor) Decrypt(cipherText string, _ []byte) (string, error) {
	if e.decryptFailed {
		return "", fmt.Errorf("decrypt error")
	}
	if e.decryptPanic {
		panic("decrypt panic")
	}
	return cipherText, nil
}

func (e *errorEncryptor) NeedEncrypt(key string) bool {
	return true
}

func (e *errorEncryptor) GetSecret() []byte {
	return []byte("secret")
}

func errorStepFunc(_ flow.Step) error { return fmt.Errorf("error step persist") }

func errorProcFunc(_ flow.Process) error { return fmt.Errorf("error proc persist") }

func errorFlowFunc(_ flow.WorkFlow) error { return fmt.Errorf("error flow persist") }

func panicStepFunc(_ flow.Step) error { panic("step panic") }

func panicProcFunc(_ flow.Process) error { panic("proc panic") }

func panicFlowFunc(_ flow.WorkFlow) error { panic("flow panic") }

func resetEventEnv() {
	flow.SuspendPersist(&persisitImpl{})
	flow.EventHandler().Clear()
	flow.SetEncryptor(flow.NewAES256Encryptor([]byte("light-flow"), "pwd", "password"))
	resetCurrent()
	resetPersist()
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
		if event.Layer() == flow.ProcLayer {
			if event.FlowName() == "" {
				t.Errorf("Expected flow name, got empty string")
			}
			if event.FlowName() == "" {
				t.Errorf("Expected flow name, got empty string")
			}
			if event.ProcessID() == "" {
				t.Errorf("Expected proc ID, got empty string")
			}
			if event.ProcessName() == "" {
				t.Errorf("Expected proc name, got empty string")
			}
		}
		if event.Layer() == flow.StepLayer {
			if event.FlowID() == "" {
				t.Errorf("Expected flow ID, got empty string")
			}
			if event.FlowName() == "" {
				t.Errorf("Expected flow name, got empty string")
			}
			if event.ProcessID() == "" {
				t.Errorf("Expected proc ID, got empty string")
			}
			if event.ProcessName() == "" {
				t.Errorf("Expected proc name, got empty string")
			}
		}
	}
}

func flexErrorCheck(t *testing.T, stage flow.EventStage, location string, checks ...map[string]string) func(event flow.FlexEvent) bool {
	f := eventCheck(t)
	return func(event flow.FlexEvent) bool {
		t.Logf("Event[ %s ] start, stage: %s, location: %s", event.Name(), event.Stage(), event.Details("Location"))
		f(event)
		if event.Error() == "" {
			t.Errorf("Expected error, got empty")
		}
		if event.Level() != flow.ErrorLevel {
			t.Errorf("Expected error event, got %s", event.Level())
		}
		if event.Details("Location") != location {
			t.Errorf("Expected location %s, got %s", location, event.Details("Location"))
		}
		if event.Stage() != stage {
			t.Errorf("Expected stage %s, got %s", stage, event.Stage())
		}
		if len(checks) > 0 {
			check := checks[0]
			for k, v := range check {
				if event.Details(k) != v {
					t.Errorf("Expected details %s=%s, got %s", k, v, event.Details(k))
				}
			}
		}
		t.Logf("Event[ %s ] end", event.Name())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func flexPanicCheck(t *testing.T, stage flow.EventStage, location string, checks ...map[string]string) func(event flow.FlexEvent) bool {
	f := eventCheck(t)
	return func(event flow.FlexEvent) bool {
		t.Logf("Event[ %s ] start", event.Name())
		f(event)
		if event.Level() != flow.PanicLevel {
			t.Errorf("Expected panic event, got %s", event.Level())
		}
		if event.StackTrace() == nil {
			t.Errorf("Expected stack trace, got nil")
		}
		if event.Details("Location") != location {
			t.Errorf("Expected location %s, got %s", location, event.Details("Location"))
		}
		if event.Stage() != stage {
			t.Errorf("Expected stage %s, got %s", stage, event.Stage())
		}
		if len(checks) > 0 {
			check := checks[0]
			for k, v := range check {
				if event.Details(k) != v {
					t.Errorf("Expected details %s=%s, got %s", k, v, event.Details(k))
				}
			}
		}
		t.Logf("Event[ %s ] end", event.Name())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func callbackErrHandler(t *testing.T) func(event flow.FlexEvent) (keepOn bool) {
	f := eventCheck(t)
	return func(event flow.FlexEvent) (keepOn bool) {
		t.Logf("Event[ %s ] start", event.Name())
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
		if event.Details("Location") == "" {
			t.Errorf("Expected position, got empty string")
		}
		if event.Details("Order") == "" {
			t.Errorf("Expected order, got empty string")
		}
		if event.Details("Necessity") == "" {
			t.Errorf("Expected necessity, got empty string")
		}
		if event.Details("Scope") == "" {
			t.Errorf("Expected scope, got empty string")
		}
		t.Logf("Event[ %s ] end", event.Name())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func callbackPanicHandler(t *testing.T) func(event flow.FlexEvent) (keepOn bool) {
	f := eventCheck(t)
	return func(event flow.FlexEvent) (keepOn bool) {
		t.Logf("Event[ %s ] start", event.Name())
		f(event)
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
		if event.Details("Location") == "" {
			t.Errorf("Expected position, got empty string")
		}
		if event.Details("Order") == "" {
			t.Errorf("Expected order, got empty string")
		}
		if event.Details("Necessity") == "" {
			t.Errorf("Expected necessity, got empty string")
		}
		if event.Details("Scope") == "" {
			t.Errorf("Expected scope, got empty string")
		}
		t.Logf("Event[ %s ] end", event.Name())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func encryptErrorHandler(t *testing.T) func(event flow.FlexEvent) (keepOn bool) {
	f := eventCheck(t)
	return func(event flow.FlexEvent) (keepOn bool) {
		t.Logf("Event[ %s ] start", event.Name())
		f(event)
		if event.Error() == "" {
			t.Errorf("Expected error, got empty")
		}
		if event.Level() != flow.ErrorLevel {
			t.Errorf("Expected error event, got %s", event.Level())
		}
		if event.Stage() != flow.InSuspend {
			t.Errorf("Expected stage, got %s", event.Stage())
		}
		if event.Details("Location") != "Encrypt" {
			t.Errorf("Expected position, got %s", event.Details("Location"))
		}
		t.Logf("Event[ %s ] end", event.Name())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func saveErrorHandler(t *testing.T) func(event flow.FlexEvent) (keepOn bool) {
	f := eventCheck(t)
	return func(event flow.FlexEvent) (keepOn bool) {
		t.Logf("Event[ %s ] start", event.Name())
		f(event)
		if event.Error() == "" {
			t.Errorf("Expected error, got empty")
		}
		if event.Level() != flow.ErrorLevel {
			t.Errorf("Expected error event, got %s", event.Level())
		}
		if event.Stage() != flow.InSuspend {
			t.Errorf("Expected stage, got %s", event.Stage())
		}
		if event.Details("Location") != "Persist" {
			t.Errorf("Expected position, got %s", event.Details("Location"))
		}
		t.Logf("Event[ %s ] end", event.Name())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func serializeErrorHandler(t *testing.T) func(event flow.FlexEvent) (keepOn bool) {
	f := eventCheck(t)
	return func(event flow.FlexEvent) (keepOn bool) {
		t.Logf("Event[ %s ] start", event.Name())
		f(event)
		if event.Error() == "" {
			t.Errorf("Expected error, got empty")
		}
		if event.Level() != flow.ErrorLevel {
			t.Errorf("Expected error event, got %s", event.Level())
		}
		if event.Stage() != flow.InSuspend {
			t.Errorf("Expected stage, got %s", event.Stage())
		}
		if event.Details("Location") != "Serialize" {
			t.Errorf("Expected position, got %s", event.Details("Location"))
		}
		t.Logf("Event[ %s ] end", event.Name())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func suspendPanicHandler(t *testing.T) func(event flow.FlexEvent) (keepOn bool) {
	f := eventCheck(t)
	return func(event flow.FlexEvent) (keepOn bool) {
		t.Logf("Event[ %s ] start", event.Name())
		f(event)
		if event.Panic() == nil {
			t.Errorf("Expected panic, got nil")
		}
		if event.StackTrace() == nil {
			t.Errorf("Expected stack trace, got nil")
		}
		if event.Level() != flow.PanicLevel {
			t.Errorf("Expected panic event, got %s", event.Level())
		}
		if event.Stage() != flow.InSuspend {
			t.Errorf("Expected stage, got %s", event.Stage())
		}
		t.Logf("Event[ %s ] end", event.Name())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func TestCallbackErrorEvent(t *testing.T) {
	defer resetEventEnv()
	flow.EventHandler().Handle(flow.InCallback, callbackErrHandler(t))

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
	defer resetEventEnv()
	flow.EventHandler().Handle(flow.InCallback, callbackPanicHandler(t))

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

func TestSuspendEncryptErrorEvent(t *testing.T) {
	defer resetEventEnv()
	flow.SetEncryptor(&errorEncryptor{encryptFailed: true})
	flow.EventHandler().Handle(flow.InSuspend, encryptErrorHandler(t))
	wf := flow.RegisterFlow("TestSuspendEncryptErrorEvent")
	wf.EnableRecover()
	process := wf.Process("TestSuspendEncryptErrorEvent")
	process.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	ff := flow.DoneFlow("TestSuspendEncryptErrorEvent", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.Error)(any(ff).(flow.WorkFlow))
	_, err := ff.Recover()
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		t.Logf("Got error: %v", err)
	}

	resetEventEnv()
	flow.SetEncryptor(&errorEncryptor{encryptPanic: true})
	flow.EventHandler().Handle(flow.InSuspend, suspendPanicHandler(t))
	wf = flow.RegisterFlow("TestEncryptPanicEvent")
	wf.EnableRecover()
	process = wf.Process("TestEncryptPanicEvent")
	process.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	ff = flow.DoneFlow("TestEncryptPanicEvent", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.Error)(any(ff).(flow.WorkFlow))
}

func TestSuspendSaveErrorEvent(t *testing.T) {
	defer resetEventEnv()
	flow.SuspendPersist(&errorSave{persist: &persisitImpl{}, saveCheckpointAndRecordError: true})
	flow.EventHandler().Handle(flow.InSuspend, saveErrorHandler(t))
	wf := flow.RegisterFlow("TestSuspendSaveErrorEvent")
	wf.EnableRecover()
	process := wf.Process("TestSuspendSaveErrorEvent")
	process.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	ff := flow.DoneFlow("TestSuspendSaveErrorEvent", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.Error)(any(ff).(flow.WorkFlow))
	_, err := ff.Recover()
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		t.Logf("Got error: %v", err)
	}

	resetEventEnv()
	flow.SuspendPersist(&errorSave{persist: &persisitImpl{}, saveCheckpointAndRecordPanic: true})
	flow.EventHandler().Handle(flow.InSuspend, suspendPanicHandler(t))
	wf = flow.RegisterFlow("TestSavePanicEvent")
	wf.EnableRecover()
	process = wf.Process("TestSavePanicEvent")
	process.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	ff = flow.DoneFlow("TestSavePanicEvent", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.Error)(any(ff).(flow.WorkFlow))
}

func TestSuspendSerializeErrorEvent(t *testing.T) {
	defer resetEventEnv()
	flow.EventHandler().Handle(flow.InSuspend, serializeErrorHandler(t))
	wf := flow.RegisterFlow("TestSuspendSerializeErrorEvent")
	wf.EnableRecover()
	process := wf.Process("TestSuspendSerializeErrorEvent")
	process.NameStep(func(ctx flow.Step) (any, error) {
		ctx.Set("foo", noSerializable{"1", "2"})
		return nil, fmt.Errorf("error")
	}, "1")
	ff := flow.DoneFlow("TestSuspendSerializeErrorEvent", nil)
	waitCurrent(1)
	CheckResult(t, 1, flow.Error)(any(ff).(flow.WorkFlow))
}

func TestRecoverPersistErrorEvent(t *testing.T) {
	defer resetEventEnv()
	flow.SuspendPersist(&errorSave{persist: &persisitImpl{}, latestRecordError: true})
	flow.EventHandler().Handle(flow.InRecover, flexErrorCheck(t, flow.InRecover, "Persist"))
	wf := flow.RegisterFlow("TestRecoverPersistErrorEvent")
	wf.EnableRecover()
	proc := wf.Process("TestRecoverPersistErrorEvent")
	proc.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Error))
	ff := flow.DoneFlow("TestRecoverPersistErrorEvent", nil)
	_, err := ff.Recover()
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		t.Logf("Got error: %v", err)
	}
	waitCurrent(3)

	resetEventEnv()
	flow.SuspendPersist(&errorSave{persist: &persisitImpl{}, listPointsError: true})
	flow.EventHandler().Handle(flow.InRecover, flexErrorCheck(t, flow.InRecover, "Persist"))
	wf = flow.RegisterFlow("TestRecoverPersistErrorEvent0")
	wf.EnableRecover()
	proc = wf.Process("TestRecoverPersistErrorEvent0")
	proc.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	ff = flow.DoneFlow("TestRecoverPersistErrorEvent0", nil)
	CheckResult(t, 2, flow.Error)(any(ff).(flow.WorkFlow))
	_, err = ff.Recover()
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		t.Logf("Got error: %v", err)
	}
	waitCurrent(3)

	resetEventEnv()
	flow.SuspendPersist(&errorSave{persist: &persisitImpl{}, updateRecordError: true})
	flow.EventHandler().Handle(flow.InRecover, flexErrorCheck(t, flow.InRecover, "Persist"))
	wf = flow.RegisterFlow("TestRecoverPersistErrorEvent1")
	wf.EnableRecover()
	proc = wf.Process("TestRecoverPersistErrorEvent1")
	proc.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	ff = flow.DoneFlow("TestRecoverPersistErrorEvent1", nil)
	CheckResult(t, 2, flow.Error)(any(ff).(flow.WorkFlow))
	_, err = ff.Recover()
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		t.Logf("Got error: %v", err)
	}
	waitCurrent(3)

	resetEventEnv()
	flow.SuspendPersist(&errorSave{persist: &persisitImpl{}, latestRecordPanic: true})
	flow.EventHandler().Handle(flow.InRecover, flexPanicCheck(t, flow.InRecover, ""))
	wf = flow.RegisterFlow("TestRecoverPersistPanicEvent")
	wf.EnableRecover()
	proc = wf.Process("TestRecoverPersistPanicEvent")
	proc.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	ff = flow.DoneFlow("TestRecoverPersistPanicEvent", nil)
	CheckResult(t, 2, flow.Error)(any(ff).(flow.WorkFlow))
	_, err = ff.Recover()
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		t.Logf("Got error: %v", err)
	}
	waitCurrent(3)

	resetEventEnv()
	flow.SuspendPersist(&errorSave{persist: &persisitImpl{}, listPointsPanic: true})
	flow.EventHandler().Handle(flow.InRecover, flexPanicCheck(t, flow.InRecover, ""))
	wf = flow.RegisterFlow("TestRecoverPersistPanicEvent0")
	wf.EnableRecover()
	proc = wf.Process("TestRecoverPersistPanicEvent0")
	proc.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	ff = flow.DoneFlow("TestRecoverPersistPanicEvent0", nil)
	CheckResult(t, 2, flow.Error)(any(ff).(flow.WorkFlow))
	_, err = ff.Recover()
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		t.Logf("Got error: %v", err)
	}
	waitCurrent(3)

	resetEventEnv()
	flow.SuspendPersist(&errorSave{persist: &persisitImpl{}, updateRecordPanic: true})
	flow.EventHandler().Handle(flow.InRecover, flexPanicCheck(t, flow.InRecover, ""))
	wf = flow.RegisterFlow("TestRecoverPersistPanicEvent1")
	wf.EnableRecover()
	proc = wf.Process("TestRecoverPersistPanicEvent1")
	proc.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	ff = flow.DoneFlow("TestRecoverPersistPanicEvent1", nil)
	CheckResult(t, 2, flow.Error)(any(ff).(flow.WorkFlow))
	_, err = ff.Recover()
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		t.Logf("Got error: %v", err)
	}
	waitCurrent(3)
}

func TestResourceEvent(t *testing.T) {
	defer resetEventEnv()
	flow.AddResource("TestResourceEvent0").OnInitialize(func(res flow.Resource, initParam any) (entity any, err error) {
		return nil, errors.New("initialize error")
	})
	flow.EventHandler().Handle(flow.InResource, flexErrorCheck(t, flow.InResource, "", map[string]string{"Action": "Initialize", "Resource": "TestResourceEvent0"}))
	wf := flow.RegisterFlow("TestResourceEvent0")
	proc := wf.Process("TestResourceEvent0")
	proc.NameStep(Fx[flow.Step](t).Attach("TestResourceEvent0", nil, nil).Step(), "1")
	ff := flow.DoneFlow("TestResourceEvent0", nil)
	waitCurrent(1)
	CheckResult(t, 1, flow.Error)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.AddResource("TestResourceEvent1").OnInitialize(func(res flow.Resource, initParam any) (entity any, err error) {
		panic("initialize panic")
	})
	flow.EventHandler().Handle(flow.InResource, flexPanicCheck(t, flow.InResource, "", map[string]string{"Action": "Attach", "Resource": "TestResourceEvent1"}))
	wf = flow.RegisterFlow("TestResourceEvent1")
	proc = wf.Process("TestResourceEvent1")
	proc.NameStep(Fx[flow.Step](t).Attach("TestResourceEvent1", nil, nil).Step(), "1")
	ff = flow.DoneFlow("TestResourceEvent1", nil)
	waitCurrent(1)
	CheckResult(t, 1, flow.Error)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.AddResource("TestResourceEvent2").OnRelease(errorResource)
	flow.EventHandler().Handle(flow.InResource, flexErrorCheck(t, flow.InResource, "", map[string]string{"Action": "Release", "Resource": "TestResourceEvent2"}))
	wf = flow.RegisterFlow("TestResourceEvent2")
	proc = wf.Process("TestResourceEvent2")
	proc.NameStep(Fx[flow.Step](t).Attach("TestResourceEvent2", nil, nil).Step(), "1")
	ff = flow.DoneFlow("TestResourceEvent2", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.AddResource("TestResourceEvent3").OnRelease(panicResource)
	flow.EventHandler().Handle(flow.InResource, flexPanicCheck(t, flow.InResource, "", map[string]string{"Action": "Release", "Resource": "TestResourceEvent3"}))
	wf = flow.RegisterFlow("TestResourceEvent3")
	proc = wf.Process("TestResourceEvent3")
	proc.NameStep(Fx[flow.Step](t).Attach("TestResourceEvent3", nil, nil).Step(), "1")
	ff = flow.DoneFlow("TestResourceEvent3", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.AddResource("TestResourceEvent4").OnSuspend(errorResource)
	flow.EventHandler().Handle(flow.InResource, flexErrorCheck(t, flow.InResource, "", map[string]string{"Action": "Suspend", "Resource": "TestResourceEvent4"}))
	wf = flow.RegisterFlow("TestResourceEvent4")
	wf.EnableRecover()
	proc = wf.Process("TestResourceEvent4")
	proc.NameStep(Fx[flow.Step](t).Attach("TestResourceEvent4", nil, nil).Step(), "1")
	proc.NameStep(Fx[flow.Step](t).Recover().Step(), "2")
	ff = flow.DoneFlow("TestResourceEvent4", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.Failed)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.AddResource("TestResourceEvent5").OnSuspend(panicResource)
	flow.EventHandler().Handle(flow.InResource, flexPanicCheck(t, flow.InResource, "", map[string]string{"Action": "Suspend", "Resource": "TestResourceEvent5"}))
	wf = flow.RegisterFlow("TestResourceEvent5")
	wf.EnableRecover()
	proc = wf.Process("TestResourceEvent5")
	proc.NameStep(Fx[flow.Step](t).Attach("TestResourceEvent5", nil, nil).Step(), "1")
	proc.NameStep(Fx[flow.Step](t).Recover().Step(), "2", "1")
	ff = flow.DoneFlow("TestResourceEvent5", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.Failed)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.AddResource("TestResourceEvent6").OnRecover(errorResource)
	flow.EventHandler().Handle(flow.InResource, flexErrorCheck(t, flow.InResource, "", map[string]string{"Action": "Recover", "Resource": "TestResourceEvent6"}))
	wf = flow.RegisterFlow("TestResourceEvent6")
	wf.EnableRecover()
	proc = wf.Process("TestResourceEvent6")
	proc.NameStep(Fx[flow.Step](t).Attach("TestResourceEvent6", nil, nil).Step(), "1")
	proc.NameStep(Fx[flow.Step](t).Recover().Step(), "2", "1")
	ff = flow.DoneFlow("TestResourceEvent6", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Failed)(any(ff).(flow.WorkFlow))
	ff, err := ff.Recover()
	if err != nil {
		t.Errorf("expect error==nil, but got err %s", err.Error())
	}
	waitCurrent(4)

	resetEventEnv()
	flow.AddResource("TestResourceEvent7").OnRecover(panicResource)
	flow.EventHandler().Handle(flow.InResource, flexPanicCheck(t, flow.InResource, "", map[string]string{"Action": "Recover", "Resource": "TestResourceEvent7"}))
	wf = flow.RegisterFlow("TestResourceEvent7")
	wf.EnableRecover()
	proc = wf.Process("TestResourceEvent7")
	proc.NameStep(Fx[flow.Step](t).Attach("TestResourceEvent7", nil, nil).Step(), "1")
	proc.NameStep(Fx[flow.Step](t).Recover().Step(), "2", "1")
	ff = flow.DoneFlow("TestResourceEvent7", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Failed)(any(ff).(flow.WorkFlow))
	ff, err = ff.Recover()
	if err != nil {
		t.Errorf("expect error==nil, but got err %s", err.Error())
	}
	waitCurrent(4)
}

func TestPersistEvent(t *testing.T) {
	defer resetEventEnv()
	flow.FlowPersist().OnInsert(errorFlowFunc)
	flow.EventHandler().Handle(flow.InPersist, flexErrorCheck(t, flow.InPersist, "", map[string]string{"Action": "Insert"}))
	wf := flow.RegisterFlow("TestPersistEvent0")
	proc := wf.Process("TestPersistEvent0")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff := flow.DoneFlow("TestPersistEvent0", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.FlowPersist().OnInsert(panicFlowFunc)
	flow.EventHandler().Handle(flow.InPersist, flexPanicCheck(t, flow.InPersist, "", map[string]string{"Action": "Insert"}))
	wf = flow.RegisterFlow("TestPersistEvent1")
	proc = wf.Process("TestPersistEvent1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestPersistEvent1", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.ProcPersist().OnInsert(errorProcFunc)
	flow.EventHandler().Handle(flow.InPersist, flexErrorCheck(t, flow.InPersist, "", map[string]string{"Action": "Insert"}))
	wf = flow.RegisterFlow("TestPersistEvent2")
	proc = wf.Process("TestPersistEvent2")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestPersistEvent2", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.ProcPersist().OnInsert(panicProcFunc)
	flow.EventHandler().Handle(flow.InPersist, flexPanicCheck(t, flow.InPersist, "", map[string]string{"Action": "Insert"}))
	wf = flow.RegisterFlow("TestPersistEvent3")
	proc = wf.Process("TestPersistEvent3")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestPersistEvent3", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.StepPersist().OnInsert(errorStepFunc)
	flow.EventHandler().Handle(flow.InPersist, flexErrorCheck(t, flow.InPersist, "", map[string]string{"Action": "Insert"}))
	wf = flow.RegisterFlow("TestPersistEvent4")
	proc = wf.Process("TestPersistEvent4")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestPersistEvent4", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.StepPersist().OnInsert(panicStepFunc)
	flow.EventHandler().Handle(flow.InPersist, flexPanicCheck(t, flow.InPersist, "", map[string]string{"Action": "Insert"}))
	wf = flow.RegisterFlow("TestPersistEvent5")
	proc = wf.Process("TestPersistEvent5")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestPersistEvent5", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))
}

func TestPersistEvent0(t *testing.T) {
	defer resetEventEnv()
	flow.FlowPersist().OnUpdate(errorFlowFunc)
	flow.EventHandler().Handle(flow.InPersist, flexErrorCheck(t, flow.InPersist, "", map[string]string{"Action": "Update"}))
	wf := flow.RegisterFlow("TestPersistEvent00")
	proc := wf.Process("TestPersistEvent00")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff := flow.DoneFlow("TestPersistEvent00", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.FlowPersist().OnUpdate(panicFlowFunc)
	flow.EventHandler().Handle(flow.InPersist, flexPanicCheck(t, flow.InPersist, "", map[string]string{"Action": "Update"}))
	wf = flow.RegisterFlow("TestPersistEvent01")
	proc = wf.Process("TestPersistEvent01")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestPersistEvent01", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.ProcPersist().OnUpdate(errorProcFunc)
	flow.EventHandler().Handle(flow.InPersist, flexErrorCheck(t, flow.InPersist, "", map[string]string{"Action": "Update"}))
	wf = flow.RegisterFlow("TestPersistEvent02")
	proc = wf.Process("TestPersistEvent02")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestPersistEvent02", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.ProcPersist().OnUpdate(panicProcFunc)
	flow.EventHandler().Handle(flow.InPersist, flexPanicCheck(t, flow.InPersist, "", map[string]string{"Action": "Update"}))
	wf = flow.RegisterFlow("TestPersistEvent03")
	proc = wf.Process("TestPersistEvent03")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestPersistEvent03", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.StepPersist().OnUpdate(errorStepFunc)
	flow.EventHandler().Handle(flow.InPersist, flexErrorCheck(t, flow.InPersist, "", map[string]string{"Action": "Update"}))
	wf = flow.RegisterFlow("TestPersistEvent04")
	proc = wf.Process("TestPersistEvent04")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestPersistEvent04", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))

	resetEventEnv()
	flow.StepPersist().OnUpdate(panicStepFunc)
	flow.EventHandler().Handle(flow.InPersist, flexPanicCheck(t, flow.InPersist, "", map[string]string{"Action": "Update"}))
	wf = flow.RegisterFlow("TestPersistEvent05")
	proc = wf.Process("TestPersistEvent05")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestPersistEvent05", nil)
	waitCurrent(2)
	CheckResult(t, 2, flow.Success)(any(ff).(flow.WorkFlow))
}
