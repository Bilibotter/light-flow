package test

import (
	"fmt"
	"github.com/Bilibotter/light-flow/flow"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func CheckCurrent(t *testing.T, expect int64) func(ctx flow.Step) (bool, error) {
	return func(ctx flow.Step) (bool, error) {
		if atomic.LoadInt64(&current) != expect {
			t.Errorf("[Step: %s] check failed, current should be %d, but %d", ctx.Name(), expect, atomic.LoadInt64(&current))
		}
		return true, nil
	}
}

func ErrorResultPrinter(info flow.Step) (bool, error) {
	if !info.Success() {
		fmt.Printf("[Step: %s] error, explain=%v, err=%v\n", info.Name(), info.ExplainStatus(), info.Err())
	}
	return true, nil
}

func NormalStep4(_ flow.Step) (any, error) {
	atomic.AddInt64(&current, 1)
	fmt.Printf("4.normal step finish\n")
	return 4, nil
}

func NormalStep3(_ flow.Step) (any, error) {
	atomic.AddInt64(&current, 1)
	fmt.Printf("3.normal step finish\n")
	return 3, nil
}

func NormalStep2(_ flow.Step) (any, error) {
	atomic.AddInt64(&current, 1)
	fmt.Printf("2.normal step finish\n")
	return 2, nil
}

func NormalStep1(_ flow.Step) (any, error) {
	atomic.AddInt64(&current, 1)
	fmt.Printf("1.normal step finish\n")
	return 1, nil
}

func GenerateStep(i int, args ...any) func(ctx flow.Step) (any, error) {
	return func(ctx flow.Step) (any, error) {
		if len(args) > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Printf("%d.step finish\n", i)
		atomic.AddInt64(&current, 1)
		return i, nil
	}
}

func GenerateErrorStep(i int, args ...any) func(ctx flow.Step) (any, error) {
	return func(ctx flow.Step) (any, error) {
		if len(args) > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Printf("%d.step error occur \n", i)
		atomic.AddInt64(&current, 1)
		return i, fmt.Errorf("%d.step error", i)
	}
}

func AfterProcProcessor(info flow.Process) (bool, error) {
	if info.Name() == "" {
		panic("process name is empty")
	}
	if len(info.ID()) == 0 {
		panic("process id is empty")
	}
	if len(info.FlowID()) == 0 {
		panic("process flow id is empty")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..[Process: %s] AfterProcProcessor execute \n", info.Name())
	return true, nil
}

func GeneratePanicStep(i int, args ...any) func(ctx flow.Step) (any, error) {
	return func(ctx flow.Step) (any, error) {
		if len(args) > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Printf("%d.step panic \n", i)
		atomic.AddInt64(&current, 1)
		panic(fmt.Sprintf("%d.step panic", i))
	}
}

func TestMultipleExceptionStatus(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	workflow := flow.RegisterFlow("TestMultipleExceptionStatus")
	process := workflow.Process("TestMultipleExceptionStatus")
	process.CustomStep(Fn(t).Errors(), "1")
	process.CustomStep(Fn(t).Panic(), "2")
	step := process.CustomStep(Fn(t).WaitLetGO(1), "3")
	step.StepTimeout(time.Nanosecond)
	workflow.AfterStep(false, func(step flow.Step) (keepOn bool, err error) {
		e := step.Err()
		if e == nil {
			t.Errorf("step %s should have error", step.Name())
		} else {
			t.Logf("step %s error: %v", step.Name(), e)
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	})
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Timeout, flow.Error, flow.Panic))
	ff := flow.DoneFlow("TestMultipleExceptionStatus", nil)
	// DoneFlow return due to timeout, but process not complete
	atomic.StoreInt64(&letGo, 1)
	waitCurrent(7)
	if len(ff.Exceptions()) != 1 {
		t.Errorf("flow should have 1 exception, but %d", len(ff.Exceptions()))
	}
	for _, ps := range ff.Processes() {
		atomic.AddInt64(&current, 1)
		if len(ps.Exceptions()) != 3 {
			t.Errorf("process %s should have 3 exceptions, but %d", ps.Name(), len(ps.Exceptions()))
		}
	}
	if atomic.LoadInt64(&current) != 8 {
		t.Errorf("current should be 8, but %d", atomic.LoadInt64(&current))
	}
}

func TestSinglePanicStep(t *testing.T) {
	defer resetCurrent()
	t.Parallel()
	workflow := flow.RegisterFlow("TestSinglePanicStep")
	process := workflow.Process("TestSinglePanicStep")
	process.CustomStep(GeneratePanicStep(1), "1")
	flow.DoneFlow("TestSinglePanicStep", nil)
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Panic))
}

func TestGoAheadWithoutDependPanicStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestGoAheadWithoutDependPanicStep")
	process := workflow.Process("TestGoAheadWithoutDependPanicStep")
	process.CustomStep(GeneratePanicStep(1), "1")
	process.CustomStep(GenerateStep(2), "2", "1")
	process.CustomStep(GenerateStep(3), "3", "2")
	process.CustomStep(GenerateStep(11), "11")
	process.CustomStep(GenerateStep(12), "12", "11")
	process.CustomStep(GenerateStep(13), "13", "12")
	process.CustomStep(GenerateStep(14), "14", "13")
	workflow.AfterFlow(false, CheckResult(t, 5, flow.Panic))
	flow.DoneFlow("TestGoAheadWithoutDependPanicStep", nil)
}

func TestSingleErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleErrorStep")
	process := workflow.Process("TestSingleErrorStep")
	process.CustomStep(GenerateErrorStep(1), "1")
	flow.DoneFlow("TestSingleErrorStep", nil)
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Error))
}

func TestGoAheadWithoutDependErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestGoAheadWithoutDependErrorStep")
	process := workflow.Process("TestGoAheadWithoutDependErrorStep")
	process.CustomStep(GenerateErrorStep(1), "1")
	process.CustomStep(GenerateStep(2), "2", "1")
	process.CustomStep(GenerateStep(3), "3", "2")
	process.CustomStep(GenerateStep(11), "11")
	process.CustomStep(GenerateStep(12), "12", "11")
	process.CustomStep(GenerateStep(13), "13", "12")
	process.CustomStep(GenerateStep(14), "14", "13")
	workflow.AfterFlow(false, CheckResult(t, 5, flow.Error))
	flow.DoneFlow("TestGoAheadWithoutDependErrorStep", nil)
}

func TestSingleNormalStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleNormalStep")
	process := workflow.Process("TestSingleNormalStep")
	process.CustomStep(GenerateStep(1), "1")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestSingleNormalStep", nil)
}
func TestTestMultipleNormalStepWithoutAlias(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestTestMultipleNormalStepWithoutAlias")
	process := workflow.Process("TestTestMultipleNormalStepWithoutAlias")
	process.Follow(NormalStep1)
	process.Follow(NormalStep2)
	process.Follow(NormalStep3)
	process.CustomStep(NormalStep1, "4", NormalStep1)
	process.CustomStep(NormalStep1, "5", "NormalStep1")
	workflow.AfterFlow(false, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestTestMultipleNormalStepWithoutAlias", nil)
}

func TestMultipleNormalStepWithMultipleBranches(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleNormalStepWithMultipleBranches")
	process := workflow.Process("TestMultipleNormalStepWithMultipleBranches")
	process.CustomStep(GenerateStep(1), "1")
	process.CustomStep(GenerateStep(2), "2", "1")
	process.CustomStep(GenerateStep(3), "3", "2")
	process.CustomStep(GenerateStep(4), "4", "2")
	process.CustomStep(GenerateStep(5), "5", "2")
	process.SyncAll(GenerateStep(6), "6")
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestMultipleNormalStepWithMultipleBranches", nil)
}

func TestMultipleNormalStepsWithWaitBefore(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleNormalStepsWithWaitBefore")
	process := workflow.Process("TestMultipleNormalStepsWithWaitBefore")
	process.CustomStep(GenerateStep(1), "1").
		Next(GenerateStep(2), "2").
		Next(GenerateStep(3), "3")
	process.CustomStep(GenerateStep(11), "11").
		Next(GenerateStep(12), "12").
		Next(GenerateStep(13), "13")
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestMultipleNormalStepsWithWaitBefore", nil)
}

func TestMultipleNormalSteps(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleNormalSteps")
	process := workflow.Process("TestMultipleNormalSteps")
	process.CustomStep(GenerateStep(1), "1")
	process.CustomStep(GenerateStep(2), "2", "1")
	process.CustomStep(GenerateStep(3), "3", "2")
	process.CustomStep(GenerateStep(11), "11")
	process.CustomStep(GenerateStep(12), "12", "11")
	process.CustomStep(GenerateStep(13), "13", "12")
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestMultipleNormalSteps", nil)
}

func TestWorkFlowPause(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	workflow := flow.RegisterFlow("TestWorkFlowPause")
	process := workflow.Process("TestWorkFlowPause")
	process.CustomStep(Fn(t).WaitLetGO(), "1w1")
	process.CustomStep(Fn(t).WaitLetGO(), "1w2", "1w1")
	process.CustomStep(Fn(t).WaitLetGO(), "1w3", "1w2")
	process.CustomStep(Fn(t).WaitLetGO(), "2w1")
	process.CustomStep(Fn(t).WaitLetGO(), "2w2", "2w1")
	process.CustomStep(Fn(t).WaitLetGO(), "2w3", "2w2")
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	wf := flow.AsyncFlow("TestWorkFlowPause", nil)
	waitCurrent(2)
	rf := wf.Pause()
	rp, exist := rf.Process("TestWorkFlowPause")
	if !exist {
		t.Errorf("process[TestWorkFlowPause] not exist")
	}
	if !rp.Has(flow.Pause) {
		t.Errorf("process[TestWorkFlowPause] pause fail")
	}
	if !strings.Contains(strings.Join(rp.ExplainStatus(), ""), flow.Pause.String()) {
		t.Errorf("pause status explain error")
	}
	wf.Resume()
	atomic.StoreInt64(&letGo, 1)
	if rp.Has(flow.Pause) {
		t.Errorf("process[TestWorkFlowPause] resume fail")
	}
	wf.Done()
	waitCurrent(6)
}

func TestProcessPause(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	workflow := flow.RegisterFlow("TestProcessPause")
	process := workflow.Process("TestProcessPause")
	fn := Fn(t)
	process.CustomStep(fn.WaitLetGO(), "1w1")
	process.CustomStep(fn.Normal(), "1w2", "1w1")
	process.CustomStep(fn.Normal(), "1w3", "1w2")
	process.CustomStep(fn.WaitLetGO(), "2w1")
	process.CustomStep(fn.Normal(), "2w2", "2w1")
	process.CustomStep(fn.Normal(), "2w3", "2w2")
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	c := flow.AsyncFlow("TestProcessPause", nil)
	waitCurrent(2)
	cc, exist := c.Process("TestProcessPause")
	if !exist {
		t.Errorf("process[TestProcessPause] not exist")
	}
	cc.Pause()
	atomic.StoreInt64(&letGo, 1)
	time.Sleep(10 * time.Millisecond)
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("Prcess[%s] pause fail", cc.Name())
	}
	cc.Resume()
	c.Done()
	waitCurrent(6)
}

func TestCopyDepend(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestCopyDepend")
	process := workflow.Process("TestCopyDepend")
	process.CustomStep(GenerateStep(1), "1").
		Next(GenerateStep(2), "2").
		Same(GenerateStep(3), "3")
	process.CustomStep(GenerateStep(11), "11").
		Next(GenerateStep(12), "12").
		Same(GenerateStep(13), "13")
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestCopyDepend", nil)
}

func TestNewFlowReturn(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestNewFlowReturn")
	process := workflow.Process("TestNewFlowReturn")
	process.CustomStep(GenerateStep(1), "1")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNewFlowReturn", nil)
}

func TestPanicStepLook(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestPanicStepLook")
	process := workflow.Process("TestPanicStepLook")
	process.CustomStep(Fn(t).Panic(), "1")
	workflow.AfterStep(false, ErrorResultPrinter)
	flow.DoneFlow("TestPanicStepLook", nil)
}

func TestStop(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestStop")
	process := workflow.Process("TestStop")
	process.CustomStep(Fn(t).WaitLetGO(), "1")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "3", "2")
	af := flow.AsyncFlow("TestStop", nil)
	waitCurrent(1)
	atomic.StoreInt64(&letGo, 1)
	af.Stop()
	ff := af.Done()
	CheckResult(t, 1, flow.Stop)(any(ff).(flow.WorkFlow))

	resetCurrent()
	workflow = flow.RegisterFlow("TestStop0")
	process = workflow.Process("TestStop0")
	process.CustomStep(Fn(t).WaitLetGO(), "1")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "3", "2")
	af = flow.AsyncFlow("TestStop0", nil)
	waitCurrent(1)
	fp, _ := af.Process("TestStop0")
	fp.Stop()
	ff = af.Done()
	CheckResult(t, 1, flow.Stop)(any(ff).(flow.WorkFlow))
}

func TestSerial(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSerial0")
	process := workflow.Process("TestSerial0")
	process.Follow(NormalStep1, NormalStep2, NormalStep3)
	process.AfterStep(true, CheckCurrent(t, 1)).OnlyFor("NormalStep1")
	process.AfterStep(true, CheckCurrent(t, 2)).OnlyFor("NormalStep2")
	process.AfterStep(true, CheckCurrent(t, 3)).OnlyFor("NormalStep3")
	workflow.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestSerial0", nil)

	resetCurrent()
	workflow = flow.RegisterFlow("TestSerial1")
	process = workflow.Process("TestSerial1")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "01")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "02")
	process.Follow(NormalStep1)
	process.Follow(NormalStep2, NormalStep3).After("01", "02", NormalStep1)
	process.AfterStep(true, CheckCurrent(t, 4)).OnlyFor("NormalStep2")
	process.AfterStep(true, CheckCurrent(t, 5)).OnlyFor("NormalStep3")
	workflow.AfterFlow(true, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestSerial1", nil)

	resetCurrent()
	workflow = flow.RegisterFlow("TestSerial2")
	process = workflow.Process("TestSerial2")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "01")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "02")
	process.Follow(NormalStep1)
	process.Follow(NormalStep2, NormalStep3).After("01", "02", NormalStep1)
	process.SyncAll(Fx[flow.Step](t).Inc().Step(), "6")
	workflow.AfterFlow(true, CheckResult(t, 6, flow.Success))

	workflow = flow.RegisterFlow("TestSerial3")
	process = workflow.Process("TestSerial3")
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("TestSerial3 should panic")
		}
	}()
	process.Follow()
}

func TestParallel(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestParallel0")
	process := workflow.Process("TestParallel0")
	process.Parallel(NormalStep1, NormalStep2, NormalStep3)
	workflow.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestParallel0", nil)

	resetCurrent()
	workflow = flow.RegisterFlow("TestParallel1")
	process = workflow.Process("TestParallel1")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "01")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "02")
	process.Follow(NormalStep1)
	process.Parallel(NormalStep2, NormalStep3).After("01", "02", NormalStep1)
	workflow.AfterFlow(true, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestParallel1", nil)

	resetCurrent()
	workflow = flow.RegisterFlow("TestParallel2")
	process = workflow.Process("TestParallel2")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "01")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "02")
	process.Follow(NormalStep1)
	process.Parallel(NormalStep2, NormalStep3).After("01", "02", NormalStep1)
	process.SyncAll(Fx[flow.Step](t).Inc().Step(), "6")
	workflow.AfterFlow(true, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestParallel2", nil)

	workflow = flow.RegisterFlow("TestParallel3")
	process = workflow.Process("TestParallel3")
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("TestParallel3 should panic")
		}
	}()
	process.Parallel()
}

func TestSyncPoint(t *testing.T) {
	defer resetCurrent()
	process := flow.FlowWithProcess("TestSyncPoint")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "NormalStep0")
	point := process.Parallel(NormalStep1, NormalStep2, NormalStep3).After("NormalStep0")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "NormalStep4", point)
	process.AfterStep(true, CheckCurrent(t, 5)).OnlyFor("NormalStep5")
	process.Flow().AfterFlow(true, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestSyncPoint", nil)

	resetCurrent()
	process = flow.FlowWithProcess("TestSyncPoint1")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "NormalStep0")
	point = process.Parallel(NormalStep1, NormalStep2).After("NormalStep0")
	point = process.Parallel(NormalStep3, NormalStep4).After(point)
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "NormalStep5", point)
	process.AfterStep(true, CheckCurrent(t, 6)).OnlyFor("NormalStep6")
	process.Flow().AfterFlow(true, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestSyncPoint1", nil)

	resetCurrent()
	process = flow.FlowWithProcess("TestSyncPoint2")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "NormalStep0")
	point = process.Parallel(NormalStep1, NormalStep2).After("NormalStep0")
	point = process.Parallel(NormalStep3, NormalStep4).After(point, NormalStep1, NormalStep2)
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "NormalStep5", point, NormalStep3, NormalStep4)
	process.AfterStep(true, CheckCurrent(t, 6)).OnlyFor("NormalStep6")
	process.Flow().AfterFlow(true, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestSyncPoint2", nil)

	resetCurrent()
	process = flow.FlowWithProcess("TestSyncPoint3")
	point = process.Parallel(NormalStep1, NormalStep2)
	point = process.Parallel(NormalStep3, NormalStep4).After(point, NormalStep1, NormalStep2)
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "NormalStep5", point, NormalStep3, NormalStep4)
	process.AfterStep(true, CheckCurrent(t, 5)).OnlyFor("NormalStep6")
	process.Flow().AfterFlow(true, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestSyncPoint3", nil)
}
