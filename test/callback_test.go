package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"sync/atomic"
)

func AfterFlowProcessor(info flow.WorkFlow) (bool, error) {
	if info.Name() == "" {
		panic("flow name is empty")
	}
	if len(info.ID()) == 0 {
		panic("flow id is empty")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..process[%s] AfterFlowProcessor execute \n", info.Name())
	return true, nil
}

func BeforeFlowProcessor(info flow.WorkFlow) (bool, error) {
	if info.Name() == "" {
		panic("flow name is empty")
	}
	if len(info.ID()) == 0 {
		panic("flow id is empty")
	}
	atomic.AddInt64(&current, 1)
	fmt.Printf("..process[%s] BeforeFlowProcessor execute \n", info.Name())
	return true, nil
}

//func TestInfoCorrect(t *testing.T) {
//	defer resetCurrent()
//	defer func() {
//		flow.CreateDefaultConfig()
//	}()
//	config := flow.CreateDefaultConfig()
//	config.BeforeStep(true, StepInfoChecker("1", []string{}, []string{"2", "3"}))
//	config.BeforeStep(true, StepInfoChecker("2", []string{"1"}, []string{"4"}))
//	config.BeforeStep(true, StepInfoChecker("3", []string{"1"}, []string{"4"}))
//	config.BeforeStep(true, StepInfoChecker("4", []string{"2", "3"}, []string{}))
//	config.AfterStep(true, StepInfoChecker("1", []string{}, []string{"2", "3"}))
//	config.AfterStep(true, StepInfoChecker("2", []string{"1"}, []string{"4"}))
//	config.AfterStep(true, StepInfoChecker("3", []string{"1"}, []string{"4"}))
//	config.AfterStep(true, StepInfoChecker("4", []string{"2", "3"}, []string{}))
//
//	workflow := flow.RegisterFlow("TestInfoCorrect")
//	process := workflow.Process("TestInfoCorrect")
//	process.NameStep(GenerateStep(1), "1")
//	process.NameStep(GenerateStep(2), "2", "1")
//	process.NameStep(GenerateStep(3), "3", "1")
//	process.NameStep(GenerateStep(4), "4", "2", "3")
//	workflow.AfterFlow(false, CheckResult(t, 12, flow.Success))
//	flow.DoneFlow("TestInfoCorrect", nil)
//}
//
//func TestDefaultProcessConfig(t *testing.T) {
//	defer resetCurrent()
//	defer func() {
//		flow.CreateDefaultConfig()
//	}()
//	config := flow.CreateDefaultConfig()
//	config.BeforeStep(true, PreProcessor)
//	config.AfterStep(true, PostProcessor)
//	config.BeforeProcess(true, ProcProcessor)
//	config.AfterProcess(true, ProcProcessor)
//	config.BeforeFlow(true, BeforeFlowProcessor)
//	config.AfterFlow(true, AfterFlowProcessor)
//
//	workflow := flow.RegisterFlow("TestDefaultProcessConfig")
//	process := workflow.Process("TestDefaultProcessConfig")
//	process.NameStep(GenerateStep(1), "1")
//	process.NameStep(GenerateStep(2), "2", "1")
//	process.NameStep(GenerateStep(3), "3", "2")
//	process.NameStep(GenerateStep(4), "4", "3")
//	workflow.AfterFlow(false, CheckResult(t, 16, flow.Success))
//	flow.DoneFlow("TestDefaultProcessConfig", nil)
//}
//
//func TestMergeDefaultProcessConfig(t *testing.T) {
//	defer resetCurrent()
//	defer func() {
//		flow.CreateDefaultConfig()
//	}()
//	config := flow.CreateDefaultConfig()
//	config.BeforeStep(true, PreProcessor)
//	config.AfterStep(true, PostProcessor)
//	config.BeforeProcess(true, ProcProcessor)
//	config.AfterProcess(true, ProcProcessor)
//	config.BeforeFlow(true, BeforeFlowProcessor)
//	config.AfterFlow(true, AfterFlowProcessor)
//
//	workflow := flow.RegisterFlow("TestMergeDefaultProcessConfig")
//	workflow.BeforeFlow(true, BeforeFlowProcessor)
//	workflow.AfterFlow(true, AfterFlowProcessor)
//	process := workflow.Process("TestMergeDefaultProcessConfig")
//	process.NameStep(GenerateStep(1), "1")
//	process.NameStep(GenerateStep(2), "2", "1")
//	process.BeforeStep(true, PreProcessor)
//	process.AfterStep(true, PostProcessor)
//	process.BeforeProcess(true, ProcProcessor)
//	process.AfterProcess(true, ProcProcessor)
//	workflow.AfterFlow(false, CheckResult(t, 18, flow.Success))
//	flow.DoneFlow("TestMergeDefaultProcessConfig", nil)
//}
//
//func TestCallbackCond(t *testing.T) {
//	defer resetCurrent()
//	defer func() {
//		flow.CreateDefaultConfig()
//	}()
//	config := flow.CreateDefaultConfig()
//	config.BeforeFlow(true, BeforeFlowProcessor).OnlyFor("TestCallbackCond")
//	config.BeforeFlow(true, BeforeFlowProcessor).OnlyFor("TestCallbackCondNotExist")
//	config.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").When(flow.Failed)
//	config.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").When(flow.Success)
//	config.BeforeProcess(true, ProcProcessor).OnlyFor("TestCallbackCond")
//	config.BeforeProcess(true, ProcProcessor).OnlyFor("TestCallbackCondNotExist")
//	config.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond").When(flow.Failed)
//	config.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond").When(flow.Success)
//	config.BeforeStep(true, PreProcessor).OnlyFor("1")
//	config.BeforeStep(true, PreProcessor).OnlyFor("NotExist")
//	config.AfterStep(true, PostProcessor).OnlyFor("1").When(flow.Success)
//	config.AfterStep(true, PostProcessor).OnlyFor("1").When(flow.Failed)
//
//	workflow := flow.RegisterFlow("TestCallbackCond")
//	process := workflow.Process("TestCallbackCond")
//	process.NameStep(GenerateStep(1), "1")
//	process.NameStep(GenerateStep(2), "2", "1")
//	workflow.AfterFlow(false, CheckResult(t, 8, flow.Success))
//	flow.DoneFlow("TestCallbackCond", nil)
//}
//
//func TestCallbackCond0(t *testing.T) {
//	defer resetCurrent()
//	defer func() {
//		flow.CreateDefaultConfig()
//	}()
//	config := flow.CreateDefaultConfig()
//	config.BeforeFlow(true, BeforeFlowProcessor).NotFor("TestCallbackCond0")
//	config.BeforeFlow(true, BeforeFlowProcessor).NotFor("TestCallbackCondNotExist")
//	config.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Failed)
//	config.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").Exclude(flow.Success)
//	config.BeforeProcess(true, ProcProcessor).NotFor("TestCallbackCond0")
//	config.BeforeProcess(true, ProcProcessor).NotFor("TestCallbackCondNotExist")
//	config.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Failed)
//	config.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Success)
//	config.BeforeStep(true, PreProcessor).NotFor("1")
//	config.AfterStep(true, PostProcessor).OnlyFor("1").Exclude(flow.Success)
//	config.AfterStep(true, PostProcessor).OnlyFor("1").Exclude(flow.Failed)
//
//	workflow := flow.RegisterFlow("TestCallbackCond0")
//	process := workflow.Process("TestCallbackCond0")
//	process.NameStep(GenerateStep(1), "1")
//	process.NameStep(GenerateStep(2), "2", "1")
//	workflow.AfterFlow(false, CheckResult(t, 8, flow.Success))
//	flow.DoneFlow("TestCallbackCond0", nil)
//}
//
//func TestUnableDefaultProcessConfig(t *testing.T) {
//	defer resetCurrent()
//	defer func() {
//		flow.CreateDefaultConfig()
//	}()
//	config := flow.CreateDefaultConfig()
//	config.BeforeStep(true, PreProcessor)
//	config.AfterStep(true, PostProcessor)
//	config.BeforeProcess(true, ProcProcessor)
//	config.AfterProcess(true, ProcProcessor)
//	config.BeforeFlow(true, BeforeFlowProcessor)
//	config.AfterFlow(true, AfterFlowProcessor)
//
//	workflow := flow.RegisterFlow("TestUnableDefaultProcessConfig")
//	workflow.NoUseDefault()
//	process := workflow.Process("TestUnableDefaultProcessConfig")
//	process.NameStep(GenerateStep(1), "1")
//	process.NameStep(GenerateStep(2), "2", "1")
//	process.NameStep(GenerateStep(3), "3", "2")
//	process.NameStep(GenerateStep(4), "4", "3")
//	process.NotUseDefault()
//	workflow.AfterFlow(false, CheckResult(t, 4, flow.Success))
//	flow.DoneFlow("TestUnableDefaultProcessConfig", nil)
//}
