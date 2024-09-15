package test

import (
	"errors"
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"strconv"
	"sync/atomic"
	"testing"
)

func suspendResource(t *testing.T, check, update any, kv map[string]any) func(res flow.Resource) error {
	return func(res flow.Resource) error {
		t.Logf("[Process: %s ] start suspend Resource[ %s ]", res.ProcessName(), res.Name())
		if res.Entity() != check {
			t.Errorf("Resource[ %s ] entity expect %v, but got %v", res.Name(), check, res.Entity())
			return fmt.Errorf("Resource[ %s ] entity expect %v, but got %v", res.Name(), check, res.Entity())
		}
		last := res.Update(update)
		if last != update {
			t.Errorf("Resource[ %s ] update expect %v, but got %v", res.Name(), update, last)
			return fmt.Errorf("Resource[ %s ] update expect %v, but got %v", res.Name(), update, last)
		}
		for k, v := range kv {
			res.Put(k, v)
		}
		atomic.AddInt64(&current, 1)
		t.Logf("[Process: %s ] finish suspend Resource[ %s ]", res.ProcessName(), res.Name())
		return nil
	}
}

func recoverResource(t *testing.T, check, update any, kv map[string]any) func(res flow.Resource) error {
	return func(res flow.Resource) error {
		t.Logf("[Process: %s ] start recover Resource[ %s ]", res.ProcessName(), res.Name())
		if res.Entity() != check {
			t.Errorf("Resource[ %s ] entity expect %v, but got %v", res.Name(), check, res.Entity())
			return fmt.Errorf("Resource[ %s ] entity expect %v, but got %v", res.Name(), check, res.Entity())
		}
		last := res.Update(update)
		if last != update {
			t.Errorf("Resource[ %s ] update expect %v, but got %v", res.Name(), update, last)
			return fmt.Errorf("Resource[ %s ] update expect %v, but got %v", res.Name(), update, last)
		}
		for k, v := range kv {
			if v1, exist := res.Fetch(k); !exist || v1 != v {
				t.Errorf("Resource[ %s ] get key %s expect %v, but got %v", res.Name(), k, v, v1)
				return fmt.Errorf("Resource[ %s ] get key %s expect %v, but got %v", res.Name(), k, v, v1)
			}
		}
		atomic.AddInt64(&current, 1)
		t.Logf("[Process: %s ] finish recover Resource[ %s ]", res.ProcessName(), res.Name())
		return nil
	}
}

func recoverResource0(t *testing.T, check, update any, kv map[string]any) func(res flow.Resource) error {
	return func(res flow.Resource) error {
		t.Logf("[Process: %s ] start recover Resource[ %s ]", res.ProcessName(), res.Name())
		if res.Entity() != check {
			t.Errorf("Resource[ %s ] entity expect %v, but got %v", res.Name(), check, res.Entity())
			return fmt.Errorf("Resource[ %s ] entity expect %v, but got %v", res.Name(), check, res.Entity())
		}
		last := res.Update(update)
		if last != update {
			t.Errorf("Resource[ %s ] update expect %v, but got %v", res.Name(), update, last)
			return fmt.Errorf("Resource[ %s ] update expect %v, but got %v", res.Name(), update, last)
		}
		for k, v := range kv {
			res.Put(k, v)
		}
		atomic.AddInt64(&current, 1)
		t.Logf("[Process: %s ] finish recover Resource[ %s ]", res.ProcessName(), res.Name())
		return nil
	}
}

func resourceRelease(t *testing.T) func(res flow.Resource) error {
	return func(res flow.Resource) error {
		t.Logf("Resource[ %s ] released", res.Name())
		atomic.AddInt64(&current, 1)
		return nil
	}
}

func resourceReleaseError(t *testing.T) func(res flow.Resource) error {
	return func(res flow.Resource) error {
		t.Logf("Resource[ %s ] release error", res.Name())
		atomic.AddInt64(&current, 1)
		return fmt.Errorf("Resource[ %s ] release error", res.Name())
	}
}

func resourceReleasePanic(t *testing.T) func(res flow.Resource) error {
	return func(res flow.Resource) error {
		t.Logf("Resource[ %s ] release panic", res.Name())
		atomic.AddInt64(&current, 1)
		panic("Resource[ " + res.Name() + " ] release panic")
	}
}

func resourceInit(res flow.Resource, initParam any) (entity any, err error) {
	return initParam.(string) + res.Name(), nil
}

func resourceInitError(res flow.Resource, initParam any) (entity any, err error) {
	return initParam.(string) + res.Name(), fmt.Errorf("Resource[ %s ] init error", res.Name())
}

func resourceInitPanic(res flow.Resource, _ any) (entity any, err error) {
	panic("Resource[" + res.Name() + "] init panic")
}

func TestNoRegisterResourceAttach(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestNoRegisterResourceAttach1")
	process := wf.Process("TestNoRegisterResourceAttach1")
	process.NameStep(Fx[flow.Step](t).Attach("NoExistResource", "***", nil).Step(), "2")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Error))
	flow.DoneFlow("TestNoRegisterResourceAttach1", nil)

	resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	wf = flow.RegisterFlow("TestNoRegisterResourceAttach2")
	process = wf.Process("TestNoRegisterResourceAttach2")
	process.NameStep(Fx[flow.Step](t).Attach("NoExistResource", "***", nil).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire("NoExistResource", "***", nil).Step(), "3")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Error, flow.Panic))
	flow.DoneFlow("TestNoRegisterResourceAttach2", nil)
}

func TestResourceAttach(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	flow.AddResource("TestResourceAttachResource")
	wf := flow.RegisterFlow("TestResourceAttach")
	process := wf.Process("TestResourceAttach")
	process.NameStep(Fx[flow.Step](t).Attach("TestResourceAttachResource", "***", map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire("TestResourceAttachResource", nil, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire("TestResourceAttachResource", nil, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestResourceAttach", nil)

	resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	flow.AddResource("TestResourceAttachResource2")
	wf = flow.RegisterFlow("TestResourceAttach2")
	process = wf.Process("TestResourceAttach2")
	process.NameStep(Fx[flow.Step](t).Attach("TestResourceAttachResource", "***", map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire("TestResourceAttachResource", nil, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire("TestResourceAttachResource", nil, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	// concurrent acquire
	for i := 4; i <= 20; i++ {
		process.NameStep(Fx[flow.Step](t).Acquire("TestResourceAttachResource", nil, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), strconv.Itoa(i), "3")
	}
	wf.AfterFlow(false, CheckResult(t, 20, flow.Success))
	flow.DoneFlow("TestResourceAttach2", nil)
}

func TestResourceAttachInBeforeProcess(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	flow.AddResource("TestResourceAttachInBeforeProcess0")
	wf := flow.RegisterFlow("TestResourceAttachInBeforeProcess0")
	process := wf.Process("TestResourceAttachInBeforeProcess0")
	process.BeforeProcess(true, Fx[flow.Process](t).Attach("TestResourceAttachInBeforeProcess0", "***", map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Callback())
	process.NameStep(Fx[flow.Step](t).Acquire("TestResourceAttachInBeforeProcess0", nil, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "1")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestResourceAttachInBeforeProcess0", nil)

	// Test if the resource will be released while before-process callback failed
	resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	flow.AddResource("TestResourceAttachInBeforeProcess1").
		OnRelease(resourceRelease(t)).
		OnInitialize(resourceInit)
	wf = flow.RegisterFlow("TestResourceAttachInBeforeProcess1")
	process = wf.Process("TestResourceAttachInBeforeProcess1")
	process.BeforeProcess(true, Fx[flow.Process](t).Attach("TestResourceAttachInBeforeProcess1", "***", map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Callback())
	process.BeforeProcess(true, Fx[flow.Process](t).Error().Callback())
	ff := flow.DoneFlow("TestResourceAttachInBeforeProcess1", nil)
	CheckResult(t, 3, flow.CallbackFail)(any(ff).(flow.WorkFlow))
}

func TestResourceInitialize(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestResourceInitialize"
	flow.AddResource(resName).
		OnInitialize(resourceInit)
	wf := flow.RegisterFlow("TestResourceInitialize")
	process := wf.Process("TestResourceInitialize")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestResourceInitialize", nil)
}

func TestResourceInitializeError(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestResourceInitializeError"
	flow.AddResource("TestResourceInitializeResourceError").
		OnInitialize(resourceInitError)
	wf := flow.RegisterFlow("TestResourceInitializeError")
	process := wf.Process("TestResourceInitializeError")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Error, flow.Panic))
	flow.DoneFlow("TestResourceInitializeError", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestResourceInitializeError1")
	process = wf.Process("TestResourceInitializeError1")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Error))
	flow.DoneFlow("TestResourceInitializeError1", nil)
}

func TestResourceInitializePanic(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestResourceInitializePanic"
	flow.AddResource(resName).
		OnInitialize(resourceInitPanic)
	wf := flow.RegisterFlow("TestResourceInitializePanic")
	process := wf.Process("TestResourceInitializePanic")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2", "1")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Error))
	flow.DoneFlow("TestResourceInitializePanic", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestResourceInitializePanic1")
	process = wf.Process("TestResourceInitializePanic1")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Error))
	flow.DoneFlow("TestResourceInitializePanic1", nil)
}

func TestResourceUpdate(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestResourceUpdate0"
	flow.AddResource(resName).
		OnInitialize(resourceInit).
		OnRelease(resourceRelease(t))
	wf := flow.RegisterFlow("TestResourceUpdate0")
	process := wf.Process("TestResourceUpdate0")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	process.NameStep(func(ctx flow.Step) (any, error) {
		r, _ := ctx.Acquire(resName)
		r.Update("TestResourceUpdate0")
		atomic.AddInt64(&current, 1)
		return nil, nil
	}, "4", "2", "3")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, "TestResourceUpdate0", map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "5", "4")
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestResourceUpdate0", nil)
}

func TestResourceRelease(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestResourceRelease"
	flow.AddResource(resName).
		OnInitialize(resourceInit).
		OnRelease(resourceRelease(t))
	wf := flow.RegisterFlow("TestResourceRelease")
	process := wf.Process("TestResourceRelease")
	process.NameStep(Fx[flow.Step](t).Attach("TestResourceRelease", initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire("TestResourceRelease", initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire("TestResourceRelease", initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestResourceRelease", nil)
}

func TestResourceReleaseError(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestResourceReleaseError"
	flow.AddResource(resName).
		OnInitialize(resourceInit).
		OnRelease(resourceReleaseError(t))
	wf := flow.RegisterFlow("TestResourceReleaseError")
	process := wf.Process("TestResourceReleaseError")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestResourceReleaseError", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestResourceReleaseError1")
	process = wf.Process("TestResourceReleaseError1")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestResourceReleaseError1", nil)
}

func TestResourceReleasePanic(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestResourceReleasePanic"
	flow.AddResource(resName).
		OnInitialize(resourceInit).
		OnRelease(resourceReleasePanic(t))
	wf := flow.RegisterFlow("TestResourceReleasePanic")
	process := wf.Process("TestResourceReleasePanic")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestResourceReleasePanic", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestResourceReleasePanic1")
	process = wf.Process("TestResourceReleasePanic1")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestResourceReleasePanic1", nil)
}

func TestReleaseAttachedFailedResource(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestReleaseAttachedFailedResource"
	flow.AddResource(resName).
		OnInitialize(resourceInitError).
		OnRelease(resourceRelease(t))
	wf := flow.RegisterFlow("TestReleaseAttachedFailedResource")
	process := wf.Process("TestReleaseAttachedFailedResource")
	process.NameStep(func(ctx flow.Step) (any, error) {
		t.Logf("[Step: %s ] start", ctx.Name())
		ctx.Attach(resName, initParam+resName)
		atomic.AddInt64(&current, 1)
		atomic.StoreInt64(&letGo, 1)
		t.Logf("[Step: %s ] end", ctx.Name())
		return ctx.Name(), nil
	}, "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	wf.AfterFlow(false, CheckResult(t, 1, flow.Panic))
	flow.DoneFlow("TestReleaseAttachedFailedResource", nil)
}

func TestReleaseAttachedPanicResource(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestReleaseAttachedPanicResource"
	flow.AddResource(resName).
		OnInitialize(resourceInitPanic).
		OnRelease(resourceRelease(t))
	wf := flow.RegisterFlow("TestReleaseAttachedPanicResource")
	process := wf.Process("TestReleaseAttachedPanicResource")
	process.NameStep(func(ctx flow.Step) (any, error) {
		t.Logf("[Step: %s ] start", ctx.Name())
		if _, err := ctx.Attach(resName, initParam+resName); err != nil {
			panic(fmt.Sprintf("attach resource failed: %s", err.Error()))
		}
		atomic.AddInt64(&current, 1)
		atomic.StoreInt64(&letGo, 1)
		t.Logf("[Step: %s ] end", ctx.Name())
		return ctx.Name(), nil
	}, "1")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Panic))
	flow.DoneFlow("TestReleaseAttachedPanicResource", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestReleaseAttachedPanicResource0")
	process = wf.Process("TestReleaseAttachedPanicResource0")
	process.NameStep(func(ctx flow.Step) (any, error) {
		defer func() {
			atomic.StoreInt64(&letGo, 1)
		}()
		t.Logf("[Step: %s ] start", ctx.Name())
		if _, err := ctx.Attach(resName, initParam+resName); err != nil {
			panic(fmt.Sprintf("attach resource failed: %s", err.Error()))
		}
		atomic.AddInt64(&current, 1)
		t.Logf("[Step: %s ] end", ctx.Name())
		return ctx.Name(), nil
	}, "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Panic))
	flow.DoneFlow("TestReleaseAttachedPanicResource0", nil)
}

func TestReleaseWhileStepError(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestReleaseWhileStepError"
	flow.AddResource(resName).
		OnInitialize(resourceInit).
		OnRelease(resourceRelease(t))
	wf := flow.RegisterFlow("TestReleaseWhileStepError")
	process := wf.Process("TestReleaseWhileStepError")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Error().Step(), "1")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Error))
	flow.DoneFlow("TestReleaseWhileStepError", nil)
}

func TestReleaseWhileStepPanic(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestReleaseWhileStepPanic"
	flow.AddResource(resName).
		OnInitialize(resourceInit).
		OnRelease(resourceRelease(t))
	wf := flow.RegisterFlow("TestReleaseWhileStepPanic")
	process := wf.Process("TestReleaseWhileStepPanic")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Panic().Step(), "1")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Panic))
	flow.DoneFlow("TestReleaseWhileStepPanic", nil)
}

func TestResourceRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestResourceRecover"
	flow.AddResource(resName).
		OnInitialize(resourceInit).
		OnRelease(resourceRelease(t)).
		OnRecover(recoverResource(t, initParam+resName, initParam+initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}))
	wf := flow.RegisterFlow("TestResourceRecover")
	wf.EnableRecover()
	process := wf.Process("TestResourceRecover")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().SetCtx().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Step(), "3", "1")
	process.NameStep(Fn(t).Fail(CheckCtx("2")).Suc(CheckCtx("2")).ErrStep(), "4", "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Step(), "5", "4")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestResourceRecover", nil)
	CheckResult(t, 6, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestResourceRecover")
}

func TestResourceRecover0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestResourceRecover0"
	flow.AddResource(resName).
		OnInitialize(resourceInit).
		OnRelease(resourceRelease(t)).
		OnRecover(recoverResource0(t, initParam+resName, initParam+initParam+resName, map[string]any{"int": 10, "str": "hello0", "bool": false, "float": 33.14, "person": Person{"Amy", 21}}))
	wf := flow.RegisterFlow("TestResourceRecover0")
	wf.EnableRecover()
	process := wf.Process("TestResourceRecover0")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().SetCtx().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Step(), "3", "1")
	process.NameStep(Fn(t).Fail(CheckCtx("2")).Suc(CheckCtx("2")).ErrStep(), "4", "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+initParam+resName, map[string]any{"int": 10, "str": "hello0", "bool": false, "float": 33.14, "person": Person{"Amy", 21}}).Step(), "5", "4")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestResourceRecover0", nil)
	CheckResult(t, 6, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestResourceRecover0")
}

func TestResourceSuspend(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestResourceSuspend"
	flow.AddResource(resName).
		OnInitialize(resourceInit).
		OnRelease(resourceRelease(t)).
		OnSuspend(suspendResource(t, initParam+resName, initParam+initParam+resName, map[string]any{"int": 10, "str": "hello0", "bool": false, "float": 33.14, "person": Person{"Amy", 21}}))
	wf := flow.RegisterFlow("TestResourceSuspend")
	wf.EnableRecover()
	process := wf.Process("TestResourceSuspend")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().SetCtx().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Step(), "3", "1")
	process.NameStep(Fn(t).Fail(CheckCtx("2")).Suc(CheckCtx("2")).ErrStep(), "4", "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+initParam+resName, map[string]any{"int": 10, "str": "hello0", "bool": false, "float": 33.14, "person": Person{"Amy", 21}}).Step(), "5", "4")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestResourceSuspend", nil)
	CheckResult(t, 7, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestResourceSuspend")
}

func TestResourceSuspendAndRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	atomic.StoreInt64(&letGo, 0)
	initParam := "*"
	resName := "TestResourceSuspendAndRecover"
	flow.AddResource(resName).
		OnInitialize(resourceInit).
		OnRelease(resourceRelease(t)).
		OnSuspend(suspendResource(t, initParam+resName, initParam+initParam+resName, map[string]any{"int": 10, "str": "hello0", "bool": false, "float": 33.14, "person": Person{"Amy", 21}})).
		OnRecover(recoverResource(t, initParam+initParam+resName, initParam+initParam+resName, map[string]any{"int": 10, "str": "hello0", "bool": false, "float": 33.14, "person": Person{"Amy", 21}}))
	wf := flow.RegisterFlow("TestResourceSuspendAndRecover")
	wf.EnableRecover()
	process := wf.Process("TestResourceSuspendAndRecover")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().SetCtx().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14, "person": Person{"Tom", 20}}).Step(), "3", "1")
	process.NameStep(Fn(t).Fail(CheckCtx("2")).Suc(CheckCtx("2")).ErrStep(), "4", "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+initParam+resName, map[string]any{"int": 10, "str": "hello0", "bool": false, "float": 33.14, "person": Person{"Amy", 21}}).Step(), "5", "4")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestResourceSuspendAndRecover", nil)
	CheckResult(t, 7, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestResourceSuspendAndRecover")
}

func TestResourceClearAndRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	resName := "TestResourceClearAndRecover"
	flow.AddResource(resName).
		OnInitialize(func(res flow.Resource, initParam any) (entity any, err error) {
			res.Put("password", "******")
			atomic.AddInt64(&current, 1)
			return "password", nil
		}).
		OnSuspend(func(res flow.Resource) error {
			res.Clear()
			atomic.AddInt64(&current, 1)
			return nil
		}).
		OnRecover(func(res flow.Resource) error {
			if _, exist := res.Fetch("password"); exist {
				t.Errorf("password should not exist")
			}
			if res.Entity() != nil {
				t.Errorf("entity should be nil")
			}
			res.Put("drowssap", "+++++")
			res.Update("drowssap")
			atomic.AddInt64(&current, 1)
			return nil
		})
	wf := flow.RegisterFlow("TestResourceClearAndRecover")
	wf.EnableRecover()
	process := wf.Process("TestResourceClearAndRecover")
	process.NameStep(func(ctx flow.Step) (any, error) {
		if !executeSuc {
			ctx.Attach(resName, nil)
			atomic.AddInt64(&current, 1)
			return nil, errors.New("execute failed")
		}
		res, _ := ctx.Acquire(resName)
		if res.Entity() != "drowssap" {
			t.Errorf("entity should be drowssap")
		}
		if _, exist := res.Fetch("password"); exist {
			t.Errorf("password should not exist")
		}
		if _, exist := res.Fetch("drowssap"); !exist {
			t.Errorf("drowssap should exist")
		}
		atomic.AddInt64(&current, 1)
		return nil, nil
	}, "1")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	ff := flow.DoneFlow("TestResourceClearAndRecover", nil)
	CheckResult(t, 3, flow.Error)(any(ff).(flow.WorkFlow))
	Recover("TestResourceClearAndRecover")
}
