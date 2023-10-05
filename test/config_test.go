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

func ErrorResultPrinter(info *flow.StepInfo) bool {
	if !flow.IsStatusNormal(info.Status) {
		fmt.Printf("step[%s] error, explain=%v, err=%v\n", info.Name, flow.ExplainStatus(info.Status), info.Err)
	}
	return true
}

func ProcProcessor(info *flow.ProcessInfo) bool {
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
	fmt.Printf("process[%s] ProcProcessor execute \n", info.Name)
	return true
}

func PanicProcProcessor(info *flow.ProcessInfo) bool {
	atomic.AddInt64(&current, 1)
	panic("PanicProcProcessor panic")
	return true
}

func PanicStepProcessor(info *flow.StepInfo) bool {
	atomic.AddInt64(&current, 1)
	panic("PanicStepProcessor panic")
	return true
}

func TestNonEssentialProcProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestNonEssentialProcProcessorPanic1")
	process := workflow.AddProcess("TestNonEssentialProcProcessorPanic1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddBeforeProcess(false, PanicProcProcessor)
	features := flow.DoneFlow("TestNonEssentialProcProcessorPanic1", nil)
	for name, feature := range features {
		explain := feature.ExplainStatus()
		if !feature.Success() {
			t.Errorf("process[%s] failed,explian=%v", name, explain)
		}
		if slices.Contains(explain, "Panic") {
			t.Errorf("process[%s] not panic, but explain contain, but explain=%v", name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
	resetCurrent()
	workflow = flow.RegisterFlow("TestNonEssentialProcProcessorPanic2")
	process = workflow.AddProcess("TestNonEssentialProcProcessorPanic2", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddAfterProcess(false, PanicProcProcessor)
	features = flow.DoneFlow("TestNonEssentialProcProcessorPanic2", nil)
	for name, feature := range features {
		explain := feature.ExplainStatus()
		if !feature.Success() {
			t.Errorf("process[%s] failed,explian=%v", name, explain)
		}
		if slices.Contains(explain, "Panic") {
			t.Errorf("process[%s] not panic, but explain not contain, but explain=%v", name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

func TestEssentialProcProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestEssentialProcProcessorPanic1")
	process := workflow.AddProcess("TestEssentialProcProcessorPanic1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddBeforeProcess(true, PanicProcProcessor)
	features := flow.DoneFlow("TestEssentialProcProcessorPanic1", nil)
	for name, feature := range features {
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
		explain := feature.ExplainStatus()
		if !slices.Contains(explain, "Panic") {
			t.Errorf("process[%s] panic, but explain not cotain, but explain=%v", name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
	resetCurrent()
	workflow = flow.RegisterFlow("TestEssentialProcProcessorPanic2")
	process = workflow.AddProcess("TestEssentialProcProcessorPanic2", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddAfterProcess(true, PanicProcProcessor)
	features = flow.DoneFlow("TestEssentialProcProcessorPanic2", nil)
	for name, feature := range features {
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
		explain := feature.ExplainStatus()
		if !slices.Contains(explain, "Panic") {
			t.Errorf("process[%s] panic, but explain not cotain, but explain=%v", name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

func TestNonEssentialStepProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestNonEssentialStepProcessorPanic1")
	process := workflow.AddProcess("TestNonEssentialStepProcessorPanic1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddBeforeStep(false, PanicStepProcessor)
	features := flow.DoneFlow("TestNonEssentialStepProcessorPanic1", nil)
	for name, feature := range features {
		explain := feature.ExplainStatus()
		if !feature.Success() {
			t.Errorf("process[%s] failed,explian=%v", name, explain)
		}
		if slices.Contains(explain, "Panic") {
			t.Errorf("process[%s] not panic, but explain contain, but explain=%v", name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
	resetCurrent()
	workflow = flow.RegisterFlow("TestNonEssentialStepProcessorPanic2")
	process = workflow.AddProcess("TestNonEssentialStepProcessorPanic2", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddAfterStep(false, PanicStepProcessor)
	features = flow.DoneFlow("TestNonEssentialStepProcessorPanic2", nil)
	for name, feature := range features {
		explain := feature.ExplainStatus()
		if !feature.Success() {
			t.Errorf("process[%s] failed,explian=%v", name, explain)
		}
		if slices.Contains(explain, "Panic") {
			t.Errorf("process[%s] not panic, but explain not contain, but explain=%v", name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

func TestEssentialStepProcessorPanic(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestEssentialStepProcessorPanic1")
	process := workflow.AddProcess("TestEssentialStepProcessorPanic1", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddBeforeStep(true, PanicStepProcessor)
	features := flow.DoneFlow("TestEssentialStepProcessorPanic1", nil)
	for name, feature := range features {
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
		explain := feature.ExplainStatus()
		if !slices.Contains(explain, "Panic") {
			t.Errorf("process[%s] panic, but explain not cotain, but explain=%v", name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
	resetCurrent()
	workflow = flow.RegisterFlow("TestEssentialStepProcessorPanic2")
	process = workflow.AddProcess("TestEssentialStepProcessorPanic2", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddAfterStep(true, PanicStepProcessor)
	features = flow.DoneFlow("TestEssentialStepProcessorPanic2", nil)
	for name, feature := range features {
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
		explain := feature.ExplainStatus()
		if !slices.Contains(explain, "Panic") {
			t.Errorf("process[%s] panic, but explain not cotain, but explain=%v", name, explain)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

func TestProcessorWhenExceptionOccur(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcessorWhenExceptionOccur")
	process := workflow.AddProcess("TestProcessorWhenExceptionOccur", nil)
	process.AddBeforeStep(true, PreProcessor)
	process.AddAfterStep(true, PostProcessor)
	process.AddBeforeProcess(true, ProcProcessor)
	process.AddAfterProcess(true, ProcProcessor)
	process.AddStepWithAlias("1", GenerateErrorStep(1))
	process.AddStepWithAlias("2", GeneratePanicStep(2))
	step := process.AddStepWithAlias("3", GenerateErrorStep(3))
	step.AddConfig(&flow.StepConfig{Timeout: time.Millisecond})
	features := flow.DoneFlow("TestProcessorWhenExceptionOccur", nil)
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
	if atomic.LoadInt64(&current) != 11 {
		t.Errorf("execute 11 step, but current = %d", current)
	}
}
func TestInfoCorrect(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.SetDefaultProcessConfig(nil)
	}()
	config := flow.ProcessConfig{}
	config.AddBeforeStep(true, StepInfoChecker("1", []string{}, []string{"2", "3"}))
	config.AddBeforeStep(true, StepInfoChecker("2", []string{"1"}, []string{"4"}))
	config.AddBeforeStep(true, StepInfoChecker("3", []string{"1"}, []string{"4"}))
	config.AddBeforeStep(true, StepInfoChecker("4", []string{"2", "3"}, []string{}))
	config.AddAfterStep(true, StepInfoChecker("1", []string{}, []string{"2", "3"}))
	config.AddAfterStep(true, StepInfoChecker("2", []string{"1"}, []string{"4"}))
	config.AddAfterStep(true, StepInfoChecker("3", []string{"1"}, []string{"4"}))
	config.AddAfterStep(true, StepInfoChecker("4", []string{"2", "3"}, []string{}))
	flow.SetDefaultProcessConfig(&config)
	workflow := flow.RegisterFlow("TestInfoCorrect")
	process := workflow.AddProcess("TestInfoCorrect", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "1")
	process.AddStepWithAlias("4", GenerateStep(4), "2", "3")
	features := flow.DoneFlow("TestInfoCorrect", nil)
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 12 {
		t.Errorf("execute 12 step, but current = %d", current)
	}
}

func TestDefaultProcessConfig(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.SetDefaultProcessConfig(nil)
	}()
	config := flow.ProcessConfig{}
	config.AddBeforeStep(true, PreProcessor)
	config.AddAfterStep(true, PostProcessor)
	config.AddBeforeProcess(true, ProcProcessor)
	config.AddAfterProcess(true, ProcProcessor)
	flow.SetDefaultProcessConfig(&config)
	workflow := flow.RegisterFlow("TestDefaultProcessConfig")
	process := workflow.AddProcess("TestDefaultProcessConfig", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestDefaultProcessConfig", nil)
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 14 {
		t.Errorf("execute 14 step, but current = %d", current)
	}
}

func TestPreAndPostProcessor(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestPreAndPostProcessor")
	config := flow.ProcessConfig{}
	config.AddBeforeStep(true, PreProcessor)
	config.AddAfterStep(true, PostProcessor)
	config.AddBeforeProcess(true, ProcProcessor)
	config.AddAfterProcess(true, ProcProcessor)
	process := workflow.AddProcess("TestPreAndPostProcessor", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestPreAndPostProcessor", nil)
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 14 {
		t.Errorf("execute 14 step, but current = %d", current)
	}
}

func TestWithLongProcessTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithLongProcessTimeout")
	config := flow.ProcessConfig{ProcTimeout: 1 * time.Second}
	process := workflow.AddProcess("TestWithLongProcessTimeout", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestWithLongProcessTimeout", nil)
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestWithShortProcessTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithShortProcessTimeout")
	config := flow.ProcessConfig{ProcTimeout: 1 * time.Millisecond}
	process := workflow.AddProcess("TestWithShortProcessTimeout", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestWithShortProcessTimeout", nil)
	for name, feature := range features {
		if feature.Success() {
			t.Errorf("process[%s] success with timeout", name)
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
	config := flow.ProcessConfig{StepRetry: 3, StepTimeout: 300 * time.Millisecond}
	process := workflow.AddProcess("TestParallelWithLongDefaultStepTimeout", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2))
	process.AddStepWithAlias("3", GenerateStep(3))
	process.AddStepWithAlias("4", GenerateStep(4))
	features := flow.DoneFlow("TestParallelWithLongDefaultStepTimeout", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestWithLongDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithLongDefaultStepTimeout")
	config := flow.ProcessConfig{StepRetry: 3, StepTimeout: 1 * time.Second}
	process := workflow.AddProcess("TestWithLongDefaultStepTimeout", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestWithLongDefaultStepTimeout", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestWithShortDefaultStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithShortDefaultStepTimeout")
	config := flow.ProcessConfig{StepRetry: 3, StepTimeout: 1 * time.Millisecond}
	process := workflow.AddProcess("TestWithShortDefaultStepTimeout", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestWithShortDefaultStepTimeout", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success with timeout", name)
		}
	}
	time.Sleep(400 * time.Millisecond)
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
}

func TestWithLongStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithLongStepTimeout")
	process := workflow.AddProcess("TestWithLongStepTimeout", nil)
	config := flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Second}
	process.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestWithLongStepTimeout", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] failed", name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestWithShortStepTimeout(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestWithShortStepTimeout")
	process := workflow.AddProcess("TestWithShortStepTimeout", nil)
	config := flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Millisecond}
	process.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestWithShortStepTimeout", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success with timeout", name)
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
	config := flow.ProcessConfig{StepRetry: 3}
	process := workflow.AddProcess("TestSingleErrorStepWithProcessRetry", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateErrorStep(1))
	features := flow.DoneFlow("TestSingleErrorStepWithProcessRetry", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if atomic.LoadInt64(&current) != 3 {
		t.Errorf("execute 3 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithStepRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleErrorStepWithStepRetry")
	config := flow.StepConfig{MaxRetry: 3}
	process := workflow.AddProcess("TestSingleErrorStepWithStepRetry", nil)
	process.AddStepWithAlias("1", GenerateErrorStep(1)).AddConfig(&config)
	features := flow.DoneFlow("TestSingleErrorStepWithStepRetry", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if atomic.LoadInt64(&current) != 3 {
		t.Errorf("execute 3 step, but current = %d", current)
	}
}

func TestSingleErrorStepWithProcessAndStepRetry(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSingleErrorStepWithProcessAndStepRetry")
	config := flow.ProcessConfig{StepRetry: 3}
	stepConfig := flow.StepConfig{MaxRetry: 2}
	process := workflow.AddProcess("TestSingleErrorStepWithProcessAndStepRetry", nil)
	process.AddConfig(&config)
	process.AddStepWithAlias("1", GenerateErrorStep(1)).AddConfig(&stepConfig)
	features := flow.DoneFlow("TestSingleErrorStepWithProcessAndStepRetry", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if feature.Success() {
			t.Errorf("process[%s] success, but expected failed", name)
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

func TestRecoverSerialStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestRecoverSerialStep")
	process := workflow.AddProcess("TestRecoverSerialStep", nil)
	config := flow.StepConfig{MaxRetry: 3, Timeout: 1 * time.Second}
	process.AddStepWithAlias("1", GenerateStep(1)).AddConfig(&config)
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	process.AddStepWithAlias("5", GenerateStep(5), "4")
	wf := flow.BuildWorkflow("TestRecoverSerialStep", nil)
	if err := wf.SkipFinishedStep("1", nil); err != nil {
		panic(err.Error())
	}
	if err := wf.SkipFinishedStep("2", nil); err != nil {
		panic(err.Error())
	}
	features := wf.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] failed", name)
		}
	}
	if atomic.LoadInt64(&current) != 3 {
		t.Errorf("execute 3 step, but current = %d", current)
	}
}

func TestRecoverParallelStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestRecoverParallelStep")
	process := workflow.AddProcess("TestRecoverParallelStep", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("11", GenerateStep(11), "1")
	process.AddStepWithAlias("2", GenerateStep(2))
	process.AddStepWithAlias("12", GenerateStep(12), "2")
	process.AddStepWithAlias("3", GenerateStep(3))
	process.AddStepWithAlias("13", GenerateStep(13), "3")
	process.AddStepWithAlias("4", GenerateStep(4))
	process.AddStepWithAlias("14", GenerateStep(14), "4")
	wf := flow.BuildWorkflow("TestRecoverParallelStep", nil)
	if err := wf.SkipFinishedStep("1", nil); err != nil {
		panic(err.Error())
	}
	if err := wf.SkipFinishedStep("2", nil); err != nil {
		panic(err.Error())
	}
	if err := wf.SkipFinishedStep("3", nil); err != nil {
		panic(err.Error())
	}
	if err := wf.SkipFinishedStep("4", nil); err != nil {
		panic(err.Error())
	}
	features := wf.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestRecoverAndWaitAll(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestRecoverAndWaitAll")
	process := workflow.AddProcess("TestRecoverAndWaitAll", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2))
	process.AddStepWithAlias("3", GenerateStep(3))
	process.AddStepWithAlias("4", GenerateStep(4))
	process.AddWaitAll("5", GenerateStep(5))
	wf := flow.BuildWorkflow("TestRecoverAndWaitAll", nil)
	if err := wf.SkipFinishedStep("1", nil); err != nil {
		panic(err.Error())
	}
	if err := wf.SkipFinishedStep("2", nil); err != nil {
		panic(err.Error())
	}
	if err := wf.SkipFinishedStep("3", nil); err != nil {
		panic(err.Error())
	}
	if err := wf.SkipFinishedStep("4", nil); err != nil {
		panic(err.Error())
	}
	features := wf.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("execute 1 step, but current = %d", current)
	}
}
