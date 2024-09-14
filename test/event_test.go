package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
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

func resetEventEnv() {
	flow.SuspendPersist(&persisitImpl{})
	flow.HandlerRegistry().Clear()
	flow.SetEncryptor(flow.NewAES256Encryptor([]byte("light-flow"), "pwd", "password"))
	resetCurrent()
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
		if event.Layer() == flow.ProcLyr {
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
		if event.Layer() == flow.StepLyr {
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

func flexErrorCheck(t *testing.T, stage flow.EventStage, location string) func(event flow.FlexEvent) bool {
	f := eventCheck(t)
	return func(event flow.FlexEvent) bool {
		t.Logf("Event[ %s ] start", event.Name())
		f(event)
		if event.Error() == "" {
			t.Errorf("Expected error, got empty")
		}
		if event.Level() != flow.ErrorLevel {
			t.Errorf("Expected error event, got %s", event.Level())
		}
		if event.Extra("Location") != location {
			t.Errorf("Expected location %s, got %s", location, event.Extra("Location"))
		}
		if event.Stage() != stage {
			t.Errorf("Expected stage %s, got %s", stage, event.Stage())
		}
		t.Logf("Event[ %s ] end", event.Name())
		atomic.AddInt64(&current, 1)
		return true
	}
}

func flexPanicCheck(t *testing.T, stage flow.EventStage, location string) func(event flow.FlexEvent) bool {
	f := eventCheck(t)
	return func(event flow.FlexEvent) bool {
		t.Logf("Event[ %s ] start", event.Name())
		f(event)
		if event.Level() != flow.PanicLevel {
			t.Errorf("Expected error event, got %s", event.Level())
		}
		if event.StackTrace() == nil {
			t.Errorf("Expected stack trace, got nil")
		}
		if event.Extra("Location") != location {
			t.Errorf("Expected location %s, got %s", location, event.Extra("Location"))
		}
		if event.Stage() != stage {
			t.Errorf("Expected stage %s, got %s", stage, event.Stage())
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
		if event.Extra("Location") == "" {
			t.Errorf("Expected position, got empty string")
		}
		if event.Extra("Order") == "" {
			t.Errorf("Expected order, got empty string")
		}
		if event.Extra("Necessity") == "" {
			t.Errorf("Expected necessity, got empty string")
		}
		if event.Extra("Scope") == "" {
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
		if event.Extra("Location") == "" {
			t.Errorf("Expected position, got empty string")
		}
		if event.Extra("Order") == "" {
			t.Errorf("Expected order, got empty string")
		}
		if event.Extra("Necessity") == "" {
			t.Errorf("Expected necessity, got empty string")
		}
		if event.Extra("Scope") == "" {
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
		if event.Extra("Location") != "Encrypt" {
			t.Errorf("Expected position, got %s", event.Extra("Location"))
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
		if event.Extra("Location") != "Persist" {
			t.Errorf("Expected position, got %s", event.Extra("Location"))
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
		if event.Extra("Location") != "Serialize" {
			t.Errorf("Expected position, got %s", event.Extra("Location"))
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
	defer resetEventEnv()
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

func TestSuspendEncryptErrorEvent(t *testing.T) {
	defer resetEventEnv()
	flow.SetEncryptor(&errorEncryptor{encryptFailed: true})
	flow.HandlerRegistry().Handle(flow.InSuspend, encryptErrorHandler(t))
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
	flow.HandlerRegistry().Handle(flow.InSuspend, suspendPanicHandler(t))
	wf = flow.RegisterFlow("TestEncryptPanicEvent")
	wf.EnableRecover()
	process = wf.Process("TestEncryptPanicEvent")
	process.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	ff = flow.DoneFlow("TestEncryptPanicEvent", nil)
	waitCurrent(3)
	CheckResult(t, 3, flow.Error)(any(ff).(flow.WorkFlow))
	_, err = ff.Recover()
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		t.Logf("Got error: %v", err)
	}
}

func TestSuspendSaveErrorEvent(t *testing.T) {
	defer resetEventEnv()
	flow.SuspendPersist(&errorSave{persist: &persisitImpl{}, saveCheckpointAndRecordError: true})
	flow.HandlerRegistry().Handle(flow.InSuspend, saveErrorHandler(t))
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
	flow.HandlerRegistry().Handle(flow.InSuspend, suspendPanicHandler(t))
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
	flow.HandlerRegistry().Handle(flow.InSuspend, serializeErrorHandler(t))
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
	_, err := ff.Recover()
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		t.Logf("Got error: %v", err)
	}
}

func TestRecoverPersistErrorEvent(t *testing.T) {
	defer resetEventEnv()
	flow.SuspendPersist(&errorSave{persist: &persisitImpl{}, latestRecordError: true})
	flow.HandlerRegistry().Handle(flow.InRecover, flexErrorCheck(t, flow.InRecover, "Persist"))
	wf := flow.RegisterFlow("TestRecoverPersistErrorEvent")
	wf.EnableRecover()
	proc := wf.Process("TestRecoverPersistErrorEvent")
	proc.NameStep(Fx[flow.Step](t).SetCtx().Recover().Step(), "1")
	ff := flow.DoneFlow("TestRecoverPersistErrorEvent", nil)
	CheckResult(t, 2, flow.Error)(any(ff).(flow.WorkFlow))
	_, err := ff.Recover()
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else {
		t.Logf("Got error: %v", err)
	}
	waitCurrent(3)

	resetEventEnv()
	flow.SuspendPersist(&errorSave{persist: &persisitImpl{}, listPointsError: true})
	flow.HandlerRegistry().Handle(flow.InRecover, flexErrorCheck(t, flow.InRecover, "Persist"))
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
	flow.HandlerRegistry().Handle(flow.InRecover, flexErrorCheck(t, flow.InRecover, "Persist"))
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
	flow.HandlerRegistry().Handle(flow.InRecover, flexPanicCheck(t, flow.InRecover, ""))
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
	flow.HandlerRegistry().Handle(flow.InRecover, flexPanicCheck(t, flow.InRecover, ""))
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
	flow.HandlerRegistry().Handle(flow.InRecover, flexPanicCheck(t, flow.InRecover, ""))
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
