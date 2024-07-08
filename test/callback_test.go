package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"sync/atomic"
	"testing"
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

func TestInfoCorrect(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	dc := flow.DefaultCallback()
	dc.BeforeStep(true, Ck(t, "Step1").Exclude("Step2", "Step3", "Step4").Check())
	dc.BeforeStep(true, Ck(t, "Step2").Contain("Step1").Exclude("Step3", "Step4").Check())
	dc.BeforeStep(true, Ck(t, "Step3").Contain("Step1").Exclude("Step2", "Step4").Check())
	dc.BeforeStep(true, Ck(t, "Step4").Contain("Step1", "Step2", "Step3").Check())

	dc.AfterStep(true, Ck(t, "Step1").Exclude("Step2", "Step3", "Step4").Check())
	dc.AfterStep(true, Ck(t, "Step2").Contain("Step1").Exclude("Step3", "Step4").Check())
	dc.AfterStep(true, Ck(t, "Step3").Contain("Step1").Exclude("Step2", "Step4").Check())
	dc.AfterStep(true, Ck(t, "Step4").Contain("Step1", "Step2", "Step3").Check())

	workflow := flow.RegisterFlow("TestInfoCorrect")
	process := workflow.Process("TestInfoCorrect")
	process.NameStep(Ck(t).SetFn(), "Step1")
	process.NameStep(Ck(t).SetFn(), "Step2", "Step1")
	process.NameStep(Ck(t).SetFn(), "Step3", "Step1")
	process.NameStep(Ck(t).SetFn(), "Step4", "Step2", "Step3")
	workflow.AfterFlow(false, CheckResult(t, 12, flow.Success))
	flow.DoneFlow("TestInfoCorrect", nil)
}

func TestDefaultProcessConfig(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	df := flow.DefaultCallback()
	df.BeforeStep(true, PreProcessor)
	df.AfterStep(true, PostProcessor)
	df.BeforeProcess(true, ProcProcessor)
	df.AfterProcess(true, ProcProcessor)
	df.BeforeFlow(true, BeforeFlowProcessor)
	df.AfterFlow(true, AfterFlowProcessor)

	workflow := flow.RegisterFlow("TestDefaultProcessConfig")
	process := workflow.Process("TestDefaultProcessConfig")
	process.NameStep(Fn(t).Normal(), "1")
	process.NameStep(Fn(t).Normal(), "2", "1")
	process.NameStep(Fn(t).Normal(), "3", "2")
	process.NameStep(Fn(t).Normal(), "4", "3")
	workflow.AfterFlow(false, CheckResult(t, 16, flow.Success))
	flow.DoneFlow("TestDefaultProcessConfig", nil)
}

func TestMergeDefaultProcessConfig(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	df := flow.DefaultCallback()
	df.BeforeStep(true, PreProcessor)
	df.AfterStep(true, PostProcessor)
	df.BeforeProcess(true, ProcProcessor)
	df.AfterProcess(true, ProcProcessor)
	df.BeforeFlow(true, BeforeFlowProcessor)
	df.AfterFlow(true, AfterFlowProcessor)

	workflow := flow.RegisterFlow("TestMergeDefaultProcessConfig")
	workflow.BeforeFlow(true, BeforeFlowProcessor)
	workflow.AfterFlow(true, AfterFlowProcessor)
	process := workflow.Process("TestMergeDefaultProcessConfig")
	process.NameStep(GenerateStep(1), "1")
	process.NameStep(GenerateStep(2), "2", "1")
	process.BeforeStep(true, PreProcessor)
	process.AfterStep(true, PostProcessor)
	process.BeforeProcess(true, ProcProcessor)
	process.AfterProcess(true, ProcProcessor)
	workflow.AfterFlow(false, CheckResult(t, 18, flow.Success))
	flow.DoneFlow("TestMergeDefaultProcessConfig", nil)
}

func TestCallbackCond(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	df := flow.DefaultCallback()
	df.BeforeFlow(true, BeforeFlowProcessor).OnlyFor("TestCallbackCond")
	df.BeforeFlow(true, BeforeFlowProcessor).OnlyFor("TestCallbackCondNotExist")
	df.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").When(flow.Failed)
	df.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").When(flow.Success)
	df.BeforeProcess(true, ProcProcessor).OnlyFor("TestCallbackCond")
	df.BeforeProcess(true, ProcProcessor).OnlyFor("TestCallbackCondNotExist")
	df.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond").When(flow.Failed)
	df.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond").When(flow.Success)
	df.BeforeStep(true, PreProcessor).OnlyFor("1")
	df.BeforeStep(true, PreProcessor).OnlyFor("NotExist")
	df.AfterStep(true, PostProcessor).OnlyFor("1").When(flow.Success)
	df.AfterStep(true, PostProcessor).OnlyFor("1").When(flow.Failed)

	workflow := flow.RegisterFlow("TestCallbackCond")
	process := workflow.Process("TestCallbackCond")
	process.NameStep(GenerateStep(1), "1")
	process.NameStep(GenerateStep(2), "2", "1")
	workflow.AfterFlow(false, CheckResult(t, 8, flow.Success))
	flow.DoneFlow("TestCallbackCond", nil)
}

func TestCallbackCond0(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	df := flow.DefaultCallback()
	df.BeforeFlow(true, BeforeFlowProcessor).NotFor("TestCallbackCond0")
	df.BeforeFlow(true, BeforeFlowProcessor).NotFor("TestCallbackCondNotExist")
	df.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Failed)
	df.AfterFlow(true, AfterFlowProcessor).OnlyFor("TestCallbackCond").Exclude(flow.Success)
	df.BeforeProcess(true, ProcProcessor).NotFor("TestCallbackCond0")
	df.BeforeProcess(true, ProcProcessor).NotFor("TestCallbackCondNotExist")
	df.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Failed)
	df.AfterProcess(true, AfterProcProcessor).OnlyFor("TestCallbackCond0").Exclude(flow.Success)
	df.BeforeStep(true, PreProcessor).NotFor("1")
	df.AfterStep(true, PostProcessor).OnlyFor("1").Exclude(flow.Success)
	df.AfterStep(true, PostProcessor).OnlyFor("1").Exclude(flow.Failed)

	workflow := flow.RegisterFlow("TestCallbackCond0")
	process := workflow.Process("TestCallbackCond0")
	process.NameStep(GenerateStep(1), "1")
	process.NameStep(GenerateStep(2), "2", "1")
	workflow.AfterFlow(false, CheckResult(t, 8, flow.Success))
	flow.DoneFlow("TestCallbackCond0", nil)
}

func TestUnableDefaultProcessConfig(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	df := flow.DefaultCallback()
	df.BeforeStep(true, PreProcessor)
	df.AfterStep(true, PostProcessor)
	df.BeforeProcess(true, ProcProcessor)
	df.AfterProcess(true, ProcProcessor)
	df.BeforeFlow(true, BeforeFlowProcessor)
	df.AfterFlow(true, AfterFlowProcessor)

	workflow := flow.RegisterFlow("TestUnableDefaultProcessConfig")
	workflow.DisableDefaultCallback()
	process := workflow.Process("TestUnableDefaultProcessConfig")
	process.NameStep(GenerateStep(1), "1")
	process.NameStep(GenerateStep(2), "2", "1")
	process.NameStep(GenerateStep(3), "3", "2")
	process.NameStep(GenerateStep(4), "4", "3")
	process.DisableDefaultCallback()
	workflow.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestUnableDefaultProcessConfig", nil)
}
