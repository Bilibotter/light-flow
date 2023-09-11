package test

import (
	"fmt"
	"gitee.com/MetaphysicCoding/light-flow"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func PreProcessor(stepName string, ctx *light_flow.Context) {
	if stepName == "" {
		panic("step name is empty")
	}
	if ctx == nil {
		panic("step context is nil")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("step[%s] start\n", stepName)
}

func PostProcessor(info *light_flow.StepInfo) {
	if info.Name == "" {
		panic("step name is empty")
	}
	if info.Start.IsZero() {
		panic("step start time is zero")
	}
	if info.End.IsZero() {
		panic("step end time is zero")
	}
	if info.Ctx == nil {
		panic("step context is nil")
	}
	if info.Status == light_flow.Pending {
		panic("step status is pending")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("step[%s] finish\n", info.Name)
}

func TestPreAndPostProcessor(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	config := light_flow.ProduceConfig{
		PreProcessor:  PreProcessor,
		PostProcessor: PostProcessor,
	}
	procedure := workflow.AddProcedure("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.WaitToDone()
	for name, feature := range features {
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 12 {
		t.Errorf("excute 12 step, but current = %d", current)
	}
}

func TestWithLongProcedureTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	config := light_flow.ProduceConfig{ProcedureTimeout: 1 * time.Second}
	procedure := workflow.AddProcedure("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.WaitToDone()
	for name, feature := range features {
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestWithShortProcedureTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	config := light_flow.ProduceConfig{ProcedureTimeout: 1 * time.Millisecond}
	procedure := workflow.AddProcedure("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.WaitToDone()
	for name, feature := range features {
		if feature.IsSuccess() {
			t.Errorf("procedure[%s] success with timeout", name)
		}
	}
	time.Sleep(400 * time.Millisecond)
	if current != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestParallelWithLongDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	config := light_flow.ProduceConfig{StepRetry: 3, StepTimeout: 300 * time.Millisecond}
	procedure := workflow.AddProcedure("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2))
	procedure.AddStepWithAlias("3", GenerateStep(3))
	procedure.AddStepWithAlias("4", GenerateStep(4))
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestWithLongDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	config := light_flow.ProduceConfig{StepRetry: 3, StepTimeout: 1 * time.Second}
	procedure := workflow.AddProcedure("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestWithShortDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	config := light_flow.ProduceConfig{StepRetry: 3, StepTimeout: 1 * time.Millisecond}
	procedure := workflow.AddProcedure("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.IsSuccess() {
			t.Errorf("procedure[%s] success with timeout", name)
		}
	}
	time.Sleep(400 * time.Millisecond)
	if current != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestWithLongStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	procedure := workflow.AddProcedure("test1", nil)
	config := light_flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Second}
	procedure.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] failed", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestWithShortStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	procedure := workflow.AddProcedure("test1", nil)
	config := light_flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Millisecond}
	procedure.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.IsSuccess() {
			t.Errorf("procedure[%s] success with timeout", name)
		}
	}
	time.Sleep(300 * time.Millisecond)
	if current != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithProcedureRetry(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	config := light_flow.ProduceConfig{StepRetry: 3}
	procedure := workflow.AddProcedure("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateErrorStep(1))
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.IsSuccess() {
			t.Errorf("procedure[%s] success, but expected failed", name)
		}
	}
	if current != 3 {
		t.Errorf("excute 3 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithStepRetry(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	config := light_flow.StepConfig{MaxRetry: 3}
	procedure := workflow.AddProcedure("test1", nil)
	procedure.AddStepWithAlias("1", GenerateErrorStep(1)).AddConfig(&config)
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.IsSuccess() {
			t.Errorf("procedure[%s] success, but expected failed", name)
		}
	}
	if current != 3 {
		t.Errorf("excute 3 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithProcedureAndStepRetry(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	config := light_flow.ProduceConfig{StepRetry: 3}
	stepConfig := light_flow.StepConfig{MaxRetry: 2}
	procedure := workflow.AddProcedure("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateErrorStep(1)).AddConfig(&stepConfig)
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.IsSuccess() {
			t.Errorf("procedure[%s] success, but expected failed", name)
		}
	}
	if current != 2 {
		t.Errorf("excute 2 step, but current = %d", current)
	}
}

func TestRecoverSerialStep(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	procedure := workflow.AddProcedure("test1", nil)
	config := light_flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Second}
	procedure.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	procedure.AddStepWithAlias("5", GenerateStep(5), "4")
	procedure.SkipFinishedStep("1", nil)
	procedure.SkipFinishedStep("2", nil)
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] failed", name)
		}
	}
	if current != 3 {
		t.Errorf("excute 3 step, but current = %d", current)
	}
}

func TestRecoverParallelStep(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	procedure := workflow.AddProcedure("test1", nil)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("11", GenerateStep(11), "1")
	procedure.AddStepWithAlias("2", GenerateStep(2))
	procedure.AddStepWithAlias("12", GenerateStep(12), "2")
	procedure.AddStepWithAlias("3", GenerateStep(3))
	procedure.AddStepWithAlias("13", GenerateStep(13), "3")
	procedure.AddStepWithAlias("4", GenerateStep(4))
	procedure.AddStepWithAlias("14", GenerateStep(14), "4")
	procedure.SkipFinishedStep("1", nil)
	procedure.SkipFinishedStep("2", nil)
	procedure.SkipFinishedStep("3", nil)
	procedure.SkipFinishedStep("4", nil)
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestRecoverAndWaitAll(t *testing.T) {
	defer resetCurrent()
	workflow := light_flow.NewWorkflow(nil)
	procedure := workflow.AddProcedure("test1", nil)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2))
	procedure.AddStepWithAlias("3", GenerateStep(3))
	procedure.AddStepWithAlias("4", GenerateStep(4))
	procedure.AddWaitAll("5", GenerateStep(5))
	procedure.SkipFinishedStep("1", nil)
	procedure.SkipFinishedStep("2", nil)
	procedure.SkipFinishedStep("3", nil)
	procedure.SkipFinishedStep("4", nil)
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}
