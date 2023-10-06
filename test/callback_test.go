package test

import (
	flow "gitee.com/MetaphysicCoding/light-flow"
	"sync/atomic"
	"testing"
)

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
	if atomic.LoadInt64(&current) != 14 {
		t.Errorf("execute 14 step, but current = %d", current)
	}
}
