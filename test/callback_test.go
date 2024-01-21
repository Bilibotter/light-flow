package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"sync/atomic"
	"testing"
)

func AfterFlowProcessor(info *flow.FlowInfo) (bool, error) {
	if info.Name == "" {
		panic("flow name is empty")
	}
	if len(info.Id) == 0 {
		panic("flow id is empty")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..process[%s] AfterFlowProcessor execute \n", info.Name)
	return true, nil
}

func BeforeFlowProcessor(info *flow.FlowInfo) (bool, error) {
	if info.Name == "" {
		panic("flow name is empty")
	}
	if len(info.Id) == 0 {
		panic("flow id is empty")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..process[%s] BeforeFlowProcessor execute \n", info.Name)
	return true, nil
}

func TestInfoCorrect(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.CreateDefaultConfig()
	}()
	config := flow.CreateDefaultConfig()
	config.BeforeStep(true, StepInfoChecker("1", []string{}, []string{"2", "3"}))
	config.BeforeStep(true, StepInfoChecker("2", []string{"1"}, []string{"4"}))
	config.BeforeStep(true, StepInfoChecker("3", []string{"1"}, []string{"4"}))
	config.BeforeStep(true, StepInfoChecker("4", []string{"2", "3"}, []string{}))
	config.AfterStep(true, StepInfoChecker("1", []string{}, []string{"2", "3"}))
	config.AfterStep(true, StepInfoChecker("2", []string{"1"}, []string{"4"}))
	config.AfterStep(true, StepInfoChecker("3", []string{"1"}, []string{"4"}))
	config.AfterStep(true, StepInfoChecker("4", []string{"2", "3"}, []string{}))

	workflow := flow.RegisterFlow("TestInfoCorrect")
	process := workflow.Process("TestInfoCorrect")
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "1")
	process.AliasStep("4", GenerateStep(4), "2", "3")
	features := flow.DoneFlow("TestInfoCorrect", nil)
	for _, feature := range features.Futures() {
		if !feature.Success() {
			t.Errorf("process[%s] fail, exception=%v", feature.GetName(), feature.Exceptions())
		}
	}
	if atomic.LoadInt64(&current) != 12 {
		t.Errorf("execute 12 step, but current = %d", current)
	}
}

func TestDefaultProcessConfig(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.CreateDefaultConfig()
	}()
	config := flow.CreateDefaultConfig()
	config.BeforeStep(true, PreProcessor)
	config.AfterStep(true, PostProcessor)
	config.BeforeProcess(true, ProcProcessor)
	config.AfterProcess(true, ProcProcessor)
	config.BeforeFlow(true, BeforeFlowProcessor)
	config.AfterFlow(true, AfterFlowProcessor)

	workflow := flow.RegisterFlow("TestDefaultProcessConfig")
	process := workflow.Process("TestDefaultProcessConfig")
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "2")
	process.AliasStep("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestDefaultProcessConfig", nil)
	for _, feature := range features.Futures() {
		if !feature.Success() {
			t.Errorf("process[%s] fail, exception=%v", feature.GetName(), feature.ExplainStatus())
		}
	}
	if atomic.LoadInt64(&current) != 16 {
		t.Errorf("execute 16 step, but current = %d", current)
	}
}

func TestMergeDefaultProcessConfig(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.CreateDefaultConfig()
	}()
	config := flow.CreateDefaultConfig()
	config.BeforeStep(true, PreProcessor)
	config.AfterStep(true, PostProcessor)
	config.BeforeProcess(true, ProcProcessor)
	config.AfterProcess(true, ProcProcessor)
	config.BeforeFlow(true, BeforeFlowProcessor)
	config.AfterFlow(true, AfterFlowProcessor)

	workflow := flow.RegisterFlow("TestMergeDefaultProcessConfig")
	workflow.BeforeFlow(true, BeforeFlowProcessor)
	workflow.AfterFlow(true, AfterFlowProcessor)
	process := workflow.Process("TestMergeDefaultProcessConfig")
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.BeforeStep(true, PreProcessor)
	process.AfterStep(true, PostProcessor)
	process.BeforeProcess(true, ProcProcessor)
	process.AfterProcess(true, ProcProcessor)
	features := flow.DoneFlow("TestMergeDefaultProcessConfig", nil)
	for _, feature := range features.Futures() {
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
	if atomic.LoadInt64(&current) != 18 {
		t.Errorf("execute 18 step, but current = %d", current)
	}
}

func TestCallbackCond(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.CreateDefaultConfig()
	}()
	config := flow.CreateDefaultConfig()
	config.BeforeFlow(true, BeforeFlowProcessor).OnlyFor("TestCallbackCond")
	config.BeforeFlow(true, BeforeFlowProcessor).OnlyFor("TestCallbackCondNotExist")
	config.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").When(flow.Failed)
	config.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").When(flow.Success)
	config.BeforeProcess(true, ProcProcessor).OnlyFor("TestCallbackCond")
	config.BeforeProcess(true, ProcProcessor).OnlyFor("TestCallbackCondNotExist")
	config.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond").When(flow.Failed)
	config.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond").When(flow.Success)
	config.BeforeStep(true, PreProcessor).OnlyFor("1")
	config.BeforeStep(true, PreProcessor).OnlyFor("NotExist")
	config.AfterStep(true, PostProcessor).OnlyFor("1").When(flow.Success)
	config.AfterStep(true, PostProcessor).OnlyFor("1").When(flow.Failed)

	workflow := flow.RegisterFlow("TestCallbackCond")
	process := workflow.Process("TestCallbackCond")
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	features := flow.DoneFlow("TestCallbackCond", nil)
	for _, feature := range features.Futures() {
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
	if atomic.LoadInt64(&current) != 8 {
		t.Errorf("execute 8 step, but current = %d", current)
	}
}

func TestCallbackCond0(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.CreateDefaultConfig()
	}()
	config := flow.CreateDefaultConfig()
	config.BeforeFlow(true, BeforeFlowProcessor).NotFor("TestCallbackCond0")
	config.BeforeFlow(true, BeforeFlowProcessor).NotFor("TestCallbackCondNotExist")
	config.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Failed)
	config.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").Exclude(flow.Success)
	config.BeforeProcess(true, ProcProcessor).NotFor("TestCallbackCond0")
	config.BeforeProcess(true, ProcProcessor).NotFor("TestCallbackCondNotExist")
	config.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Failed)
	config.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Success)
	config.BeforeStep(true, PreProcessor).NotFor("1")
	config.AfterStep(true, PostProcessor).OnlyFor("1").Exclude(flow.Success)
	config.AfterStep(true, PostProcessor).OnlyFor("1").Exclude(flow.Failed)

	workflow := flow.RegisterFlow("TestCallbackCond0")
	process := workflow.Process("TestCallbackCond0")
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	features := flow.DoneFlow("TestCallbackCond0", nil)
	for _, feature := range features.Futures() {
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
	if atomic.LoadInt64(&current) != 8 {
		t.Errorf("execute 8 step, but current = %d", current)
	}
}

func TestUnableDefaultProcessConfig(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.CreateDefaultConfig()
	}()
	config := flow.CreateDefaultConfig()
	config.BeforeStep(true, PreProcessor)
	config.AfterStep(true, PostProcessor)
	config.BeforeProcess(true, ProcProcessor)
	config.AfterProcess(true, ProcProcessor)
	config.BeforeFlow(true, BeforeFlowProcessor)
	config.AfterFlow(true, AfterFlowProcessor)

	workflow := flow.RegisterFlow("TestUnableDefaultProcessConfig")
	workflow.NoUseDefault()
	process := workflow.Process("TestUnableDefaultProcessConfig")
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "2")
	process.AliasStep("4", GenerateStep(4), "3")
	process.NotUseDefault()
	features := flow.DoneFlow("TestUnableDefaultProcessConfig", nil)
	for _, feature := range features.Futures() {
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}
