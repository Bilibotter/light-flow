package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var (
	current int64
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

func CheckResult(t *testing.T, check int64, statuses ...*flow.StatusEnum) func(flow.WorkFlow) (keepOn bool, err error) {
	return func(workFlow flow.WorkFlow) (keepOn bool, err error) {
		ss := make([]string, len(statuses))
		for i, status := range statuses {
			ss[i] = status.Message()
		}
		t.Logf("start check, expected current=%d, status include %s", check, strings.Join(ss, ","))
		if current != check {
			t.Errorf("execute %d step, but current = %d\n", check, current)
		}
		for _, status := range statuses {
			if status == flow.Success && !workFlow.Success() {
				t.Errorf("workFlow not success\n")
			}
			//if status == flow.Timeout {
			//	time.Sleep(50 * time.Millisecond)
			//}
			if !workFlow.Has(status) {
				t.Errorf("workFlow has not %s status\n", status.Message())
			}
		}
		t.Logf("status expalin=%s", strings.Join(workFlow.ExplainStatus(), ","))
		t.Logf("finish check")
		println()
		return true, nil
	}
}

func resetCurrent() {
	atomic.StoreInt64(&current, 0)
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
	if len(info.Id()) == 0 {
		panic("process id is empty")
	}
	if len(info.FlowId()) == 0 {
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
	workflow := flow.RegisterFlow("TestMultipleExceptionStatus")
	process := workflow.Process("TestMultipleExceptionStatus")
	process.NameStep(GenerateErrorStep(1), "1")
	process.NameStep(GeneratePanicStep(2), "2")
	step := process.NameStep(GenerateErrorStep(3, "ms"), "3")
	step.Timeout(time.Nanosecond)
	workflow.AfterFlow(false, CheckResult(t, 2, flow.Timeout, flow.Error, flow.Panic))
	flow.DoneFlow("TestMultipleExceptionStatus", nil)
	// DoneFlow return due to timeout, but process not complete
	time.Sleep(110 * time.Millisecond)
	if current != 3 {
		t.Errorf("execute 3 step, but current = %d\n", current)
	}
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
//func TestWorkFlowPause(t *testing.T) {
//	defer resetCurrent()
//	workflow := flow.RegisterFlow("TestWorkFlowPause")
//	process := workflow.Process("TestWorkFlowPause")
//	process.NameStep(GenerateStep(1, "ms"), "1")
//	process.NameStep(GenerateStep(2, "ms"), "2", "1")
//	process.NameStep(GenerateStep(3, "ms"), "3", "2")
//	process.NameStep(GenerateStep(11, "ms"), "11")
//	process.NameStep(GenerateStep(12, "ms"), "12", "11")
//	process.NameStep(GenerateStep(13, "ms"), "13", "12")
//	wf := flow.AsyncFlow("TestWorkFlowPause", nil)
//	time.Sleep(10 * time.Millisecond)
//	wf.Pause()
//	time.Sleep(200 * time.Millisecond)
//	if atomic.LoadInt64(&current) != 2 {
//		t.Errorf("execute 2 step, but current = %d", current)
//	}
//	for _, feature := range wf.Futures() {
//		explain := strings.Join(feature.ExplainStatus(), ", ")
//		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
//		if !feature.Has(flow.Pause) {
//			t.Errorf("process[%s] pause fail", feature.Name)
//		}
//	}
//	wf.Resume()
//	wf.Done()
//	for _, feature := range wf.Futures() {
//		explain := strings.Join(feature.ExplainStatus(), ", ")
//		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
//		if !feature.Success() {
//			t.Errorf("process[%s] fail", feature.Name)
//		}
//	}
//	if atomic.LoadInt64(&current) != 6 {
//		t.Errorf("execute 6 step, but current = %d", current)
//	}
//}

//func TestArgs(t *testing.T) {
//	workflow := flow.RegisterFlow("TestArgs")
//	process := workflow.Process("TestArgs")
//	process.AfterStep(true, ErrorResultPrinter)
//	process.NameStep(func(ctx flow.Step) (any, error) {
//		a, ok := ctx.Get("InputA")
//		if !ok {
//			panic("InputA not found")
//		}
//		aa, ok := a.(InputA)
//		if !ok {
//			panic("InputA type error")
//		}
//		if aa.Name != "a" {
//			panic("InputA value error")
//		}
//
//		b, ok := ctx.Get("*InputB")
//		if !ok {
//			panic("InputB not found")
//		}
//		bb, ok := b.(*InputB)
//		if !ok {
//			panic("InputB type error")
//		}
//		if bb.Name != "b" {
//			panic("InputB value error")
//		}
//		return nil, nil
//	}, "1")
//	workflow.AfterFlow(false, CheckResult(t, 0, flow.Success))
//	flow.DoneArgs("TestArgs", InputA{Name: "a"}, &InputB{Name: "b"})
//}
//
//func TestProcessPause(t *testing.T) {
//	defer resetCurrent()
//	factory := flow.RegisterFlow("TestProcessPause")
//	process := factory.Process("TestProcessPause")
//	process.NameStep(GenerateStep(1, "ms"), "1")
//	process.NameStep(GenerateStep(2, "ms"), "2", "1")
//	process.NameStep(GenerateStep(3, "ms"), "3", "2")
//	process.NameStep(GenerateStep(11, "ms"), "11")
//	process.NameStep(GenerateStep(12, "ms"), "12", "11")
//	process.NameStep(GenerateStep(13, "ms"), "13", "12")
//	workflow := flow.AsyncFlow("TestProcessPause", nil)
//	time.Sleep(10 * time.Millisecond)
//	workflow.ProcessController("TestProcessPause").Pause()
//	time.Sleep(200 * time.Millisecond)
//	if atomic.LoadInt64(&current) != 2 {
//		t.Errorf("execute 2 step, but current = %d", current)
//	}
//	for _, feature := range workflow.Futures() {
//		explain := strings.Join(feature.ExplainStatus(), ", ")
//		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
//		if !feature.Has(flow.Pause) {
//			t.Errorf("process[%s] pause fail", feature.Name)
//		}
//	}
//	workflow.ProcessController("TestProcessPause").Resume()
//	workflow.Done()
//	for _, feature := range workflow.Futures() {
//		explain := strings.Join(feature.ExplainStatus(), ", ")
//		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
//		if !feature.Success() {
//			t.Errorf("process[%s] fail", feature.Name)
//		}
//	}
//	if atomic.LoadInt64(&current) != 6 {
//		t.Errorf("execute 6 step, but current = %d", current)
//	}
//}

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
