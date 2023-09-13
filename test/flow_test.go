package test

import (
	"fmt"
	"gitee.com/MetaphysicCoding/light-flow"
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

func NormalStep3(ctx *light_flow.Context) (any, error) {
	atomic.AddInt64(&current, 1)
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("3.normal step finish\n")
	return 3, nil
}

func NormalStep2(ctx *light_flow.Context) (any, error) {
	atomic.AddInt64(&current, 1)
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("2.normal step finish\n")
	return 2, nil
}

func NormalStep1(ctx *light_flow.Context) (any, error) {
	atomic.AddInt64(&current, 1)
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("1.normal step finish\n")
	return 1, nil
}

func GenerateStep(i int) func(ctx *light_flow.Context) (any, error) {
	return func(ctx *light_flow.Context) (any, error) {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("%d.step finish\n", i)
		atomic.AddInt64(&current, 1)
		return i, nil
	}
}

func GenerateErrorStep(i int) func(ctx *light_flow.Context) (any, error) {
	return func(ctx *light_flow.Context) (any, error) {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("%d.step error occur \n", i)
		atomic.AddInt64(&current, 1)
		return i, fmt.Errorf("%d.step error", i)
	}
}

func GeneratePanicStep(i int) func(ctx *light_flow.Context) (any, error) {
	return func(ctx *light_flow.Context) (any, error) {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("%d.step panic \n", i)
		atomic.AddInt64(&current, 1)
		panic(fmt.Sprintf("%d.step panic", i))
	}
}

func TestSinglePanicStep(t *testing.T) {
	defer resetCurrent()
	t.Parallel()
	workflow := light_flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GeneratePanicStep(1))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("procedure[%s] success, but expected failed", name)
		}
	}
	if current != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestGoAheadWithoutDependPanicStep(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GeneratePanicStep(1))
	procedure.AddStepWithAlias("-1", GenerateStep(-1), "1")
	procedure.AddStepWithAlias("-2", GenerateStep(-2), "-1")
	procedure.AddStepWithAlias("11", GenerateStep(11))
	procedure.AddStepWithAlias("12", GenerateStep(12), "11")
	procedure.AddStepWithAlias("13", GenerateStep(13), "12")
	procedure.AddStepWithAlias("14", GenerateStep(14), "13")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("procedure[%s] success, but expected failed", name)
		}
	}
	if current != 5 {
		t.Errorf("excute 5 step, but current = %d", current)
	}
}

func TestSingleErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateErrorStep(1))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("procedure[%s] success, but expected failed", name)
		}
	}
	if current != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestGoAheadWithoutDependErrorStep(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateErrorStep(1))
	procedure.AddStepWithAlias("-1", GenerateStep(-1), "1")
	procedure.AddStepWithAlias("-2", GenerateStep(-2), "-1")
	procedure.AddStepWithAlias("11", GenerateStep(11))
	procedure.AddStepWithAlias("12", GenerateStep(12), "11")
	procedure.AddStepWithAlias("13", GenerateStep(13), "12")
	procedure.AddStepWithAlias("14", GenerateStep(14), "13")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("procedure[%s] success, but expected failed", name)
		}
	}
	if current != 5 {
		t.Errorf("excute 5 step, but current = %d", current)
	}
}

func TestSingleNormalStep(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestTestMultipleNormalStepWithoutAlias(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStep(NormalStep1)
	procedure.AddStep(NormalStep2)
	procedure.AddStep(NormalStep3)
	procedure.AddStepWithAlias("4", NormalStep1, NormalStep1)
	procedure.AddStepWithAlias("5", NormalStep1, "NormalStep1")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 5 {
		t.Errorf("excute 5 step, but current = %d", current)
	}
}

func TestMultipleNormalStepWithMultipleBranches(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "2")
	procedure.AddStepWithAlias("5", GenerateStep(5), "2")
	procedure.AddWaitAll("6", GenerateStep(6))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestMultipleNormalStepsWithWaitBefore(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddWaitBefore("2", GenerateStep(2))
	procedure.AddWaitBefore("3", GenerateStep(3))
	procedure.AddStepWithAlias("11", GenerateStep(11))
	procedure.AddWaitBefore("12", GenerateStep(12))
	procedure.AddWaitBefore("13", GenerateStep(13))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestMultipleNormalSteps(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("11", GenerateStep(11))
	procedure.AddStepWithAlias("12", GenerateStep(12), "11")
	procedure.AddStepWithAlias("13", GenerateStep(13), "12")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestWorkFlowPause(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("11", GenerateStep(11))
	procedure.AddStepWithAlias("12", GenerateStep(12), "11")
	procedure.AddStepWithAlias("13", GenerateStep(13), "12")
	features := workflow.Flow()
	time.Sleep(10 * time.Millisecond)
	workflow.Pause()
	time.Sleep(200 * time.Millisecond)
	if current != 2 {
		t.Errorf("excute 2 step, but current = %d", current)
	}
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !slices.Contains(feature.ExplainStatus(), "Pause") {
			t.Errorf("procedure[%s] pause fail", name)
		}
	}
	workflow.Resume()
	workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}

func TestProcessPause(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("11", GenerateStep(11))
	procedure.AddStepWithAlias("12", GenerateStep(12), "11")
	procedure.AddStepWithAlias("13", GenerateStep(13), "12")
	features := workflow.Flow()
	time.Sleep(10 * time.Millisecond)
	workflow.PauseProcess("test1")
	time.Sleep(200 * time.Millisecond)
	if current != 2 {
		t.Errorf("excute 2 step, but current = %d", current)
	}
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !slices.Contains(feature.ExplainStatus(), "Pause") {
			t.Errorf("procedure[%s] pause fail", name)
		}
	}
	workflow.ResumeProcess("test1")
	workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 6 {
		t.Errorf("excute 6 step, but current = %d", current)
	}
}
