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
	fmt.Printf("step[%s] PreProcessor exeucte\n", info.Name)
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
	fmt.Printf("step[%s] PostProcessor execute\n", info.Name)
	return true
}

func CompleteProcessor(info *flow.ProcessInfo) bool {
	if info.Name == "" {
		panic("process name is empty")
	}
	if len(info.Id) == 0 {
		panic("process id is empty")
	}
	if len(info.FlowId) == 0 {
		panic("process flow id is empty")
	}
	if info.Ctx == nil {
		panic("process context is nil")
	}
	if len(info.StepMap) == 0 {
		panic("process step map is empty")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("process[%s] CompleteProcessor execute \n", info.Name)
	return true
}

func TestProcessorWhenExceptionOccur(t *testing.T) {
	defer resetCurrent()
	config := flow.ProcessConfig{
		PreProcessors:      []func(*flow.StepInfo) bool{PreProcessor},
		PostProcessors:     []func(*flow.StepInfo) bool{PostProcessor},
		CompleteProcessors: []func(*flow.ProcessInfo) bool{CompleteProcessor},
	}
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", &config)
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
	if current != 10 {
		t.Errorf("excute 10 step, but current = %d", current)
	}
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
		CompleteProcessors: []func(*flow.ProcessInfo) bool{ProcessInfoChecker},
	}
	flow.SetDefaultProcessConfig(&config)
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "1")
	process.AddStepWithAlias("4", GenerateStep(4), "2", "3")
	features := workflow.Done()
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
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
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
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
	process := workflow.AddProcess("test1", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if current != 13 {
		t.Errorf("excute 13 step, but current = %d", current)
	}
}

func TestWithLongprocessTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{ProcessTimeout: 1 * time.Second}
	process := workflow.AddProcess("test1", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestWithShortprocessTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{ProcessTimeout: 1 * time.Millisecond}
	process := workflow.AddProcess("test1", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		if feature.Success() {
			t.Errorf("process[%s] success with timeout", name)
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
	process := workflow.AddProcess("test1", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2))
	process.AddStepWithAlias("3", GenerateStep(3))
	process.AddStepWithAlias("4", GenerateStep(4))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
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
	process := workflow.AddProcess("test1", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
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
	process := workflow.AddProcess("test1", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success with timeout", name)
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
	process := workflow.AddProcess("test1", nil)
	config := flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Second}
	process.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] failed", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestWithShortStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	config := flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Millisecond}
	process.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success with timeout", name)
		}
	}
	time.Sleep(300 * time.Millisecond)
	if current != 1 {
		t.Errorf("excute 1 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithprocessRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{StepRetry: 3}
	process := workflow.AddProcess("test1", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateErrorStep(1))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
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
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateErrorStep(1)).AddConfig(&config)
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if current != 3 {
		t.Errorf("excute 3 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithprocessAndStepRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	config := flow.ProcessConfig{StepRetry: 3}
	stepConfig := flow.StepConfig{MaxRetry: 2}
	process := workflow.AddProcess("test1", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateErrorStep(1)).AddConfig(&stepConfig)
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if current != 2 {
		t.Errorf("excute 2 step, but current = %d", current)
	}
}

func TestRecoverSerialStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	config := flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Second}
	process.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	process.AddStepWithAlias("5", GenerateStep(5), "4")
	process.SkipFinishedStep("1", nil)
	process.SkipFinishedStep("2", nil)
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] failed", name)
		}
	}
	if current != 3 {
		t.Errorf("excute 3 step, but current = %d", current)
	}
}

func TestRecoverParallelStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("11", GenerateStep(11), "1")
	process.AddStepWithAlias("2", GenerateStep(2))
	process.AddStepWithAlias("12", GenerateStep(12), "2")
	process.AddStepWithAlias("3", GenerateStep(3))
	process.AddStepWithAlias("13", GenerateStep(13), "3")
	process.AddStepWithAlias("4", GenerateStep(4))
	process.AddStepWithAlias("14", GenerateStep(14), "4")
	process.SkipFinishedStep("1", nil)
	process.SkipFinishedStep("2", nil)
	process.SkipFinishedStep("3", nil)
	process.SkipFinishedStep("4", nil)
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if current != 4 {
		t.Errorf("excute 4 step, but current = %d", current)
	}
}

func TestRecoverAndWaitAll(t *testing.T) {
	defer resetCurrent()
	workflow := flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2))
	process.AddStepWithAlias("3", GenerateStep(3))
	process.AddStepWithAlias("4", GenerateStep(4))
	process.AddWaitAll("5", GenerateStep(5))
	process.SkipFinishedStep("1", nil)
	process.SkipFinishedStep("2", nil)
	process.SkipFinishedStep("3", nil)
	process.SkipFinishedStep("4", nil)
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
