package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"sync/atomic"
	"testing"
	"time"
)

type InputA struct {
	Name string
}

type InputB struct {
	Name string
}

func ErrorResultPrinter(info flow.Step) (bool, error) {
	if !info.Success() {
		fmt.Printf("step[%s] error, explain=%v, err=%v\n", info.Name(), info.ExplainStatus(), info.Err())
	}
	return true, nil
}

func NormalStep3(ctx flow.Step) (any, error) {
	atomic.AddInt64(&current, 1)
	fmt.Printf("3.normal step finish\n")
	return 3, nil
}

func NormalStep2(ctx flow.Step) (any, error) {
	atomic.AddInt64(&current, 1)
	fmt.Printf("2.normal step finish\n")
	return 2, nil
}

func NormalStep1(ctx flow.Step) (any, error) {
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
	fmt.Printf("..process[%s] AfterProcProcessor execute \n", info.Name())
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
	letGo = false
	workflow := flow.RegisterFlow("TestMultipleExceptionStatus")
	process := workflow.Process("TestMultipleExceptionStatus")
	process.NameStep(Fn(t).Errors(), "1")
	process.NameStep(Fn(t).Panic(), "2")
	step := process.NameStep(Fn(t).WaitLetGO(1), "3")
	step.Timeout(time.Nanosecond)
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Timeout, flow.Error, flow.Panic))
	flow.DoneFlow("TestMultipleExceptionStatus", nil)
	// DoneFlow return due to timeout, but process not complete
	letGo = true
	waitCurrent(4)
}

func TestSinglePanicStep(t *testing.T) {
	defer resetCurrent()
	t.Parallel()
	workflow := flow.RegisterFlow("TestSinglePanicStep")
	process := workflow.Process("TestSinglePanicStep")
	process.NameStep(GeneratePanicStep(1), "1")
	flow.DoneFlow("TestSinglePanicStep", nil)
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Panic))
}

func TestGoAheadWithoutDependPanicStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestGoAheadWithoutDependPanicStep")
	process := workflow.Process("TestGoAheadWithoutDependPanicStep")
	process.NameStep(GeneratePanicStep(1), "1")
	process.NameStep(GenerateStep(-1), "-1", "1")
	process.NameStep(GenerateStep(-2), "-2", "-1")
	process.NameStep(GenerateStep(11), "11")
	process.NameStep(GenerateStep(12), "12", "11")
	process.NameStep(GenerateStep(13), "13", "12")
	process.NameStep(GenerateStep(14), "14", "13")
	workflow.AfterFlow(false, CheckResult(t, 5, flow.Panic))
	flow.DoneFlow("TestGoAheadWithoutDependPanicStep", nil)
}

func TestSingleErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleErrorStep")
	process := workflow.Process("TestSingleErrorStep")
	process.NameStep(GenerateErrorStep(1), "1")
	flow.DoneFlow("TestSingleErrorStep", nil)
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Error))
}

func TestGoAheadWithoutDependErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestGoAheadWithoutDependErrorStep")
	process := workflow.Process("TestGoAheadWithoutDependErrorStep")
	process.NameStep(GenerateErrorStep(1), "1")
	process.NameStep(GenerateStep(-1), "-1", "1")
	process.NameStep(GenerateStep(-2), "-2", "-1")
	process.NameStep(GenerateStep(11), "11")
	process.NameStep(GenerateStep(12), "12", "11")
	process.NameStep(GenerateStep(13), "13", "12")
	process.NameStep(GenerateStep(14), "14", "13")
	workflow.AfterFlow(false, CheckResult(t, 5, flow.Error))
	flow.DoneFlow("TestGoAheadWithoutDependErrorStep", nil)
}

func TestSingleNormalStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleNormalStep")
	process := workflow.Process("TestSingleNormalStep")
	process.NameStep(GenerateStep(1), "1")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestSingleNormalStep", nil)
}
func TestTestMultipleNormalStepWithoutAlias(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestTestMultipleNormalStepWithoutAlias")
	process := workflow.Process("TestTestMultipleNormalStepWithoutAlias")
	process.Step(NormalStep1)
	process.Step(NormalStep2)
	process.Step(NormalStep3)
	process.NameStep(NormalStep1, "4", NormalStep1)
	process.NameStep(NormalStep1, "5", "NormalStep1")
	workflow.AfterFlow(false, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestTestMultipleNormalStepWithoutAlias", nil)
}

func TestMultipleNormalStepWithMultipleBranches(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleNormalStepWithMultipleBranches")
	process := workflow.Process("TestMultipleNormalStepWithMultipleBranches")
	process.NameStep(GenerateStep(1), "1")
	process.NameStep(GenerateStep(2), "2", "1")
	process.NameStep(GenerateStep(3), "3", "2")
	process.NameStep(GenerateStep(4), "4", "2")
	process.NameStep(GenerateStep(5), "5", "2")
	process.Tail(GenerateStep(6), "6")
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestMultipleNormalStepWithMultipleBranches", nil)
}

func TestMultipleNormalStepsWithWaitBefore(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleNormalStepsWithWaitBefore")
	process := workflow.Process("TestMultipleNormalStepsWithWaitBefore")
	process.NameStep(GenerateStep(1), "1").
		Next(GenerateStep(2), "2").
		Next(GenerateStep(3), "3")
	process.NameStep(GenerateStep(11), "11").
		Next(GenerateStep(12), "12").
		Next(GenerateStep(13), "13")
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestMultipleNormalStepsWithWaitBefore", nil)
}

func TestMultipleNormalSteps(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleNormalSteps")
	process := workflow.Process("TestMultipleNormalSteps")
	process.NameStep(GenerateStep(1), "1")
	process.NameStep(GenerateStep(2), "2", "1")
	process.NameStep(GenerateStep(3), "3", "2")
	process.NameStep(GenerateStep(11), "11")
	process.NameStep(GenerateStep(12), "12", "11")
	process.NameStep(GenerateStep(13), "13", "12")
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestMultipleNormalSteps", nil)
}

// todo add it
func TestWorkFlowPause(t *testing.T) {
	defer resetCurrent()
	letGo = false
	workflow := flow.RegisterFlow("TestWorkFlowPause")
	process := workflow.Process("TestWorkFlowPause")
	process.NameStep(Fn(t).WaitLetGO(), "1-1")
	process.NameStep(Fn(t).WaitLetGO(), "1-2", "1-1")
	process.NameStep(Fn(t).WaitLetGO(), "1-3", "1-2")
	process.NameStep(Fn(t).WaitLetGO(), "2-1")
	process.NameStep(Fn(t).WaitLetGO(), "2-2", "2-1")
	process.NameStep(Fn(t).WaitLetGO(), "2-3", "2-2")
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
	wf.Resume()
	letGo = true
	if rp.Has(flow.Pause) {
		t.Errorf("process[TestWorkFlowPause] resume fail")
	}
	wf.Done()
	waitCurrent(6)
}

func TestProcessPause(t *testing.T) {
	defer resetCurrent()
	letGo = false
	workflow := flow.RegisterFlow("TestProcessPause")
	process := workflow.Process("TestProcessPause")
	fn := Fn(t)
	process.NameStep(fn.WaitLetGO(), "1-1")
	process.NameStep(fn.Normal(), "1-2", "1-1")
	process.NameStep(fn.Normal(), "1-3", "1-2")
	process.NameStep(fn.WaitLetGO(), "2-1")
	process.NameStep(fn.Normal(), "2-2", "2-1")
	process.NameStep(fn.Normal(), "2-3", "2-2")
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	c := flow.AsyncFlow("TestProcessPause", nil)
	waitCurrent(2)
	cc, exist := c.Process("TestProcessPause")
	if !exist {
		t.Errorf("process[TestProcessPause] not exist")
	}
	cc.Pause()
	letGo = true
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
	process.NameStep(GenerateStep(1), "1").
		Next(GenerateStep(2), "2").
		Same(GenerateStep(3), "3")
	process.NameStep(GenerateStep(11), "11").
		Next(GenerateStep(12), "12").
		Same(GenerateStep(13), "13")
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestCopyDepend", nil)
}

func TestNewFlowReturn(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestNewFlowReturn")
	process := workflow.Process("TestNewFlowReturn")
	process.NameStep(GenerateStep(1), "1")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNewFlowReturn", nil)
}

func TestPanicStepLook(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestPanicStepLook")
	process := workflow.Process("TestPanicStepLook")
	process.NameStep(Fn(t).Panic(), "1")
	workflow.AfterStep(false, ErrorResultPrinter)
	flow.DoneFlow("TestPanicStepLook", nil)
}
