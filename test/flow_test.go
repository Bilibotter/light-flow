package test

import (
	"fmt"
	flow "gitee.com/MetaphysicCoding/light-flow"
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

func resetCurrent() {
	atomic.StoreInt64(&current, 0)
}

func NormalStep3(ctx flow.Context) (any, error) {
	atomic.AddInt64(&current, 1)
	fmt.Printf("3.normal step finish\n")
	return 3, nil
}

func NormalStep2(ctx flow.Context) (any, error) {
	atomic.AddInt64(&current, 1)
	fmt.Printf("2.normal step finish\n")
	return 2, nil
}

func NormalStep1(ctx flow.Context) (any, error) {
	atomic.AddInt64(&current, 1)
	fmt.Printf("1.normal step finish\n")
	return 1, nil
}

func GenerateStep(i int, args ...any) func(ctx flow.Context) (any, error) {
	return func(ctx flow.Context) (any, error) {
		if len(args) > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Printf("%d.step finish\n", i)
		atomic.AddInt64(&current, 1)
		return i, nil
	}
}

func GenerateErrorStep(i int, args ...any) func(ctx flow.Context) (any, error) {
	return func(ctx flow.Context) (any, error) {
		if len(args) > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Printf("%d.step error occur \n", i)
		atomic.AddInt64(&current, 1)
		return i, fmt.Errorf("%d.step error", i)
	}
}

func AfterProcProcessor(info *flow.ProcessInfo) (bool, error) {
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
	fmt.Printf("..process[%s] AfterProcProcessor execute \n", info.Name)
	return true, nil
}

func GeneratePanicStep(i int, args ...any) func(ctx flow.Context) (any, error) {
	return func(ctx flow.Context) (any, error) {
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
	process.AliasStep("1", GenerateErrorStep(1, "ms"))
	process.AliasStep("2", GeneratePanicStep(2, "ms"))
	step := process.AliasStep("3", GenerateErrorStep(3, "ms"))
	step.Timeout(time.Millisecond)
	result := flow.DoneFlow("TestMultipleExceptionStatus", nil)
	// DoneFlow return due to timeout, but process not complete
	time.Sleep(100 * time.Millisecond)
	if result.Success() {
		t.Errorf("process[%s] success, but expected failed", result.GetName())
	} else {
		features := result.Fails()
		if len(features) != 1 {
			t.Errorf("process[%s] not found in failed feature", result.GetName())
			return
		}
		feature := features[0]
		if features[0].GetName() != "TestMultipleExceptionStatus" {
			t.Errorf("process[%s] not found in failed feature", result.GetName())
			return
		}
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.GetName())
		}
		explain := feature.ExplainStatus()
		if !feature.Contain(flow.Timeout) {
			t.Errorf("process[%s] timeout, but explain not contain, explain=%v", feature.GetName(), explain)
		}
		if !feature.Contain(flow.Error) {
			t.Errorf("process[%s] error, but explain not contain, but explain=%v", feature.GetName(), explain)
		}
		if !feature.Contain(flow.Panic) {
			t.Errorf("process[%s] panic, but explain not contain, but explain=%v", feature.GetName(), explain)
		}
	}
	if atomic.LoadInt64(&current) != 3 {
		t.Errorf("execute 3 step, but current = %d", current)
	}
}

func TestSinglePanicStep(t *testing.T) {
	defer resetCurrent()
	t.Parallel()
	workflow := flow.RegisterFlow("TestSinglePanicStep")
	process := workflow.Process("TestSinglePanicStep")
	process.AliasStep("1", GeneratePanicStep(1))
	features := flow.DoneFlow("TestSinglePanicStep", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
}

func TestGoAheadWithoutDependPanicStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestGoAheadWithoutDependPanicStep")
	process := workflow.Process("TestGoAheadWithoutDependPanicStep")
	process.AliasStep("1", GeneratePanicStep(1))
	process.AliasStep("-1", GenerateStep(-1), "1")
	process.AliasStep("-2", GenerateStep(-2), "-1")
	process.AliasStep("11", GenerateStep(11))
	process.AliasStep("12", GenerateStep(12), "11")
	process.AliasStep("13", GenerateStep(13), "12")
	process.AliasStep("14", GenerateStep(14), "13")
	features := flow.DoneFlow("TestGoAheadWithoutDependPanicStep", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
	time.Sleep(100 * time.Millisecond)
}

func TestSingleErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleErrorStep")
	process := workflow.Process("TestSingleErrorStep")
	process.AliasStep("1", GenerateErrorStep(1))
	features := flow.DoneFlow("TestSingleErrorStep", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
}

func TestGoAheadWithoutDependErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestGoAheadWithoutDependErrorStep")
	process := workflow.Process("TestGoAheadWithoutDependErrorStep")
	process.AliasStep("1", GenerateErrorStep(1))
	process.AliasStep("-1", GenerateStep(-1), "1")
	process.AliasStep("-2", GenerateStep(-2), "-1")
	process.AliasStep("11", GenerateStep(11))
	process.AliasStep("12", GenerateStep(12), "11")
	process.AliasStep("13", GenerateStep(13), "12")
	process.AliasStep("14", GenerateStep(14), "13")
	features := flow.DoneFlow("TestGoAheadWithoutDependErrorStep", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
}

func TestSingleNormalStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleNormalStep")
	process := workflow.Process("TestSingleNormalStep")
	process.AliasStep("1", GenerateStep(1))
	features := flow.DoneFlow("TestSingleNormalStep", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
}
func TestTestMultipleNormalStepWithoutAlias(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestTestMultipleNormalStepWithoutAlias")
	process := workflow.Process("TestTestMultipleNormalStepWithoutAlias")
	process.Step(NormalStep1)
	process.Step(NormalStep2)
	process.Step(NormalStep3)
	process.AliasStep("4", NormalStep1, NormalStep1)
	process.AliasStep("5", NormalStep1, "NormalStep1")
	features := flow.DoneFlow("TestTestMultipleNormalStepWithoutAlias", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
}

func TestMultipleNormalStepWithMultipleBranches(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleNormalStepWithMultipleBranches")
	process := workflow.Process("TestMultipleNormalStepWithMultipleBranches")
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "2")
	process.AliasStep("4", GenerateStep(4), "2")
	process.AliasStep("5", GenerateStep(5), "2")
	process.Tail("6", GenerateStep(6))
	features := flow.DoneFlow("TestMultipleNormalStepWithMultipleBranches", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
}

func TestMultipleNormalStepsWithWaitBefore(t *testing.T) {
	defer resetCurrent()
	factory := flow.RegisterFlow("TestMultipleNormalStepsWithWaitBefore")
	process := factory.Process("TestMultipleNormalStepsWithWaitBefore")
	process.AliasStep("1", GenerateStep(1)).
		Next(GenerateStep(2), "2").
		Next(GenerateStep(3), "3")
	process.AliasStep("11", GenerateStep(11)).
		Next(GenerateStep(12), "12").
		Next(GenerateStep(13), "13")
	features := flow.DoneFlow("TestMultipleNormalStepsWithWaitBefore", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
}

func TestMultipleNormalSteps(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleNormalSteps")
	process := workflow.Process("TestMultipleNormalSteps")
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "2")
	process.AliasStep("11", GenerateStep(11))
	process.AliasStep("12", GenerateStep(12), "11")
	process.AliasStep("13", GenerateStep(13), "12")
	features := flow.DoneFlow("TestMultipleNormalSteps", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
}

func TestWorkFlowPause(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWorkFlowPause")
	process := workflow.Process("TestWorkFlowPause")
	process.AliasStep("1", GenerateStep(1, "ms"))
	process.AliasStep("2", GenerateStep(2, "ms"), "1")
	process.AliasStep("3", GenerateStep(3, "ms"), "2")
	process.AliasStep("11", GenerateStep(11, "ms"))
	process.AliasStep("12", GenerateStep(12, "ms"), "11")
	process.AliasStep("13", GenerateStep(13, "ms"), "12")
	wf := flow.AsyncFlow("TestWorkFlowPause", nil)
	time.Sleep(10 * time.Millisecond)
	wf.Pause()
	time.Sleep(200 * time.Millisecond)
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
	for _, feature := range wf.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Contain(flow.Pause) {
			t.Errorf("process[%s] pause fail", feature.Name)
		}
	}
	wf.Resume()
	wf.Done()
	for _, feature := range wf.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
}

func TestArgs(t *testing.T) {
	factory := flow.RegisterFlow("TestArgs")
	process := factory.Process("TestArgs")
	process.AfterStep(true, ErrorResultPrinter)
	process.AliasStep("1", func(ctx flow.Context) (any, error) {
		a, ok := ctx.Get("InputA")
		if !ok {
			panic("InputA not found")
		}
		aa, ok := a.(InputA)
		if !ok {
			panic("InputA type error")
		}
		if aa.Name != "a" {
			panic("InputA value error")
		}

		b, ok := ctx.Get("*InputB")
		if !ok {
			panic("InputB not found")
		}
		bb, ok := b.(*InputB)
		if !ok {
			panic("InputB type error")
		}
		if bb.Name != "b" {
			panic("InputB value error")
		}
		return nil, nil
	})

	features := flow.DoneArgs("TestArgs", InputA{Name: "a"}, &InputB{Name: "b"})
	for _, feature := range features.Futures() {
		if !feature.Success() {
			t.Errorf("process[%s] fail, exception=%v", feature.Name, feature.ExplainStatus())
		}
	}
}

func TestProcessPause(t *testing.T) {
	defer resetCurrent()
	factory := flow.RegisterFlow("TestProcessPause")
	process := factory.Process("TestProcessPause")
	process.AliasStep("1", GenerateStep(1, "ms"))
	process.AliasStep("2", GenerateStep(2, "ms"), "1")
	process.AliasStep("3", GenerateStep(3, "ms"), "2")
	process.AliasStep("11", GenerateStep(11, "ms"))
	process.AliasStep("12", GenerateStep(12, "ms"), "11")
	process.AliasStep("13", GenerateStep(13, "ms"), "12")
	workflow := flow.AsyncFlow("TestProcessPause", nil)
	time.Sleep(10 * time.Millisecond)
	workflow.ProcessController("TestProcessPause").Pause()
	time.Sleep(200 * time.Millisecond)
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
	for _, feature := range workflow.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Contain(flow.Pause) {
			t.Errorf("process[%s] pause fail", feature.Name)
		}
	}
	workflow.ProcessController("TestProcessPause").Resume()
	workflow.Done()
	for _, feature := range workflow.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
}

func TestCopyDepend(t *testing.T) {
	defer resetCurrent()
	factory := flow.RegisterFlow("TestCopyDepend")
	process := factory.Process("TestCopyDepend")
	process.AliasStep("1", GenerateStep(1)).
		Next(GenerateStep(2), "2").
		Same(GenerateStep(3), "3")
	process.AliasStep("11", GenerateStep(11)).
		Next(GenerateStep(12), "12").
		Same(GenerateStep(13), "13")
	features := flow.DoneFlow("TestCopyDepend", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
}

func TestNewFlowReturn(t *testing.T) {
	defer resetCurrent()
	factory := flow.RegisterFlow("TestNewFlowReturn")
	process := factory.Process("TestNewFlowReturn")
	process.AliasStep("1", GenerateStep(1))
	result := flow.DoneFlow("TestNewFlowReturn", nil)
	if !result.Success() {
		t.Errorf("flow fail")
	}
	if len(result.Exceptions()) > 0 {
		t.Errorf("flow fail, exception=%v", result.Exceptions())
	}
	for _, feature := range result.Futures() {
		feature.ExplainStatus()
		if !feature.Normal() {
			t.Errorf("process[%s] fail, explain=%v", feature.Name, feature.ExplainStatus())
		}
		if len(feature.Exceptions()) > 0 {
			t.Errorf("process[%s] fail, explain=%v", feature.Name, feature.Exceptions())
		}
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
		fmt.Printf("process[%s] explain=%v\n", feature.Name, feature.ExplainStatus())
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
}
