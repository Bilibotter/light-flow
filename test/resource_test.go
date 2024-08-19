package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"strconv"
	"sync/atomic"
	"testing"
)

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

	defer resetCurrent()
	letGo = false
	wf = flow.RegisterFlow("TestNoRegisterResourceAttach2")
	process = wf.Process("TestNoRegisterResourceAttach2")
	process.NameStep(Fx[flow.Step](t).Attach("NoExistResource", "***", nil).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire("NoExistResource", "***", nil).Step(), "3")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Error, flow.Panic))
	flow.DoneFlow("TestNoRegisterResourceAttach2", nil)
}

func TestResourceAttach(t *testing.T) {
	defer resetCurrent()
	letGo = false
	flow.RegisterResourceManager("TestResourceAttachResource")
	wf := flow.RegisterFlow("TestResourceAttach")
	process := wf.Process("TestResourceAttach")
	process.NameStep(Fx[flow.Step](t).Attach("TestResourceAttachResource", "***", map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire("TestResourceAttachResource", nil, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire("TestResourceAttachResource", nil, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestResourceAttach", nil)

	resetCurrent()
	letGo = false
	flow.RegisterResourceManager("TestResourceAttachResource2")
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

func TestResourceInitialize(t *testing.T) {
	defer resetCurrent()
	letGo = false
	initParam := "*"
	resName := "TestResourceInitialize"
	flow.RegisterResourceManager(resName).
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
	letGo = false
	initParam := "*"
	resName := "TestResourceInitializeError"
	flow.RegisterResourceManager("TestResourceInitializeResourceError").
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
	letGo = false
	initParam := "*"
	resName := "TestResourceInitializePanic"
	flow.RegisterResourceManager(resName).
		OnInitialize(resourceInitPanic)
	wf := flow.RegisterFlow("TestResourceInitializePanic")
	process := wf.Process("TestResourceInitializePanic")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2", "1")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Panic))
	flow.DoneFlow("TestResourceInitializePanic", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestResourceInitializePanic1")
	process = wf.Process("TestResourceInitializePanic1")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Broadcast().Step(), "1")
	process.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Panic))
	flow.DoneFlow("TestResourceInitializePanic1", nil)
}

func TestResourceUpdate(t *testing.T) {
	defer resetCurrent()
	letGo = false
	initParam := "*"
	resName := "TestResourceUpdate0"
	flow.RegisterResourceManager(resName).
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
	letGo = false
	initParam := "*"
	resName := "TestResourceRelease"
	flow.RegisterResourceManager(resName).
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
	letGo = false
	initParam := "*"
	resName := "TestResourceReleaseError"
	flow.RegisterResourceManager(resName).
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
	letGo = false
	initParam := "*"
	resName := "TestResourceReleasePanic"
	flow.RegisterResourceManager(resName).
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
	letGo = false
	initParam := "*"
	resName := "TestReleaseAttachedFailedResource"
	flow.RegisterResourceManager(resName).
		OnInitialize(resourceInitError).
		OnRelease(resourceRelease(t))
	wf := flow.RegisterFlow("TestReleaseAttachedFailedResource")
	process := wf.Process("TestReleaseAttachedFailedResource")
	process.NameStep(func(ctx flow.Step) (any, error) {
		t.Logf("Step[ %s ] start", ctx.Name())
		ctx.Attach(resName, initParam+resName)
		atomic.AddInt64(&current, 1)
		letGo = true
		t.Logf("Step[ %s ] end", ctx.Name())
		return ctx.Name(), nil
	}, "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	wf.AfterFlow(false, CheckResult(t, 1, flow.Panic))
	flow.DoneFlow("TestReleaseAttachedFailedResource", nil)
}

func TestReleaseAttachedPanicResource(t *testing.T) {
	defer resetCurrent()
	letGo = false
	initParam := "*"
	resName := "TestReleaseAttachedPanicResource"
	flow.RegisterResourceManager(resName).
		OnInitialize(resourceInitPanic).
		OnRelease(resourceRelease(t))
	wf := flow.RegisterFlow("TestReleaseAttachedPanicResource")
	process := wf.Process("TestReleaseAttachedPanicResource")
	process.NameStep(func(ctx flow.Step) (any, error) {
		t.Logf("Step[ %s ] start", ctx.Name())
		ctx.Attach(resName, initParam+resName)
		atomic.AddInt64(&current, 1)
		letGo = true
		t.Logf("Step[ %s ] end", ctx.Name())
		return ctx.Name(), nil
	}, "1")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Panic))
	flow.DoneFlow("TestReleaseAttachedPanicResource", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestReleaseAttachedPanicResource0")
	process = wf.Process("TestReleaseAttachedPanicResource0")
	process.NameStep(func(ctx flow.Step) (any, error) {
		defer func() {
			letGo = true
		}()
		t.Logf("Step[ %s ] start", ctx.Name())
		ctx.Attach(resName, initParam+resName)
		atomic.AddInt64(&current, 1)
		t.Logf("Step[ %s ] end", ctx.Name())
		return ctx.Name(), nil
	}, "1")
	process.NameStep(Fx[flow.Step](t).Wait().Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "2")
	process.NameStep(Fx[flow.Step](t).Acquire(resName, initParam+resName, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Step(), "3", "1")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Panic))
	flow.DoneFlow("TestReleaseAttachedPanicResource0", nil)
}

func TestReleaseWhileStepError(t *testing.T) {
	defer resetCurrent()
	letGo = false
	initParam := "*"
	resName := "TestReleaseWhileStepError"
	flow.RegisterResourceManager(resName).
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
	letGo = false
	initParam := "*"
	resName := "TestReleaseWhileStepPanic"
	flow.RegisterResourceManager(resName).
		OnInitialize(resourceInit).
		OnRelease(resourceRelease(t))
	wf := flow.RegisterFlow("TestReleaseWhileStepPanic")
	process := wf.Process("TestReleaseWhileStepPanic")
	process.NameStep(Fx[flow.Step](t).Attach(resName, initParam, map[string]any{"int": 1, "str": "hello", "bool": true, "float": 3.14}).Panic().Step(), "1")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Panic))
	flow.DoneFlow("TestReleaseWhileStepPanic", nil)
}
