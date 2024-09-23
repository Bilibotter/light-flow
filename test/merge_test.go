package test

import (
	"github.com/Bilibotter/light-flow/flow"
	"strings"
	"testing"
)

func TestDependMergedWithEmptyHead(t *testing.T) {
	defer resetCurrent()
	workflow1 := flow.RegisterFlow("TestDependMergedWithEmptyHead1")
	process1 := workflow1.Process("TestDependMergedWithEmptyHead1")
	fn := Fn(t)
	process1.CustomStep(fn.SetCtx0("1").Step(), "1")
	process1.CustomStep(fn.SetCtx0("2").Step(), "2", "1")
	process1.CustomStep(fn.SetCtx0("3").Step(), "3", "2")
	process1.CustomStep(fn.SetCtx0("4").Step(), "4", "3")
	workflow1.AfterFlow(false, CheckResult(t, 4, flow.Success))
	workflow2 := flow.RegisterFlow("TestDependMergedWithEmptyHead2")
	process2 := workflow2.Process("TestDependMergedWithEmptyHead2")
	process2.Merge("TestDependMergedWithEmptyHead1")
	f := fn.Do(CheckCtx0("1", "1"), CheckCtx0("2", "2"), CheckCtx0("3", "3"), CheckCtx0("4", "4"))
	process2.CustomStep(f.Step(), "5", "4")
	workflow2.AfterStep(false, ErrorResultPrinter)
	workflow2.AfterFlow(false, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestDependMergedWithEmptyHead2", nil)
	resetCurrent()
	flow.DoneFlow("TestDependMergedWithEmptyHead1", nil)
}

func TestDependMergedWithNotEmptyHead(t *testing.T) {
	defer resetCurrent()
	workflow1 := flow.RegisterFlow("TestDependMergedWithNotEmptyHead1")
	process1 := workflow1.Process("TestDependMergedWithNotEmptyHead1")
	process1.CustomStep(Fn(t).Do(SetCtx()).Step(), "1")
	process1.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "2", "1")
	process1.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "3", "2")
	process1.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "4", "3")
	workflow1.AfterFlow(false, CheckResult(t, 4, flow.Success))
	workflow2 := flow.RegisterFlow("TestDependMergedWithNotEmptyHead2")
	process2 := workflow2.Process("TestDependMergedWithNotEmptyHead2")
	process2.CustomStep(Fn(t).Do(SetCtx0("+")).Step(), "0")
	process2.Merge("TestDependMergedWithNotEmptyHead1")
	process2.CustomStep(Fn(t).Do(CheckCtx0("0", "+"), CheckCtx("4")).Step(), "5", "0", "4")
	workflow2.AfterStep(false, ErrorResultPrinter)
	workflow2.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestDependMergedWithNotEmptyHead2", nil)
	resetCurrent()
	flow.DoneFlow("TestDependMergedWithNotEmptyHead1", nil)
}

func TestMergeEmpty(t *testing.T) {
	defer resetCurrent()
	wf1 := flow.RegisterFlow("TestMergeEmpty1")
	wf1.Process("TestMergeEmpty1")
	wf1.AfterFlow(false, CheckResult(t, 0, flow.Success))
	wf2 := flow.RegisterFlow("TestMergeEmpty2")
	process2 := wf2.Process("TestMergeEmpty2")
	process2.CustomStep(GenerateStep(1), "1")
	process2.CustomStep(GenerateStep(2), "2", "1")
	process2.CustomStep(GenerateStep(3), "3", "2")
	process2.CustomStep(GenerateStep(4), "4", "3")
	process2.Merge("TestMergeEmpty1")
	wf2.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestMergeEmpty2", nil)
	resetCurrent()
	flow.DoneFlow("TestMergeEmpty1", nil)
}

func TestEmptyMerge(t *testing.T) {
	defer resetCurrent()
	wf1 := flow.RegisterFlow("TestEmptyMerge1")
	process1 := wf1.Process("TestEmptyMerge1")
	process1.CustomStep(Fx[flow.Step](t).SetCtx().Step(), "1")
	process1.CustomStep(Fx[flow.Step](t).CheckDepend().SetCtx().Step(), "2", "1")
	process1.CustomStep(Fx[flow.Step](t).CheckDepend().SetCtx().Step(), "3", "2")
	process1.CustomStep(Fx[flow.Step](t).CheckDepend().SetCtx().Step(), "4", "3")
	wf1.AfterFlow(false, CheckResult(t, 7, flow.Success))

	wf2 := flow.RegisterFlow("TestEmptyMerge2")
	process2 := wf2.Process("TestEmptyMerge2")
	process2.Merge("TestEmptyMerge1")
	wf2.AfterFlow(false, CheckResult(t, 7, flow.Success))
	flow.DoneFlow("TestEmptyMerge2", nil)
	resetCurrent()
	flow.DoneFlow("TestEmptyMerge1", nil)
}

func TestMergeAbsolutelyDifferent(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeAbsolutelyDifferent1")
	process1 := factory1.Process("TestMergeAbsolutelyDifferent1")
	process1.CustomStep(GenerateStep(1), "1")
	process1.CustomStep(GenerateStep(2), "2", "1")
	process1.CustomStep(GenerateStep(3), "3", "2")
	process1.CustomStep(GenerateStep(4), "4", "3")
	factory1.AfterFlow(false, CheckResult(t, 4, flow.Success))
	factory2 := flow.RegisterFlow("TestMergeAbsolutelyDifferent2")
	process2 := factory2.Process("TestMergeAbsolutelyDifferent2")
	process2.CustomStep(GenerateStep(11), "11")
	process2.CustomStep(GenerateStep(12), "12", "11")
	process2.CustomStep(GenerateStep(13), "13", "12")
	process2.CustomStep(GenerateStep(14), "14", "13")
	process2.Merge("TestMergeAbsolutelyDifferent1")
	factory2.AfterFlow(false, CheckResult(t, 8, flow.Success))
	flow.DoneFlow("TestMergeAbsolutelyDifferent2", nil)
	resetCurrent()
	flow.DoneFlow("TestMergeAbsolutelyDifferent1", nil)
}

func TestMergeLayerSame(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeLayerSame1")
	process1 := factory1.Process("TestMergeLayerSame1")
	process1.CustomStep(GenerateStep(1), "1")
	process1.CustomStep(GenerateStep(2), "2", "1")
	process1.CustomStep(GenerateStep(3), "3", "1")
	process1.CustomStep(GenerateStep(4), "4", "3")
	factory1.AfterFlow(false, CheckResult(t, 4, flow.Success))
	factory2 := flow.RegisterFlow("TestMergeLayerSame2")
	process2 := factory2.Process("TestMergeLayerSame2")
	process2.CustomStep(GenerateStep(1), "1")
	process2.CustomStep(GenerateStep(3), "3", "1")
	process2.CustomStep(GenerateStep(4), "4", "3")
	process2.CustomStep(GenerateStep(5), "5", "4")
	process2.Merge("TestMergeLayerSame1")
	factory2.AfterFlow(false, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestMergeLayerSame2", nil)
	resetCurrent()
	flow.DoneFlow("TestMergeLayerSame1", nil)
}

func TestMergeLayerDec(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeLayerDec1")
	process1 := factory1.Process("TestMergeLayerDec1")
	process1.CustomStep(GenerateStep(1), "1")
	process1.CustomStep(GenerateStep(2), "2", "1")
	process1.CustomStep(GenerateStep(3), "3", "1")
	process1.CustomStep(GenerateStep(4), "4", "3")
	factory1.AfterFlow(false, CheckResult(t, 4, flow.Success))
	factory2 := flow.RegisterFlow("TestMergeLayerDec2")
	process2 := factory2.Process("TestMergeLayerDec2")
	process2.CustomStep(GenerateStep(1), "1")
	process2.CustomStep(GenerateStep(4), "4", "1")
	process2.CustomStep(GenerateStep(5), "5", "4")
	process2.CustomStep(GenerateStep(6), "6", "5")
	process2.Merge("TestMergeLayerDec1")
	factory2.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestMergeLayerDec2", nil)
	resetCurrent()
	flow.DoneFlow("TestMergeLayerDec1", nil)
}

func TestMergeLayerInc(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeLayerInc1")
	process1 := factory1.Process("TestMergeLayerInc1")
	process1.CustomStep(GenerateStep(1), "1")
	process1.CustomStep(GenerateStep(2), "2", "1")
	process1.CustomStep(GenerateStep(3), "3", "1")
	process1.CustomStep(GenerateStep(4), "4", "3")
	factory1.AfterFlow(false, CheckResult(t, 4, flow.Success))
	factory2 := flow.RegisterFlow("TestMergeLayerInc2")
	process2 := factory2.Process("TestMergeLayerInc2")
	process2.CustomStep(GenerateStep(1), "1")
	process2.CustomStep(GenerateStep(3), "3", "1")
	process2.CustomStep(GenerateStep(5), "5", "3")
	process2.Merge("TestMergeLayerInc1")
	factory2.AfterFlow(false, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestMergeLayerInc2", nil)
	resetCurrent()
	flow.DoneFlow("TestMergeLayerInc1", nil)
}

func TestMergeSomeSame(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeSomeSame1")
	process1 := factory1.Process("TestMergeSomeSame1")
	process1.CustomStep(GenerateStep(1), "1")
	process1.CustomStep(GenerateStep(2), "2", "1")
	process1.CustomStep(GenerateStep(3), "3", "1")
	process1.CustomStep(GenerateStep(4), "4", "3")
	factory1.AfterFlow(false, CheckResult(t, 4, flow.Success))
	factory2 := flow.RegisterFlow("TestMergeSomeSame2")
	process2 := factory2.Process("TestMergeSomeSame2")
	process2.CustomStep(GenerateStep(1), "1")
	process2.CustomStep(GenerateStep(5), "5", "1")
	process2.CustomStep(GenerateStep(3), "3", "5")
	process2.CustomStep(GenerateStep(4), "4", "5")
	process2.Merge("TestMergeSomeSame1")
	factory2.AfterFlow(false, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestMergeSomeSame2", nil)
	resetCurrent()
	flow.DoneFlow("TestMergeSomeSame1", nil)
}

func TestMergeSame(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeSame1")
	process1 := factory1.Process("TestMergeSame1")
	process1.CustomStep(GenerateStep(1), "1")
	process1.CustomStep(GenerateStep(2), "2", "1")
	process1.CustomStep(GenerateStep(3), "3", "2")
	process1.CustomStep(GenerateStep(4), "4", "3")
	factory1.AfterFlow(false, CheckResult(t, 4, flow.Success))
	factory2 := flow.RegisterFlow("TestMergeSame2")
	process2 := factory2.Process("TestMergeSame2")
	process2.CustomStep(GenerateStep(1), "1")
	process2.CustomStep(GenerateStep(2), "2", "1")
	process2.CustomStep(GenerateStep(3), "3", "2")
	process2.CustomStep(GenerateStep(4), "4", "3")
	process2.Merge("TestMergeSame1")
	factory2.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestMergeSame2", nil)
	resetCurrent()
	flow.DoneFlow("TestMergeSame1", nil)
}

func TestMergeCircle(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeCircle1")
	process1 := factory1.Process("TestMergeCircle1")
	process1.CustomStep(GenerateStep(1), "1")
	process1.CustomStep(GenerateStep(2), "2", "1")
	factory2 := flow.RegisterFlow("TestMergeCircle2")
	process2 := factory2.Process("TestMergeCircle2")
	process2.CustomStep(GenerateStep(2), "2")
	process2.CustomStep(GenerateStep(1), "1", "2")
	defer func() {
		if r := recover(); r != nil {
			if strings.Contains(r.(string), "circle") {
				t.Logf("circle detect success info: %v", r)
			} else {
				t.Errorf("circle detect fail")
			}
		} else {
			t.Errorf("circle detect fail")
		}
	}()
	process2.Merge("TestMergeCircle1")
}

func TestMergeLongCircleLongCircle(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeLongCircle1")
	process1 := factory1.Process("TestMergeLongCircle1")
	process1.CustomStep(GenerateStep(1), "1")
	process1.CustomStep(GenerateStep(2), "2", "1")
	process1.CustomStep(GenerateStep(3), "3", "2")
	process1.CustomStep(GenerateStep(4), "4", "3")
	process1.CustomStep(GenerateStep(5), "5", "4")
	factory2 := flow.RegisterFlow("TestMergeLongCircle2")
	process2 := factory2.Process("TestMergeLongCircle2")
	process2.CustomStep(GenerateStep(2), "2")
	process2.CustomStep(GenerateStep(5), "5", "2")
	process2.CustomStep(GenerateStep(3), "3", "5")
	process2.CustomStep(GenerateStep(4), "4", "3")
	defer func() {
		if r := recover(); r != nil {
			if strings.Contains(r.(string), "circle") {
				t.Logf("circle detect success info: %v", r)
			} else {
				t.Errorf("circle detect fail")
			}
		} else {
			t.Errorf("circle detect fail")
		}
	}()
	process2.Merge("TestMergeLongCircle1")
}

func TestAddWaitBeforeAfterMergeWithCircle(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestAddWaitBeforeAfterMergeWithCircle1")
	process1 := factory1.Process("TestAddWaitBeforeAfterMergeWithCircle1")
	process1.CustomStep(GenerateStep(1), "1")
	process1.CustomStep(GenerateStep(2), "2", "1")
	process1.CustomStep(GenerateStep(3), "3", "2")
	process1.CustomStep(GenerateStep(4), "4", "3")
	process1.CustomStep(GenerateStep(5), "5", "4")
	factory2 := flow.RegisterFlow("TestAddWaitBeforeAfterMergeWithCircle2")
	process2 := factory2.Process("TestAddWaitBeforeAfterMergeWithCircle2")
	tmp := process2.CustomStep(GenerateStep(5), "5")
	process2.Merge("TestAddWaitBeforeAfterMergeWithCircle1")
	defer func() {
		if r := recover(); r != nil {
			if strings.Contains(r.(string), "circle") {
				t.Logf("circle detect success info: %v", r)
			} else {
				t.Errorf("circle detect fail")
			}
		} else {
			t.Errorf("circle detect fail")
		}
	}()
	tmp.Next(GenerateStep(2), "2")
	return
}

func TestAddWaitAllAfterMergeWithCircle(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestAddWaitAllAfterMergeWithCircle1")
	process1 := factory1.Process("TestAddWaitAllAfterMergeWithCircle1")
	process1.CustomStep(GenerateStep(1), "1")
	process1.CustomStep(GenerateStep(2), "2", "1")
	process1.CustomStep(GenerateStep(3), "3", "2")
	process1.CustomStep(GenerateStep(4), "4", "3")
	process1.CustomStep(GenerateStep(5), "5", "4")
	factory2 := flow.RegisterFlow("TestAddWaitAllAfterMergeWithCircle2")
	process2 := factory2.Process("TestAddWaitAllAfterMergeWithCircle2")
	process2.CustomStep(GenerateStep(5), "5")
	process2.Merge("TestAddWaitAllAfterMergeWithCircle1")
	defer func() {
		if r := recover(); r != nil {
			if strings.Contains(r.(string), "circle") {
				t.Logf("circle detect success info: %v", r)
			} else {
				t.Errorf("circle detect fail")
			}
		} else {
			t.Errorf("circle detect fail")
		}
	}()
	process2.SyncAll(GenerateStep(2), "2")
	return
}

func TestSerialMerge(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestSerialMerge1")
	process := workflow.Process("TestSerialMerge1")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "1")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "2")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "3")
	process.Follow(NormalStep1).After("1", "2", "3")

	workflow = flow.RegisterFlow("TestSerialMerge2")
	process = workflow.Process("TestSerialMerge2")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "1")
	process.Merge("TestSerialMerge1")
	process.Follow(NormalStep1, NormalStep2, NormalStep3).After("1")
	workflow.AfterStep(true, CheckCurrent(t, 4)).OnlyFor("NormalStep1")
	workflow.AfterStep(true, CheckCurrent(t, 5)).OnlyFor("NormalStep2")
	workflow.AfterStep(true, CheckCurrent(t, 6)).OnlyFor("NormalStep3")
	workflow.AfterFlow(true, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestSerialMerge2", nil)

	resetCurrent()
	workflow = flow.RegisterFlow("TestSerialMerge3")
	process = workflow.Process("TestSerialMerge3")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "1")
	process.Follow(NormalStep1, NormalStep2, NormalStep3).After("1")
	process.Merge("TestSerialMerge1")
	workflow.AfterStep(true, CheckCurrent(t, 4)).OnlyFor("NormalStep1")
	workflow.AfterStep(true, CheckCurrent(t, 5)).OnlyFor("NormalStep2")
	workflow.AfterStep(true, CheckCurrent(t, 6)).OnlyFor("NormalStep3")
	workflow.AfterFlow(true, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestSerialMerge3", nil)

	resetCurrent()
	ff := flow.DoneFlow("TestSerialMerge1", nil)
	CheckResult(t, 4, flow.Success)(any(ff).(flow.WorkFlow))
}

func TestParallelMerge(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestParallelMerge1")
	process := workflow.Process("TestParallelMerge1")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "1")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "2")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "3")
	process.Follow(NormalStep1).After("1", "2", "3")

	workflow = flow.RegisterFlow("TestParallelMerge2")
	process = workflow.Process("TestParallelMerge2")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "1")
	process.Merge("TestParallelMerge1")
	process.Parallel(NormalStep1).After("1")
	workflow.AfterStep(true, CheckCurrent(t, 4)).OnlyFor("NormalStep1")
	workflow.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestParallelMerge2", nil)

	resetCurrent()
	workflow = flow.RegisterFlow("TestParallelMerge3")
	process = workflow.Process("TestParallelMerge3")
	process.CustomStep(Fx[flow.Step](t).Inc().Step(), "1")
	process.Parallel(NormalStep1).After("1")
	process.Merge("TestParallelMerge1")
	workflow.AfterStep(true, CheckCurrent(t, 4)).OnlyFor("NormalStep1")
	workflow.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestParallelMerge3", nil)

	resetCurrent()
	ff := flow.DoneFlow("TestParallelMerge1", nil)
	CheckResult(t, 4, flow.Success)(any(ff).(flow.WorkFlow))
}
