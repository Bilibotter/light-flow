package test

import (
	"fmt"
	flow "gitee.com/MetaphysicCoding/light-flow"
	"sync/atomic"
	"testing"
)

func FlowProcessor(info *flow.FlowInfo) bool {
	if info.Name == "" {
		panic("flow name is empty")
	}
	if len(info.Id) == 0 {
		panic("flow id is empty")
	}
	if info.Ctx == nil {
		panic("flow context is nil")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..process[%s] FlowProcessor execute \n", info.Name)
	return true
}

func TestInfoCorrect(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.SetDefaultConfig(nil)
	}()
	config := flow.Configuration{}
	config.AddBeforeStep(true, StepInfoChecker("1", []string{}, []string{"2", "3"}))
	config.AddBeforeStep(true, StepInfoChecker("2", []string{"1"}, []string{"4"}))
	config.AddBeforeStep(true, StepInfoChecker("3", []string{"1"}, []string{"4"}))
	config.AddBeforeStep(true, StepInfoChecker("4", []string{"2", "3"}, []string{}))
	config.AddAfterStep(true, StepInfoChecker("1", []string{}, []string{"2", "3"}))
	config.AddAfterStep(true, StepInfoChecker("2", []string{"1"}, []string{"4"}))
	config.AddAfterStep(true, StepInfoChecker("3", []string{"1"}, []string{"4"}))
	config.AddAfterStep(true, StepInfoChecker("4", []string{"2", "3"}, []string{}))
	flow.SetDefaultConfig(&config)
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
		flow.SetDefaultConfig(nil)
	}()
	config := flow.Configuration{}
	config.AddBeforeStep(true, PreProcessor)
	config.AddAfterStep(true, PostProcessor)
	config.AddBeforeProcess(true, ProcProcessor)
	config.AddAfterProcess(true, ProcProcessor)
	config.AddBeforeFlow(true, FlowProcessor)
	config.AddAfterFlow(true, FlowProcessor)
	flow.SetDefaultConfig(&config)
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
	if atomic.LoadInt64(&current) != 16 {
		t.Errorf("execute 16 step, but current = %d", current)
	}
}

func TestMergeDefaultProcessConfig(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.SetDefaultConfig(nil)
	}()
	config := flow.Configuration{}
	config.AddBeforeStep(true, PreProcessor)
	config.AddAfterStep(true, PostProcessor)
	config.AddBeforeProcess(true, ProcProcessor)
	config.AddAfterProcess(true, ProcProcessor)
	config.AddBeforeFlow(true, FlowProcessor)
	config.AddAfterFlow(true, FlowProcessor)
	flow.SetDefaultConfig(&config)
	workflow := flow.RegisterFlow("TestMergeDefaultProcessConfig")
	workflow.AddBeforeFlow(true, FlowProcessor)
	workflow.AddAfterFlow(true, FlowProcessor)
	process := workflow.AddProcess("TestMergeDefaultProcessConfig", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddBeforeStep(true, PreProcessor)
	process.AddAfterStep(true, PostProcessor)
	process.AddBeforeProcess(true, ProcProcessor)
	process.AddAfterProcess(true, ProcProcessor)
	features := flow.DoneFlow("TestMergeDefaultProcessConfig", nil)
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 18 {
		t.Errorf("execute 18 step, but current = %d", current)
	}
}

func TestUnableDefaultProcessConfig(t *testing.T) {
	defer resetCurrent()
	defer func() {
		flow.SetDefaultConfig(nil)
	}()
	config := flow.Configuration{}
	config.AddBeforeStep(true, PreProcessor)
	config.AddAfterStep(true, PostProcessor)
	config.AddBeforeProcess(true, ProcProcessor)
	config.AddAfterProcess(true, ProcProcessor)
	config.AddBeforeFlow(true, FlowProcessor)
	config.AddAfterFlow(true, FlowProcessor)
	flow.SetDefaultConfig(&config)
	workflow := flow.RegisterFlow("TestUnableDefaultProcessConfig")
	workflow.NotUseDefault()
	process := workflow.AddProcess("TestUnableDefaultProcessConfig", nil)
	process.AddStepWithAlias("1", GenerateStep(1))
	process.AddStepWithAlias("2", GenerateStep(2), "1")
	process.AddStepWithAlias("3", GenerateStep(3), "2")
	process.AddStepWithAlias("4", GenerateStep(4), "3")
	process.NotUseDefault()
	features := flow.DoneFlow("TestUnableDefaultProcessConfig", nil)
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}
