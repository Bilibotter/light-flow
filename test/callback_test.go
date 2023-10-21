package test

import (
	"fmt"
	flow "gitee.com/MetaphysicCoding/light-flow"
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
	if info.Context == nil {
		panic("flow context is nil")
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
	if info.Context == nil {
		panic("flow context is nil")
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
	config.AddBeforeStep(true, StepInfoChecker("1", []string{}, []string{"2", "3"}))
	config.AddBeforeStep(true, StepInfoChecker("2", []string{"1"}, []string{"4"}))
	config.AddBeforeStep(true, StepInfoChecker("3", []string{"1"}, []string{"4"}))
	config.AddBeforeStep(true, StepInfoChecker("4", []string{"2", "3"}, []string{}))
	config.AddAfterStep(true, StepInfoChecker("1", []string{}, []string{"2", "3"}))
	config.AddAfterStep(true, StepInfoChecker("2", []string{"1"}, []string{"4"}))
	config.AddAfterStep(true, StepInfoChecker("3", []string{"1"}, []string{"4"}))
	config.AddAfterStep(true, StepInfoChecker("4", []string{"2", "3"}, []string{}))

	workflow := flow.RegisterFlow("TestInfoCorrect")
	process := workflow.AddProcessWithConf("TestInfoCorrect", nil)
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "1")
	process.AliasStep("4", GenerateStep(4), "2", "3")
	features := flow.DoneFlow("TestInfoCorrect", nil)
	for name, feature := range features.Features() {
		if !feature.Success() {
			t.Errorf("process[%s] fail, exception=%v", name, feature.Exceptions())
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
	config.AddBeforeStep(true, PreProcessor)
	config.AddAfterStep(true, PostProcessor)
	config.AddBeforeProcess(true, ProcProcessor)
	config.AddAfterProcess(true, ProcProcessor)
	config.AddBeforeFlow(true, BeforeFlowProcessor)
	config.AddAfterFlow(true, AfterFlowProcessor)

	workflow := flow.RegisterFlow("TestDefaultProcessConfig")
	process := workflow.AddProcessWithConf("TestDefaultProcessConfig", nil)
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "2")
	process.AliasStep("4", GenerateStep(4), "3")
	features := flow.DoneFlow("TestDefaultProcessConfig", nil)
	for name, feature := range features.Features() {
		if !feature.Success() {
			t.Errorf("process[%s] fail, exception=%v", name, feature.ExplainStatus())
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
	config.AddBeforeStep(true, PreProcessor)
	config.AddAfterStep(true, PostProcessor)
	config.AddBeforeProcess(true, ProcProcessor)
	config.AddAfterProcess(true, ProcProcessor)
	config.AddBeforeFlow(true, BeforeFlowProcessor)
	config.AddAfterFlow(true, AfterFlowProcessor)

	workflow := flow.RegisterFlow("TestMergeDefaultProcessConfig")
	workflow.AddBeforeFlow(true, BeforeFlowProcessor)
	workflow.AddAfterFlow(true, AfterFlowProcessor)
	process := workflow.AddProcessWithConf("TestMergeDefaultProcessConfig", nil)
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AddBeforeStep(true, PreProcessor)
	process.AddAfterStep(true, PostProcessor)
	process.AddBeforeProcess(true, ProcProcessor)
	process.AddAfterProcess(true, ProcProcessor)
	features := flow.DoneFlow("TestMergeDefaultProcessConfig", nil)
	for name, feature := range features.Features() {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
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
	config.AddBeforeFlow(true, BeforeFlowProcessor).OnlyFor("TestCallbackCond")
	config.AddBeforeFlow(true, BeforeFlowProcessor).OnlyFor("TestCallbackCondNotExist")
	config.AddAfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").When(flow.Failed)
	config.AddAfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").When(flow.Success)
	config.AddBeforeProcess(true, ProcProcessor).OnlyFor("TestCallbackCond")
	config.AddBeforeProcess(true, ProcProcessor).OnlyFor("TestCallbackCondNotExist")
	config.AddAfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond").When(flow.Failed)
	config.AddAfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond").When(flow.Success)
	config.AddBeforeStep(true, PreProcessor).OnlyFor("1")
	config.AddBeforeStep(true, PreProcessor).OnlyFor("NotExist")
	config.AddAfterStep(true, PostProcessor).OnlyFor("1").When(flow.Success)
	config.AddAfterStep(true, PostProcessor).OnlyFor("1").When(flow.Failed)

	workflow := flow.RegisterFlow("TestCallbackCond")
	process := workflow.AddProcessWithConf("TestCallbackCond", nil)
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	features := flow.DoneFlow("TestCallbackCond", nil)
	for name, feature := range features.Features() {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
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
	config.AddBeforeFlow(true, BeforeFlowProcessor).NotFor("TestCallbackCond0")
	config.AddBeforeFlow(true, BeforeFlowProcessor).NotFor("TestCallbackCondNotExist")
	config.AddAfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Failed)
	config.AddAfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").Exclude(flow.Success)
	config.AddBeforeProcess(true, ProcProcessor).NotFor("TestCallbackCond0")
	config.AddBeforeProcess(true, ProcProcessor).NotFor("TestCallbackCondNotExist")
	config.AddAfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Failed)
	config.AddAfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Success)
	config.AddBeforeStep(true, PreProcessor).NotFor("1")
	config.AddAfterStep(true, PostProcessor).OnlyFor("1").Exclude(flow.Success)
	config.AddAfterStep(true, PostProcessor).OnlyFor("1").Exclude(flow.Failed)

	workflow := flow.RegisterFlow("TestCallbackCond0")
	process := workflow.AddProcessWithConf("TestCallbackCond0", nil)
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	features := flow.DoneFlow("TestCallbackCond0", nil)
	for name, feature := range features.Features() {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
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
	config.AddBeforeStep(true, PreProcessor)
	config.AddAfterStep(true, PostProcessor)
	config.AddBeforeProcess(true, ProcProcessor)
	config.AddAfterProcess(true, ProcProcessor)
	config.AddBeforeFlow(true, BeforeFlowProcessor)
	config.AddAfterFlow(true, AfterFlowProcessor)

	workflow := flow.RegisterFlow("TestUnableDefaultProcessConfig")
	workflow.NotUseDefault()
	process := workflow.AddProcessWithConf("TestUnableDefaultProcessConfig", nil)
	process.AliasStep("1", GenerateStep(1))
	process.AliasStep("2", GenerateStep(2), "1")
	process.AliasStep("3", GenerateStep(3), "2")
	process.AliasStep("4", GenerateStep(4), "3")
	process.NotUseDefault()
	features := flow.DoneFlow("TestUnableDefaultProcessConfig", nil)
	for name, feature := range features.Features() {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}
