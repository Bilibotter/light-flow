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

func StepInfoChecker(key string, prev, next []string) func(info *flow.StepInfo) bool {
	return func(info *flow.StepInfo) bool {
		if info.Name != key {
			return true
		}
		println("matched", key)
		atomic.AddInt64(&current, 1)
		for name := range info.Prev {
			if !slices.Contains(prev, name) {
				panic(fmt.Sprintf("step[%s] prev not contains %s", key, name))
			}
		}
		for name := range info.Next {
			if !slices.Contains(next, name) {
				panic(fmt.Sprintf("step[%s] next not contains %s", key, name))
			}
		}
		return true
	}
}

func ProcessInfoChecker(info *flow.ProcessInfo) bool {
	for _, stepInfo := range info.StepMap {
		for name, id := range stepInfo.Prev {
			prev := info.StepMap[name]
			if prev.Id != id {
				panic(fmt.Sprintf("step[%s] prev[%s] id not match in step map", stepInfo.Name, prev.Name))
			}
		}
	}
	atomic.AddInt64(&current, 1)
	return true
}

func PreProcessor(info *flow.StepInfo) bool {
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
	if info.Ctx == nil {
		panic("step context is nil")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("step[%s] start\n", info.Name)
	return true
}

func PostProcessor(info *flow.StepInfo) bool {
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
	if info.Ctx == nil {
		panic("step context is nil")
	}
	if info.Status == flow.Pending {
		panic("step status is pending")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("step[%s] finish\n", info.Name)
	return true
}

func CompleteProcessor(info *flow.ProcessInfo) bool {
	if info.Name == "" {
		panic("procedure name is empty")
	}
	if len(info.Id) == 0 {
		panic("process id is empty")
	}
	if len(info.FlowId) == 0 {
		panic("process flow id is empty")
	}
	if info.Ctx == nil {
		panic("procedure context is nil")
	}
	if len(info.StepMap) == 0 {
		panic("procedure step map is empty")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("procedure[%s] finish\n", info.Name)
	return true
}

func TestInfoCorrect(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.SetDefaultProcessConfig(nil)
	}()
	config := flow.ProcessConfig{
		PreProcessors: []func(*flow.StepInfo) bool{
			StepInfoChecker("1", []string{}, []string{"2", "3"}),
			StepInfoChecker("2", []string{"1"}, []string{"4"}),
			StepInfoChecker("3", []string{"1"}, []string{"4"}),
			StepInfoChecker("4", []string{"2", "3"}, []string{}),
		},

		PostProcessors: []func(*flow.StepInfo) bool{
			StepInfoChecker("1", []string{}, []string{"2", "3"}),
			StepInfoChecker("2", []string{"1"}, []string{"4"}),
			StepInfoChecker("3", []string{"1"}, []string{"4"}),
			StepInfoChecker("4", []string{"2", "3"}, []string{}),
		},
		CompleteProcessors: []func(*flow.ProcessInfo) bool{CompleteProcessor},
	}
	flow.SetDefaultProcessConfig(&config)
	workflow := flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "1")
	procedure.AddStepWithAlias("4", GenerateStep(4), "2", "3")
	features := workflow.Done()
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 13 {
		t.Errorf("excute 13 step, but current = %d", current)
	}
}

func TestDefaultProcessConfig(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.SetDefaultProcessConfig(nil)
	}()
	config := flow.ProcessConfig{
		PreProcessors:      []func(*flow.StepInfo) bool{PreProcessor},
		PostProcessors:     []func(*flow.StepInfo) bool{PostProcessor},
		CompleteProcessors: []func(*flow.ProcessInfo) bool{CompleteProcessor},
	}
	flow.SetDefaultProcessConfig(&config)
	workflow := flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 13 {
		t.Errorf("excute 13 step, but current = %d", current)
	}
}

func TestPreAndPostProcessor(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{
		PreProcessors:      []func(*flow.StepInfo) bool{PreProcessor},
		PostProcessors:     []func(*flow.StepInfo) bool{PostProcessor},
		CompleteProcessors: []func(*flow.ProcessInfo) bool{CompleteProcessor},
	}
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 13 {
		t.Errorf("excute 13 step, but current = %d", current)
	}
}

func TestWithLongProcedureTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{ProcessTimeout: 1 * time.Second}
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestWithShortProcedureTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{ProcessTimeout: 1 * time.Millisecond}
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		if feature.Success() {
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
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{StepRetry: 3, StepTimeout: 300 * time.Millisecond}
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2))
	procedure.AddStepWithAlias("3", GenerateStep(3))
	procedure.AddStepWithAlias("4", GenerateStep(4))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestWithLongDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{StepRetry: 3, StepTimeout: 1 * time.Second}
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestWithShortDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{StepRetry: 3, StepTimeout: 1 * time.Millisecond}
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.Success() {
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
	workflow := flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	config := flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Second}
	procedure.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] failed", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestWithShortStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	config := flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Millisecond}
	procedure.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.Success() {
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
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{StepRetry: 3}
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateErrorStep(1))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("procedure[%s] success, but expected failed", name)
		}
	}
	if current != 3 {
		t.Errorf("excute 3 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithStepRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	config := flow.StepConfig{MaxRetry: 3}
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateErrorStep(1)).AddConfig(&config)
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("procedure[%s] success, but expected failed", name)
		}
	}
	if current != 3 {
		t.Errorf("excute 3 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithProcedureAndStepRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{StepRetry: 3}
	stepConfig := flow.StepConfig{MaxRetry: 2}
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddConfig(&config)
	procedure.AddStepWithAlias("1", GenerateErrorStep(1)).AddConfig(&stepConfig)
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("procedure[%s] success, but expected failed", name)
		}
	}
	if current != 2 {
		t.Errorf("excute 2 step, but current = %d", current)
	}
}

func TestRecoverSerialStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	config := flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Second}
	procedure.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	procedure.AddStepWithAlias("2", GenerateStep(2), "1")
	procedure.AddStepWithAlias("3", GenerateStep(3), "2")
	procedure.AddStepWithAlias("4", GenerateStep(4), "3")
	procedure.AddStepWithAlias("5", GenerateStep(5), "4")
	procedure.SkipFinishedStep("1", nil)
	procedure.SkipFinishedStep("2", nil)
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] failed", name)
		}
	}
	if current != 3 {
		t.Errorf("excute 3 step, but current = %d", current)
	}
}

func TestRecoverParallelStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
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
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestRecoverAndWaitAll(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.AddStepWithAlias("1", GenerateStep(1))
	procedure.AddStepWithAlias("2", GenerateStep(2))
	procedure.AddStepWithAlias("3", GenerateStep(3))
	procedure.AddStepWithAlias("4", GenerateStep(4))
	procedure.AddWaitAll("5", GenerateStep(5))
	procedure.SkipFinishedStep("1", nil)
	procedure.SkipFinishedStep("2", nil)
	procedure.SkipFinishedStep("3", nil)
	procedure.SkipFinishedStep("4", nil)
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
