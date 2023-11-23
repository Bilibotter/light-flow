package test

import (
	"fmt"
	flow "gitee.com/MetaphysicCoding/light-flow"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func StepInfoChecker(key string, prev, next []string) func(info *flow.StepInfo) (bool, error) {
	return func(info *flow.StepInfo) (bool, error) {
		if info.Name != key {
			return true, nil
		}
		println("matched", key)
		atomic.AddInt64(&current, 1)
		for name := range info.Prev {
			s := flow.CreateSetBySliceFunc(prev, func(s string) string { return s })
			if !s.Contains(name) {
				panic(fmt.Sprintf("step[%s] prev not contains %s", key, name))
			}
		}
		for name := range info.Next {
			s := flow.CreateSetBySliceFunc(next, func(s string) string { return s })
			if !s.Contains(name) {
				panic(fmt.Sprintf("step[%s] next not contains %s", key, name))
			}
		}
		return true, nil
	}
}

func PreProcessor(info *flow.StepInfo) (bool, error) {
	if len(info.Id) == 0 {
		panic("step id is empty")
	}
	if len(info.ProcessId) == 0 {
		panic("step process id is empty")
	}
	if len(info.FlowId) == 0 {
		panic("step flow id is empty")
	}
	if info.Name == "" {
		panic("step name is empty")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..step[%s] PreProcessor exeucte\n", info.Name)
	return true, nil
}

func PostProcessor(info *flow.StepInfo) (bool, error) {
	if len(info.Id) == 0 {
		panic("step id is empty")
	}
	if len(info.ProcessId) == 0 {
		panic("step process id is empty")
	}
	if len(info.FlowId) == 0 {
		panic("step flow id is empty")
	}
	if info.Name == "" {
		panic("step name is empty")
	}
	if info.Start.IsZero() {
		panic("step start time is zero")
	}
	if info.End.IsZero() {
		panic("step end time is zero")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..step[%s] PostProcessor execute\n", info.Name)
	return true, nil
}
func ErrorResultPrinter(info *flow.StepInfo) (bool, error) {
	if !info.Normal() {
		fmt.Printf("step[%s] error, explain=%v, err=%v\n", info.GetName(), info.ExplainStatus(), info.Err)
	}
	return true, nil
}

func ProcProcessor(info *flow.ProcessInfo) (bool, error) {
	if info.Name == "" {
		panic("process name is empty")
	}
	if len(info.Id) == 0 {
		panic("process id is empty")
	}
	if len(info.FlowId) == 0 {
		panic("process flow id is empty")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..process[%s] ProcProcessor execute \n", info.Name)
	return true, nil
}

func PanicProcProcessor(info *flow.ProcessInfo) (bool, error) {
	atomic.AddInt64(&current, 1)
	panic("PanicProcProcessor panic")
	return true, nil
}

func CheckStepCurrent(i int) func(info *flow.StepInfo) (bool, error) {
	return func(info *flow.StepInfo) (bool, error) {
		if atomic.LoadInt64(&current) != int64(i) {
			panic(fmt.Sprintf("current number not equal check number,check number=%d", i))
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func CheckProcCurrent(i int) func(info *flow.ProcessInfo) (bool, error) {
	return func(info *flow.ProcessInfo) (bool, error) {
		if atomic.LoadInt64(&current) != int64(i) {
			panic(fmt.Sprintf("current number not equal check number,check number=%d", i))
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func PanicStepProcessor(info *flow.StepInfo) (bool, error) {
	atomic.AddInt64(&current, 1)
	panic("PanicStepProcessor panic")
	return true, nil
}

func TestProcessorRandomOrder(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcessorRandomOrder")
	process := workflow.Process("TestProcessorRandomOrder")
	process.AliasStep("1", GenerateStep(1))
	process.BeforeStep(false, CheckStepCurrent(3))
	process.BeforeStep(true, CheckStepCurrent(0))
	process.BeforeStep(true, CheckStepCurrent(1))
	process.BeforeStep(false, CheckStepCurrent(4))
	process.BeforeStep(false, CheckStepCurrent(5))
	process.BeforeStep(true, CheckStepCurrent(2))
	process.BeforeStep(false, CheckStepCurrent(6))
	features := flow.DoneFlow("TestProcessorRandomOrder", nil)
	for _, feature := range features.Futures() {
		explain := feature.ExplainStatus()
		if !feature.Success() {
			t.Errorf("process[%s] failed,explian=%v", feature.Name, explain)
		}
		if feature.Contain(flow.Panic) {
			t.Errorf("process[%s] not panic, but explain contain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 8 {
		t.Errorf("execute 8 step, but current = %d", current)
	}
}

func TestProcessorOrder2(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcessorOrder2")
	process := workflow.Process("TestProcessorOrder2")
	process.AliasStep("1", GenerateStep(1))
	process.BeforeProcess(false, CheckProcCurrent(1))
	process.BeforeProcess(true, CheckProcCurrent(0))
	process.BeforeStep(false, CheckStepCurrent(3))
	process.BeforeStep(true, CheckStepCurrent(2))
	process.AfterStep(false, CheckStepCurrent(6))
	process.AfterStep(true, CheckStepCurrent(5))
	process.AfterProcess(false, CheckProcCurrent(8))
	process.AfterProcess(true, CheckProcCurrent(7))
	features := flow.DoneFlow("TestProcessorOrder2", nil)
	for _, feature := range features.Futures() {
		explain := feature.ExplainStatus()
		if !feature.Success() {
			t.Errorf("process[%s] failed,explian=%v", feature.Name, explain)
		}
		if feature.Contain(flow.Panic) {
			t.Errorf("process[%s] not panic, but explain contain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 9 {
		t.Errorf("execute 9 step, but current = %d", current)
	}
}

func TestProcessorOrder1(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcessorOrder1")
	process := workflow.Process("TestProcessorOrder1")
	process.AliasStep("1", GenerateStep(1))
	process.BeforeProcess(true, CheckProcCurrent(0))
	process.BeforeProcess(false, CheckProcCurrent(1))
	process.BeforeStep(true, CheckStepCurrent(2))
	process.BeforeStep(false, CheckStepCurrent(3))
	process.AfterStep(true, CheckStepCurrent(5))
	process.AfterStep(false, CheckStepCurrent(6))
	process.AfterProcess(true, CheckProcCurrent(7))
	process.AfterProcess(false, CheckProcCurrent(8))
	features := flow.DoneFlow("TestProcessorOrder1", nil)
	for _, feature := range features.Futures() {
		explain := feature.ExplainStatus()
		if !feature.Success() {
			t.Errorf("process[%s] failed,explian=%v", feature.Name, explain)
		}
		if feature.Contain(flow.Panic) {
			t.Errorf("process[%s] not panic, but explain contain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 9 {
		t.Errorf("execute 9 step, but current = %d", current)
	}
}

func TestNonEssentialProcProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestNonEssentialProcProcessorPanic1")
	process := workflow.Process("TestNonEssentialProcProcessorPanic1")
	process.AliasStep("1", GenerateStep(1))
	process.BeforeProcess(false, PanicProcProcessor)
	features := flow.DoneFlow("TestNonEssentialProcProcessorPanic1", nil)
	for _, feature := range features.Futures() {
		explain := feature.ExplainStatus()
		if !feature.Success() {
			t.Errorf("process[%s] failed,explian=%v", feature.Name, explain)
		}
		if feature.Contain(flow.Panic) {
			t.Errorf("process[%s] not panic, but explain contain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
	resetCurrent()
	workflow = flow.RegisterFlow("TestNonEssentialProcProcessorPanic2")
	process = workflow.Process("TestNonEssentialProcProcessorPanic2")
	process.AliasStep("1", GenerateStep(1))
	process.AfterProcess(false, PanicProcProcessor)
	features = flow.DoneFlow("TestNonEssentialProcProcessorPanic2", nil)
	for _, feature := range features.Futures() {
		explain := feature.ExplainStatus()
		if !feature.Success() {
			t.Errorf("process[%s] failed,explian=%v", feature.Name, explain)
		}
		if feature.Contain(flow.Panic) {
			t.Errorf("process[%s] not panic, but explain not contain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

func TestEssentialProcProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestEssentialProcProcessorPanic1")
	process := workflow.Process("TestEssentialProcProcessorPanic1")
	process.AliasStep("1", GenerateStep(1))
	process.BeforeProcess(true, PanicProcProcessor)
	features := flow.DoneFlow("TestEssentialProcProcessorPanic1", nil)
	for _, feature := range features.Futures() {
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
		explain := feature.ExplainStatus()
		if !feature.Contain(flow.Panic) {
			t.Errorf("process[%s] panic, but explain not cotain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
	resetCurrent()
	workflow = flow.RegisterFlow("TestEssentialProcProcessorPanic2")
	process = workflow.Process("TestEssentialProcProcessorPanic2")
	process.AliasStep("1", GenerateStep(1))
	process.AfterProcess(true, PanicProcProcessor)
	features = flow.DoneFlow("TestEssentialProcProcessorPanic2", nil)
	for _, feature := range features.Futures() {
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
		explain := feature.ExplainStatus()
		if !feature.Contain(flow.Panic) {
			t.Errorf("process[%s] panic, but explain not cotain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

func TestNonEssentialStepProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestNonEssentialStepProcessorPanic1")
	process := workflow.Process("TestNonEssentialStepProcessorPanic1")
	process.AliasStep("1", GenerateStep(1))
	process.BeforeStep(false, PanicStepProcessor)
	features := flow.DoneFlow("TestNonEssentialStepProcessorPanic1", nil)
	for _, feature := range features.Futures() {
		explain := feature.ExplainStatus()
		if !feature.Success() {
			t.Errorf("process[%s] failed,explian=%v", feature.Name, explain)
		}
		if feature.Contain(flow.Panic) {
			t.Errorf("process[%s] not panic, but explain contain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
	resetCurrent()
	workflow = flow.RegisterFlow("TestNonEssentialStepProcessorPanic2")
	process = workflow.Process("TestNonEssentialStepProcessorPanic2")
	process.AliasStep("1", GenerateStep(1))
	process.AfterStep(false, PanicStepProcessor)
	features = flow.DoneFlow("TestNonEssentialStepProcessorPanic2", nil)
	for _, feature := range features.Futures() {
		explain := feature.ExplainStatus()
		if !feature.Success() {
			t.Errorf("process[%s] failed,explian=%v", feature.Name, explain)
		}
		if feature.Contain(flow.Panic) {
			t.Errorf("process[%s] not panic, but explain not contain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

func TestEssentialStepProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestEssentialStepProcessorPanic1")
	process := workflow.Process("TestEssentialStepProcessorPanic1")
	process.AliasStep("1", GenerateStep(1))
	process.BeforeStep(true, PanicStepProcessor)
	features := flow.DoneFlow("TestEssentialStepProcessorPanic1", nil)
	for _, feature := range features.Futures() {
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
		explain := feature.ExplainStatus()
		if !feature.Contain(flow.Panic) {
			t.Errorf("process[%s] panic, but explain not cotain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
	resetCurrent()
	workflow = flow.RegisterFlow("TestEssentialStepProcessorPanic2")
	process = workflow.Process("TestEssentialStepProcessorPanic2")
	process.AliasStep("1", GenerateStep(1))
	process.AfterStep(true, PanicStepProcessor)
	features = flow.DoneFlow("TestEssentialStepProcessorPanic2", nil)
	for _, feature := range features.Futures() {
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
		explain := feature.ExplainStatus()
		if !feature.Contain(flow.Panic) {
			t.Errorf("process[%s] panic, but explain not cotain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

func TestProcessorWhenExceptionOccur(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcessorWhenExceptionOccur")
	process := workflow.Process("TestProcessorWhenExceptionOccur")
	process.BeforeStep(true, PreProcessor)
	process.AfterStep(true, PostProcessor)
	process.BeforeProcess(true, ProcProcessor)
	process.AfterProcess(true, ProcProcessor)
	process.AliasStep("1", GenerateErrorStep(1, "ms"))
	process.AliasStep("2", GeneratePanicStep(2, "ms"))
	step := process.AliasStep("3", GenerateErrorStep(3, "ms"))
	step.Timeout(time.Millisecond)
	features := flow.DoneFlow("TestProcessorWhenExceptionOccur", nil)
	for _, feature := range features.Futures() {
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
		explain := feature.ExplainStatus()
		if !feature.Contain(flow.Timeout) {
			t.Errorf("process[%s] timeout, but explain not cotain, explain=%v", feature.Name, explain)
		}
		if !feature.Contain(flow.Error) {
			t.Errorf("process[%s] error, but explain not cotain, but explain=%v", feature.Name, explain)
		}
		if !feature.Contain(flow.Panic) {
			t.Errorf("process[%s] panic, but explain not cotain, but explain=%v", feature.Name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 11 {
		t.Errorf("execute 11 step, but current = %d", current)
	}
}

func TestPreAndPostProcessor(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestPreAndPostProcessor")
	process := workflow.Process("TestPreAndPostProcessor")
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "2")
	process.AliasStep("4", GenerateStep(4), "3")
	process.BeforeStep(true, PreProcessor)
	process.AfterStep(true, PostProcessor)
	process.BeforeProcess(true, ProcProcessor)
	process.AfterProcess(true, ProcProcessor)
	features := flow.DoneFlow("TestPreAndPostProcessor", nil)
	for _, feature := range features.Futures() {
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 14 {
		t.Errorf("execute 14 step, but current = %d", current)
	}
}

func TestWithLongProcessTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithLongProcessTimeout")

	process := workflow.Process("TestWithLongProcessTimeout")
	process.ProcessTimeout(1 * time.Second)
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "2")
	process.AliasStep("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestWithLongProcessTimeout", nil)
	for _, feature := range features.Futures() {
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestWithShortProcessTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithShortProcessTimeout")
	process := workflow.Process("TestWithShortProcessTimeout")
	process.ProcessTimeout(1 * time.Millisecond)
	process.AliasStep("1", GenerateStep(1, "ms"))
	process.AliasStep("2", GenerateStep(2, "ms"), "1")
	process.AliasStep("3", GenerateStep(3, "ms"), "2")
	process.AliasStep("4", GenerateStep(4, "ms"), "3")
	features := flow.DoneFlow("TestWithShortProcessTimeout", nil)
	for _, feature := range features.Futures() {
		if feature.Success() {
			t.Errorf("process[%s] success with timeout", feature.Name)
		}
	}
	time.Sleep(400 * time.Millisecond)
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
}

func TestParallelWithLongDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestParallelWithLongDefaultStepTimeout")
	process := workflow.Process("TestParallelWithLongDefaultStepTimeout")
	process.StepsTimeout(300 * time.Millisecond)
	process.StepsRetry(3)
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2))
	process.AliasStep("3", GenerateStep(3))
	process.AliasStep("4", GenerateStep(4))
	features := flow.DoneFlow("TestParallelWithLongDefaultStepTimeout", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestWithLongDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithLongDefaultStepTimeout")
	process := workflow.Process("TestWithLongDefaultStepTimeout")
	process.StepsTimeout(1 * time.Second)
	process.StepsRetry(3)
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "2")
	process.AliasStep("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestWithLongDefaultStepTimeout", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestWithShortDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithShortDefaultStepTimeout")
	process := workflow.Process("TestWithShortDefaultStepTimeout")
	process.StepsTimeout(1 * time.Millisecond)
	process.StepsRetry(3)
	process.AliasStep("1", GenerateStep(1, "ms"))
	process.AliasStep("2", GenerateStep(2, "ms"), "1")
	process.AliasStep("3", GenerateStep(3, "ms"), "2")
	process.AliasStep("4", GenerateStep(4, "ms"), "3")
	features := flow.DoneFlow("TestWithShortDefaultStepTimeout", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success with timeout", feature.Name)
		}
	}
	time.Sleep(400 * time.Millisecond)
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
}

func TestAddStepTimeoutAndRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestAddStepTimeoutAndRetry")
	proc := workflow.Process("TestAddStepTimeoutAndRetry")
	proc.StepsRetry(3).StepsTimeout(100 * time.Millisecond)
	proc.AliasStep("1", GenerateStep(1))
	result := flow.DoneFlow("TestAddStepTimeoutAndRetry", nil)
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
	if !result.Success() {
		t.Errorf("workflow[%s] fail, exceptions=%v", result.GetName(), result.Exceptions())
		for _, feature := range result.Futures() {
			if !feature.Success() {
				t.Errorf("process[%s] fail, exceptions=%v", feature.GetName(), feature.Exceptions())
			}
		}
	} else {
		t.Logf("workflow[%s] success", result.GetName())
	}
}

func TestWithLongStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithLongStepTimeout")
	process := workflow.Process("TestWithLongStepTimeout")
	process.AliasStep("1", GenerateStep(1)).Retry(3).Timeout(1 * time.Second)
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "2")
	process.AliasStep("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestWithLongStepTimeout", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] failed", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestWithShortStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithShortStepTimeout")
	process := workflow.Process("TestWithShortStepTimeout")
	process.AliasStep("1", GenerateStep(1, "ms")).Retry(3).Timeout(1 * time.Millisecond)
	process.AliasStep("2", GenerateStep(2, "ms"), "1")
	process.AliasStep("3", GenerateStep(3, "ms"), "2")
	process.AliasStep("4", GenerateStep(4, "ms"), "3")
	features := flow.DoneFlow("TestWithShortStepTimeout", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success with timeout", feature.Name)
		}
	}
	time.Sleep(300 * time.Millisecond)
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithProcessRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleErrorStepWithProcessRetry")
	process := workflow.Process("TestSingleErrorStepWithProcessRetry")
	process.StepsRetry(3)
	process.AliasStep("1", GenerateErrorStep(1))
	features := flow.DoneFlow("TestSingleErrorStepWithProcessRetry", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 3 {
		t.Errorf("execute 3 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithStepRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleErrorStepWithStepRetry")
	process := workflow.Process("TestSingleErrorStepWithStepRetry")
	process.AliasStep("1", GenerateErrorStep(1)).Retry(3)
	features := flow.DoneFlow("TestSingleErrorStepWithStepRetry", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 3 {
		t.Errorf("execute 3 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithProcessAndStepRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleErrorStepWithProcessAndStepRetry")
	process := workflow.Process("TestSingleErrorStepWithProcessAndStepRetry")
	process.StepsRetry(3)
	process.AliasStep("1", GenerateErrorStep(1)).Retry(2)
	features := flow.DoneFlow("TestSingleErrorStepWithProcessAndStepRetry", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

//func TestRecoverSerialStep(t *testing.T) {
//	defer resetCurrent()
//	workflow := flow.RegisterFlow("TestRecoverSerialStep")
//	process := workflow.Process("TestRecoverSerialStep", nil)
//	config := flow.stepConfig{stepRetry: 3, stepTimeout: 1 * time.Second}
//	process.AliasStep("1", GenerateStep(1)).Config(&config)
//	process.AliasStep("2", GenerateStep(2), "1")
//	process.AliasStep("3", GenerateStep(3), "2")
//	process.AliasStep("4", GenerateStep(4), "3")
//	process.AliasStep("5", GenerateStep(5), "4")
//	wf := flow.buildRunFlow("TestRecoverSerialStep", nil)
//	if err := wf.SkipFinishedStep("1", nil); err != nil {
//		panic(err.Error())
//	}
//	if err := wf.SkipFinishedStep("2", nil); err != nil {
//		panic(err.Error())
//	}
//	features := wf.Done()
//	for _, feature := range features {
//		explain := strings.Join(feature.ExplainStatus(), ", ")
//		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
//		if !feature.Success() {
//			t.Errorf("process[%s] failed", feature.Name)
//		}
//	}
//	if atomic.LoadInt64(&current) != 3 {
//		t.Errorf("execute 3 step, but current = %d", current)
//	}
//}
//
//func TestRecoverParallelStep(t *testing.T) {
//	defer resetCurrent()
//	workflow := flow.RegisterFlow("TestRecoverParallelStep")
//	process := workflow.Process("TestRecoverParallelStep", nil)
//	process.AliasStep("1", GenerateStep(1))
//	process.AliasStep("11", GenerateStep(11), "1")
//	process.AliasStep("2", GenerateStep(2))
//	process.AliasStep("12", GenerateStep(12), "2")
//	process.AliasStep("3", GenerateStep(3))
//	process.AliasStep("13", GenerateStep(13), "3")
//	process.AliasStep("4", GenerateStep(4))
//	process.AliasStep("14", GenerateStep(14), "4")
//	wf := flow.buildRunFlow("TestRecoverParallelStep", nil)
//	if err := wf.SkipFinishedStep("1", nil); err != nil {
//		panic(err.Error())
//	}
//	if err := wf.SkipFinishedStep("2", nil); err != nil {
//		panic(err.Error())
//	}
//	if err := wf.SkipFinishedStep("3", nil); err != nil {
//		panic(err.Error())
//	}
//	if err := wf.SkipFinishedStep("4", nil); err != nil {
//		panic(err.Error())
//	}
//	features := wf.Done()
//	for _, feature := range features {
//		explain := strings.Join(feature.ExplainStatus(), ", ")
//		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
//		if !feature.Success() {
//			t.Errorf("process[%s] fail", feature.Name)
//		}
//	}
//	if atomic.LoadInt64(&current) != 4 {
//		t.Errorf("execute 4 step, but current = %d", current)
//	}
//}

//func TestRecoverAndWaitAll(t *testing.T) {
//	defer resetCurrent()
//	workflow := flow.RegisterFlow("TestRecoverAndWaitAll")
//	process := workflow.Process("TestRecoverAndWaitAll", nil)
//	process.AliasStep("1", GenerateStep(1))
//	process.AliasStep("2", GenerateStep(2))
//	process.AliasStep("3", GenerateStep(3))
//	process.AliasStep("4", GenerateStep(4))
//	process.Tail("5", GenerateStep(5))
//	wf := flow.buildRunFlow("TestRecoverAndWaitAll", nil)
//	if err := wf.SkipFinishedStep("1", nil); err != nil {
//		panic(err.Error())
//	}
//	if err := wf.SkipFinishedStep("2", nil); err != nil {
//		panic(err.Error())
//	}
//	if err := wf.SkipFinishedStep("3", nil); err != nil {
//		panic(err.Error())
//	}
//	if err := wf.SkipFinishedStep("4", nil); err != nil {
//		panic(err.Error())
//	}
//	features := wf.Done()
//	for _, feature := range features {
//		explain := strings.Join(feature.ExplainStatus(), ", ")
//		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
//		if !feature.Success() {
//			t.Errorf("process[%s] fail", feature.Name)
//		}
//	}
//	if atomic.LoadInt64(&current) != 1 {
//		t.Errorf("execute 1 step, but current = %d", current)
//	}
//}
