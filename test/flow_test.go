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
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateErrorStep(1))
	process.AddStepWithAlias("2", GeneratePanicStep(2))
	step := process.AddStepWithAlias("3", GenerateErrorStep(3))
	step.AddConfig(&flow.StepConfig{Timeout: time.Millisecond})
	features := workflow.Done()
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
	if current != 3 {
		t.Errorf("excute 3 step, but current = %d", current)
	}
}

func TestSinglePanicStep(t *testing.T) {
	defer resetCurrent()
	t.Parallel()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GeneratePanicStep(1))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if current != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestGoAheadWithoutDependPanicStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GeneratePanicStep(1))
	process.AddStepWithAlias("-1", GenerateStep(-1), "1")
	process.AddStepWithAlias("-2", GenerateStep(-2), "-1")
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddStepWithAlias("12", GenerateStep(12), "11")
	process.AddStepWithAlias("13", GenerateStep(13), "12")
	process.AddStepWithAlias("14", GenerateStep(14), "13")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if current != 5 {
		t.Errorf("excute 5 step, but current = %d", current)
	}
}

func TestSingleErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateErrorStep(1))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if current != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestGoAheadWithoutDependErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateErrorStep(1))
	process.AddStepWithAlias("-1", GenerateStep(-1), "1")
	process.AddStepWithAlias("-2", GenerateStep(-2), "-1")
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddStepWithAlias("12", GenerateStep(12), "11")
	process.AddStepWithAlias("13", GenerateStep(13), "12")
	process.AddStepWithAlias("14", GenerateStep(14), "13")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if current != 5 {
		t.Errorf("excute 5 step, but current = %d", current)
	}
}

func TestSingleNormalStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if current != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestTestMultipleNormalStepWithoutAlias(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStep(NormalStep1)
	process.AddStep(NormalStep2)
	process.AddStep(NormalStep3)
	process.AddStepWithAlias("4", NormalStep1, NormalStep1)
	process.AddStepWithAlias("5", NormalStep1, "NormalStep1")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if current != 5 {
		t.Errorf("excute 5 step, but current = %d", current)
	}
}

func TestMultipleNormalStepWithMultipleBranches(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "2")
	process.AddStepWithAlias("5", GenerateStep(5), "2")
	process.AddWaitAll("6", GenerateStep(6))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if current != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestMultipleNormalStepsWithWaitBefore(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddWaitBefore("2", GenerateStep(2))
	process.AddWaitBefore("3", GenerateStep(3))
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddWaitBefore("12", GenerateStep(12))
	process.AddWaitBefore("13", GenerateStep(13))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if current != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestMultipleNormalSteps(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddStepWithAlias("12", GenerateStep(12), "11")
	process.AddStepWithAlias("13", GenerateStep(13), "12")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if current != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestWorkFlowPause(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddStepWithAlias("12", GenerateStep(12), "11")
	process.AddStepWithAlias("13", GenerateStep(13), "12")
	features := workflow.Flow()
	time.Sleep(10 * time.Millisecond)
	workflow.Pause()
	time.Sleep(200 * time.Millisecond)
	if current != 2 {
		t.Errorf("excute 2 step, but current = %d", current)
	}
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !slices.Contains(feature.ExplainStatus(), "Pause") {
			t.Errorf("process[%s] pause fail", name)
		}
	}
	workflow.Resume()
	workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if current != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestProcessPause(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("11", GenerateStep(11))
	process.AddStepWithAlias("12", GenerateStep(12), "11")
	process.AddStepWithAlias("13", GenerateStep(13), "12")
	features := workflow.Flow()
	time.Sleep(10 * time.Millisecond)
	workflow.PauseProcess("test1")
	time.Sleep(200 * time.Millisecond)
	if current != 2 {
		t.Errorf("excute 2 step, but current = %d", current)
	}
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !slices.Contains(feature.ExplainStatus(), "Pause") {
			t.Errorf("process[%s] pause fail", name)
		}
	}
	workflow.ResumeProcess("test1")
	workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if current != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}
