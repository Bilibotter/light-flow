package test

import (
	"fmt"
	flow "gitee.com/MetaphysicCoding/light-flow"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var (
	current int64
)

func resetCurrent() {
	atomic.StoreInt64(&current, 0)
}

func NormalStep3(ctx *flow.Context) (any, error) {
	atomic.AddInt64(&current, 1)
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("3.normal step finish\n")
	return 3, nil
}

func NormalStep2(ctx *flow.Context) (any, error) {
	atomic.AddInt64(&current, 1)
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("2.normal step finish\n")
	return 2, nil
}

func NormalStep1(ctx *flow.Context) (any, error) {
	atomic.AddInt64(&current, 1)
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("1.normal step finish\n")
	return 1, nil
}

func GenerateStep(i int) func(ctx *flow.Context) (any, error) {
	return func(ctx *flow.Context) (any, error) {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("%d.step finish\n", i)
		atomic.AddInt64(&current, 1)
		return i, nil
	}
}

func GenerateErrorStep(i int) func(ctx *flow.Context) (any, error) {
	return func(ctx *flow.Context) (any, error) {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("%d.step error occur \n", i)
		atomic.AddInt64(&current, 1)
		return i, fmt.Errorf("%d.step error", i)
	}
}

func GeneratePanicStep(i int) func(ctx *flow.Context) (any, error) {
	return func(ctx *flow.Context) (any, error) {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("%d.step panic \n", i)
		atomic.AddInt64(&current, 1)
		panic(fmt.Sprintf("%d.step panic", i))
	}
}

func TestMultipleExceptionStatus(t *testing.T) {
	defer resetCurrent()
	workflow := flow.AddFlowFactory("TestMultipleExceptionStatus")
	process := workflow.AddProcess("TestMultipleExceptionStatus", nil)
	process.AddStepWithAlias("1", GenerateErrorStep(1))
	process.AddStepWithAlias("2", GeneratePanicStep(2))
	step := process.AddStepWithAlias("3", GenerateErrorStep(3))
	step.AddConfig(&flow.StepConfig{Timeout: time.Millisecond})
	features := flow.DoneFlow("TestMultipleExceptionStatus", nil)
	for name, feature := range features {
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
		explain := feature.ExplainStatus()
		if !slices.Contains(explain, "Timeout") {
			t.Errorf("process[%s] timeout, but explain not cotain, explain=%v", name, explain)
		}
		if !slices.Contains(explain, "Error") {
			t.Errorf("process[%s] error, but explain not cotain, but explain=%v", name, explain)
		}
		if !slices.Contains(explain, "Panic") {
			t.Errorf("process[%s] panic, but explain not cotain, but explain=%v", name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 3 {
		t.Errorf("excute 3 step, but current = %d", current)
	}
}

func TestSinglePanicStep(t *testing.T) {
	defer resetCurrent()
	t.Parallel()
	workflow := flow.AddFlowFactory("TestSinglePanicStep")
	process := workflow.AddProcess("TestSinglePanicStep", nil)
	process.AddStepWithAlias("1", GeneratePanicStep(1))
	features := flow.DoneFlow("TestSinglePanicStep", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestGoAheadWithoutDependPanicStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.AddFlowFactory("TestGoAheadWithoutDependPanicStep")
	process := workflow.AddProcess("TestGoAheadWithoutDependPanicStep", nil)
	process.AddStepWithAlias("1", GeneratePanicStep(1))
	process.AddStepWithAlias("-1", GenerateStep(-1), "1")
	process.AddStepWithAlias("-2", GenerateStep(-2), "-1")
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddStepWithAlias("12", GenerateStep(12), "11")
	process.AddStepWithAlias("13", GenerateStep(13), "12")
	process.AddStepWithAlias("14", GenerateStep(14), "13")
	features := flow.DoneFlow("TestGoAheadWithoutDependPanicStep", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("excute 5 step, but current = %d", current)
	}
}

func TestSingleErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.AddFlowFactory("TestSingleErrorStep")
	process := workflow.AddProcess("TestSingleErrorStep", nil)
	process.AddStepWithAlias("1", GenerateErrorStep(1))
	features := flow.DoneFlow("TestSingleErrorStep", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestGoAheadWithoutDependErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.AddFlowFactory("TestGoAheadWithoutDependErrorStep")
	process := workflow.AddProcess("TestGoAheadWithoutDependErrorStep", nil)
	process.AddStepWithAlias("1", GenerateErrorStep(1))
	process.AddStepWithAlias("-1", GenerateStep(-1), "1")
	process.AddStepWithAlias("-2", GenerateStep(-2), "-1")
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddStepWithAlias("12", GenerateStep(12), "11")
	process.AddStepWithAlias("13", GenerateStep(13), "12")
	process.AddStepWithAlias("14", GenerateStep(14), "13")
	features := flow.DoneFlow("TestGoAheadWithoutDependErrorStep", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("excute 5 step, but current = %d", current)
	}
}

func TestSingleNormalStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.AddFlowFactory("TestSingleNormalStep")
	process := workflow.AddProcess("TestSingleNormalStep", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	features := flow.DoneFlow("TestSingleNormalStep", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}
func TestTestMultipleNormalStepWithoutAlias(t *testing.T) {
	defer resetCurrent()
	workflow := flow.AddFlowFactory("TestTestMultipleNormalStepWithoutAlias")
	process := workflow.AddProcess("TestTestMultipleNormalStepWithoutAlias", nil)
	process.AddStep(NormalStep1)
	process.AddStep(NormalStep2)
	process.AddStep(NormalStep3)
	process.AddStepWithAlias("4", NormalStep1, NormalStep1)
	process.AddStepWithAlias("5", NormalStep1, "NormalStep1")
	features := flow.DoneFlow("TestTestMultipleNormalStepWithoutAlias", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("excute 5 step, but current = %d", current)
	}
}

func TestMultipleNormalStepWithMultipleBranches(t *testing.T) {
	defer resetCurrent()
	workflow := flow.AddFlowFactory("TestMultipleNormalStepWithMultipleBranches")
	process := workflow.AddProcess("TestMultipleNormalStepWithMultipleBranches", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "2")
	process.AddStepWithAlias("5", GenerateStep(5), "2")
	process.AddWaitAll("6", GenerateStep(6))
	features := flow.DoneFlow("TestMultipleNormalStepWithMultipleBranches", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestMultipleNormalStepsWithWaitBefore(t *testing.T) {
	defer resetCurrent()
	factory := flow.AddFlowFactory("TestMultipleNormalStepsWithWaitBefore")
	process := factory.AddProcess("TestMultipleNormalStepsWithWaitBefore", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddWaitBefore("2", GenerateStep(2))
	process.AddWaitBefore("3", GenerateStep(3))
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddWaitBefore("12", GenerateStep(12))
	process.AddWaitBefore("13", GenerateStep(13))
	features := flow.DoneFlow("TestMultipleNormalStepsWithWaitBefore", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestMultipleNormalSteps(t *testing.T) {
	defer resetCurrent()
	workflow := flow.AddFlowFactory("TestMultipleNormalSteps")
	process := workflow.AddProcess("TestMultipleNormalSteps", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddStepWithAlias("12", GenerateStep(12), "11")
	process.AddStepWithAlias("13", GenerateStep(13), "12")
	features := flow.DoneFlow("TestMultipleNormalSteps", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestWorkFlowPause(t *testing.T) {
	defer resetCurrent()
	workflow := flow.AddFlowFactory("TestWorkFlowPause")
	process := workflow.AddProcess("TestWorkFlowPause", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddStepWithAlias("12", GenerateStep(12), "11")
	process.AddStepWithAlias("13", GenerateStep(13), "12")
	wf := flow.AsyncFlow("TestWorkFlowPause", nil)
	time.Sleep(10 * time.Millisecond)
	wf.Pause()
	time.Sleep(200 * time.Millisecond)
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("excute 2 step, but current = %d", current)
	}
	for name, feature := range wf.GetFeatures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !slices.Contains(feature.ExplainStatus(), "Pause") {
			t.Errorf("process[%s] pause fail", name)
		}
	}
	wf.Resume()
	wf.Done()
	for name, feature := range wf.GetFeatures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestProcessPause(t *testing.T) {
	defer resetCurrent()
	factory := flow.AddFlowFactory("TestProcessPause")
	process := factory.AddProcess("TestProcessPause", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddStepWithAlias("12", GenerateStep(12), "11")
	process.AddStepWithAlias("13", GenerateStep(13), "12")
	workflow := flow.AsyncFlow("TestProcessPause", nil)
	time.Sleep(10 * time.Millisecond)
	workflow.PauseProcess("TestProcessPause")
	time.Sleep(200 * time.Millisecond)
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("excute 2 step, but current = %d", current)
	}
	for name, feature := range workflow.GetFeatures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !slices.Contains(feature.ExplainStatus(), "Pause") {
			t.Errorf("process[%s] pause fail", name)
		}
	}
	workflow.ResumeProcess("TestProcessPause")
	workflow.Done()
	for name, feature := range workflow.GetFeatures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}
