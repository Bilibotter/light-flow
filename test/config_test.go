package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func StepInfoChecker(key string, prev, next []string) func(info flow.Step) (bool, error) {
	return func(info flow.Step) (bool, error) {
		if info.Name() != key {
			return true, nil
		}
		println("matched", key)
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func PreProcessor(info flow.Step) (bool, error) {
	if len(info.ID()) == 0 {
		panic("step id is empty")
	}
	if len(info.ProcessID()) == 0 {
		panic("step process id is empty")
	}
	if len(info.FlowID()) == 0 {
		panic("step flow id is empty")
	}
	if info.Name() == "" {
		panic("step name is empty")
	}
	if info.StartTime().IsZero() {
		panic("step start time is zero")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..step[%s] PreProcessor exeucte\n", info.Name())
	return true, nil
}

func PostProcessor(info flow.Step) (bool, error) {
	if len(info.ID()) == 0 {
		panic("step id is empty")
	}
	if len(info.ProcessID()) == 0 {
		panic("step process id is empty")
	}
	if len(info.FlowID()) == 0 {
		panic("step flow id is empty")
	}
	if info.Name() == "" {
		panic("step name is empty")
	}
	if info.StartTime().IsZero() {
		panic("step start time is zero")
	}
	if info.EndTime().IsZero() {
		panic("step end time is zero")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..step[%s] PostProcessor execute\n", info.Name())
	return true, nil
}

func ProcProcessor(info flow.Process) (bool, error) {
	if info.Name() == "" {
		panic("process name is empty")
	}
	if len(info.ID()) == 0 {
		panic("process id is empty")
	}
	if len(info.FlowID()) == 0 {
		panic("process flow id is empty")
	}
	if info.StartTime().IsZero() {
		panic("process start time is zero")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..process[%s] ProcProcessor execute \n", info.Name())
	return true, nil
}

func PanicProcProcessor(info flow.Process) (bool, error) {
	atomic.AddInt64(&current, 1)
	panic("PanicProcProcessor panic")
	return true, nil
}

func CheckStepCurrent(i int) func(info flow.Step) (bool, error) {
	return func(info flow.Step) (bool, error) {
		if atomic.LoadInt64(&current) != int64(i) {
			panic(fmt.Sprintf("current number not equal check number,check number=%d", i))
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func CheckProcCurrent(i int) func(info flow.Process) (bool, error) {
	return func(info flow.Process) (bool, error) {
		if atomic.LoadInt64(&current) != int64(i) {
			panic(fmt.Sprintf("current number not equal check number,check number=%d", i))
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func PanicStepProcessor(info flow.Step) (bool, error) {
	atomic.AddInt64(&current, 1)
	panic("PanicStepProcessor panic")
	return true, nil
}

func TestProcessorWrongOrder(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcessorWrongOrder")
	process := workflow.Process("TestProcessorWrongOrder")
	process.NameStep(GenerateStep(1), "1")
	process.BeforeStep(false, CheckStepCurrent(3))
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("process not panic with false callback order")
		}
	}()
	process.BeforeStep(true, CheckStepCurrent(0))
}

func TestProcessorOrder1(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcessorOrder1")
	process := workflow.Process("TestProcessorOrder1")
	process.NameStep(GenerateStep(1), "1")
	process.BeforeProcess(true, CheckProcCurrent(0))
	process.BeforeProcess(false, CheckProcCurrent(1))
	process.BeforeStep(true, CheckStepCurrent(2))
	process.BeforeStep(false, CheckStepCurrent(3))
	process.AfterStep(true, CheckStepCurrent(5))
	process.AfterStep(false, CheckStepCurrent(6))
	process.AfterProcess(true, CheckProcCurrent(7))
	process.AfterProcess(false, CheckProcCurrent(8))
	workflow.AfterFlow(false, CheckResult(t, 9, flow.Success))
	flow.DoneFlow("TestProcessorOrder1", nil)
}

func TestNonEssentialProcProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestNonEssentialProcProcessorPanic1")
	process := workflow.Process("TestNonEssentialProcProcessorPanic1")
	process.NameStep(GenerateStep(1), "1")
	process.BeforeProcess(false, PanicProcProcessor)
	workflow.AfterFlow(false, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestNonEssentialProcProcessorPanic1", nil)
	resetCurrent()
	workflow = flow.RegisterFlow("TestNonEssentialProcProcessorPanic2")
	process = workflow.Process("TestNonEssentialProcProcessorPanic2")
	process.NameStep(GenerateStep(1), "1")
	process.AfterProcess(false, PanicProcProcessor)
	workflow.AfterFlow(false, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestNonEssentialProcProcessorPanic2", nil)
}

func TestEssentialProcProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestEssentialProcProcessorPanic1")
	process := workflow.Process("TestEssentialProcProcessorPanic1")
	process.NameStep(GenerateStep(1), "1")
	process.BeforeProcess(true, PanicProcProcessor)
	workflow.AfterFlow(false, CheckResult(t, 1, flow.CallbackFail))
	flow.DoneFlow("TestEssentialProcProcessorPanic1", nil)
	resetCurrent()

	workflow = flow.RegisterFlow("TestEssentialProcProcessorPanic2")
	process = workflow.Process("TestEssentialProcProcessorPanic2")
	process.NameStep(GenerateStep(1), "1")
	process.AfterProcess(true, PanicProcProcessor)
	workflow.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail))
	flow.DoneFlow("TestEssentialProcProcessorPanic2", nil)
}

func TestNonEssentialStepProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestNonEssentialStepProcessorPanic1")
	process := workflow.Process("TestNonEssentialStepProcessorPanic1")
	process.NameStep(GenerateStep(1), "1")
	process.BeforeStep(false, PanicStepProcessor)
	workflow.AfterFlow(false, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestNonEssentialStepProcessorPanic1", nil)
	resetCurrent()

	workflow = flow.RegisterFlow("TestNonEssentialStepProcessorPanic2")
	process = workflow.Process("TestNonEssentialStepProcessorPanic2")
	process.NameStep(GenerateStep(1), "1")
	process.AfterStep(false, PanicStepProcessor)
	workflow.AfterFlow(false, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestNonEssentialStepProcessorPanic2", nil)
}

func TestEssentialStepProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestEssentialStepProcessorPanic1")
	process := workflow.Process("TestEssentialStepProcessorPanic1")
	process.NameStep(GenerateStep(1), "1")
	process.BeforeStep(true, PanicStepProcessor)
	workflow.AfterFlow(false, CheckResult(t, 1, flow.CallbackFail))
	flow.DoneFlow("TestEssentialStepProcessorPanic1", nil)
	resetCurrent()

	workflow = flow.RegisterFlow("TestEssentialStepProcessorPanic2")
	process = workflow.Process("TestEssentialStepProcessorPanic2")
	process.NameStep(GenerateStep(1), "1")
	process.AfterStep(true, PanicStepProcessor)
	workflow.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail))
	flow.DoneFlow("TestEssentialStepProcessorPanic2", nil)
}

func TestProcessorWhenExceptionOccur(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcessorWhenExceptionOccur")
	process := workflow.Process("TestProcessorWhenExceptionOccur")
	process.BeforeStep(true, PreProcessor)
	process.AfterStep(true, PostProcessor)
	process.BeforeProcess(true, ProcProcessor)
	process.AfterProcess(true, ProcProcessor)
	process.NameStep(GenerateErrorStep(1), "1")
	process.NameStep(GeneratePanicStep(2), "2")
	step := process.NameStep(GenerateErrorStep(3, "ms"), "3")
	step.StepTimeout(time.Nanosecond)
	workflow.AfterFlow(false, CheckResult(t, 9, flow.Timeout, flow.Error, flow.Panic))
	flow.DoneFlow("TestProcessorWhenExceptionOccur", nil)
	// DoneFlow return due to timeout, but step not complete
	start := time.Now()
	for atomic.LoadInt64(&current) != 11 && time.Now().Sub(start) < 200*time.Millisecond {
		runtime.Gosched()
	}
}

func TestPreAndPostProcessor(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestPreAndPostProcessor")
	process := workflow.Process("TestPreAndPostProcessor")
	process.NameStep(GenerateStep(1), "1")
	process.NameStep(GenerateStep(2), "2", "1")
	process.NameStep(GenerateStep(3), "3", "2")
	process.NameStep(GenerateStep(4), "4", "3")
	process.BeforeStep(true, PreProcessor)
	process.AfterStep(true, PostProcessor)
	process.BeforeProcess(true, ProcProcessor)
	process.AfterProcess(true, ProcProcessor)
	workflow.AfterFlow(false, CheckResult(t, 14, flow.Success))
	flow.DoneFlow("TestPreAndPostProcessor", nil)
}

func TestWithLongProcessTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithLongProcessTimeout")

	process := workflow.Process("TestWithLongProcessTimeout")
	process.ProcessTimeout(1 * time.Second)
	process.NameStep(GenerateStep(1), "1")
	process.NameStep(GenerateStep(2), "2", "1")
	process.NameStep(GenerateStep(3), "3", "2")
	process.NameStep(GenerateStep(4), "4", "3")
	workflow.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestWithLongProcessTimeout", nil)
}

func TestWithShortProcessTimeout(t *testing.T) {
	defer resetCurrent()
	letGo = false
	workflow := flow.RegisterFlow("TestWithShortProcessTimeout")
	process := workflow.Process("TestWithShortProcessTimeout")
	process.ProcessTimeout(1 * time.Nanosecond)
	fn := Fn(t)
	process.NameStep(fn.All(fn.WaitLetGO(), fn.Errors()), "1")
	process.NameStep(fn.Normal(), "2", "1")
	process.NameStep(fn.Normal(), "3", "2")
	process.NameStep(fn.Normal(), "4", "3")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Timeout))
	c := flow.AsyncFlow("TestWithShortProcessTimeout", nil)
	waitCurrent(1)
	c.Done()
	letGo = true
	waitCurrent(2)
	time.Sleep(10 * time.Millisecond)
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

func TestParallelWithLongDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestParallelWithLongDefaultStepTimeout")
	process := workflow.Process("TestParallelWithLongDefaultStepTimeout")
	process.StepTimeout(300 * time.Millisecond)
	process.StepRetry(3)
	process.NameStep(GenerateStep(1), "1")
	process.NameStep(GenerateStep(2), "2")
	process.NameStep(GenerateStep(3), "3")
	process.NameStep(GenerateStep(4), "4")
	workflow.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestParallelWithLongDefaultStepTimeout", nil)
}

func TestWithLongDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithLongDefaultStepTimeout")
	process := workflow.Process("TestWithLongDefaultStepTimeout")
	process.StepTimeout(1 * time.Second)
	process.StepRetry(3)
	process.NameStep(GenerateStep(1), "1")
	process.NameStep(GenerateStep(2), "2", "1")
	process.NameStep(GenerateStep(3), "3", "2")
	process.NameStep(GenerateStep(4), "4", "3")
	workflow.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestWithLongDefaultStepTimeout", nil)
}

func TestWithShortDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	letGo = false
	workflow := flow.RegisterFlow("TestWithShortDefaultStepTimeout")
	process := workflow.Process("TestWithShortDefaultStepTimeout")
	process.StepTimeout(1 * time.Nanosecond)
	process.StepRetry(3)
	fn := Fn(t)
	process.NameStep(fn.All(fn.WaitLetGO(), fn.Errors()), "1")
	process.NameStep(fn.Normal(), "2", "1")
	process.NameStep(fn.Normal(), "3", "2")
	process.NameStep(fn.Normal(), "4", "3")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Timeout))
	c := flow.AsyncFlow("TestWithShortDefaultStepTimeout", nil)
	waitCurrent(1)
	c.Done()
	letGo = true
	waitCurrent(8)
	time.Sleep(10 * time.Millisecond)
	if atomic.LoadInt64(&current) != 8 {
		t.Errorf("execute 8 step, but current = %d", current)
	}
}

func TestAddStepTimeoutAndRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestAddStepTimeoutAndRetry")
	proc := workflow.Process("TestAddStepTimeoutAndRetry")
	proc.StepRetry(3).StepTimeout(100 * time.Millisecond)
	proc.NameStep(GenerateStep(1), "1")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestAddStepTimeoutAndRetry", nil)
}

func TestWithLongStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithLongStepTimeout")
	process := workflow.Process("TestWithLongStepTimeout")
	process.NameStep(GenerateStep(1), "1").StepRetry(3).StepTimeout(1 * time.Second)
	process.NameStep(GenerateStep(2), "2", "1")
	process.NameStep(GenerateStep(3), "3", "2")
	process.NameStep(GenerateStep(4), "4", "3")
	workflow.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestWithLongStepTimeout", nil)
}

func TestWithShortStepTimeout(t *testing.T) {
	defer resetCurrent()
	letGo = false
	workflow := flow.RegisterFlow("TestWithShortStepTimeout")
	process := workflow.Process("TestWithShortStepTimeout")
	fn := Fn(t)
	process.NameStep(fn.All(fn.WaitLetGO(), fn.Errors()), "1").StepRetry(3).StepTimeout(1 * time.Nanosecond)
	process.NameStep(fn.Normal(), "2", "1")
	process.NameStep(fn.Normal(), "3", "2")
	process.NameStep(fn.Normal(), "4", "3")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Timeout))
	c := flow.AsyncFlow("TestWithShortStepTimeout", nil)
	waitCurrent(1)
	c.Done()
	letGo = true
	waitCurrent(8)
	time.Sleep(10 * time.Millisecond)
	if atomic.LoadInt64(&current) != 8 {
		t.Errorf("execute 8 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithProcessRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleErrorStepWithProcessRetry")
	process := workflow.Process("TestSingleErrorStepWithProcessRetry")
	process.StepRetry(2)
	process.NameStep(GenerateErrorStep(1), "1")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Error))
	flow.DoneFlow("TestSingleErrorStepWithProcessRetry", nil)
}

func TestSingleErrorStepWithStepRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleErrorStepWithStepRetry")
	process := workflow.Process("TestSingleErrorStepWithStepRetry")
	process.NameStep(GenerateErrorStep(1), "1").StepRetry(2)
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Error))
	flow.DoneFlow("TestSingleErrorStepWithStepRetry", nil)
}

func TestSingleErrorStepWithProcessAndStepRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleErrorStepWithProcessAndStepRetry")
	process := workflow.Process("TestSingleErrorStepWithProcessAndStepRetry")
	process.StepRetry(2)
	process.NameStep(GenerateErrorStep(1), "1").StepRetry(1)
	workflow.AfterFlow(false, CheckResult(t, 2, flow.Error))
	flow.DoneFlow("TestSingleErrorStepWithProcessAndStepRetry", nil)
}
