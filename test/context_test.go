package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var (
	ctx1    = int64(0)
	ctx2    = int64(0)
	ctx3    = int64(0)
	ctx4    = int64(0)
	ctx5    = int64(0)
	ctx6    = int64(0)
	addrKey = "addr"
)

func resetCtx() {
	atomic.StoreInt64(&ctx1, 0)
	atomic.StoreInt64(&ctx2, 0)
	atomic.StoreInt64(&ctx3, 0)
	atomic.StoreInt64(&ctx4, 0)
	atomic.StoreInt64(&ctx5, 0)
	atomic.StoreInt64(&ctx6, 0)
}

func GenerateSleep(duration time.Duration, args ...any) func(ctx flow.StepCtx) (any, error) {
	return func(ctx flow.StepCtx) (any, error) {
		addrWrap, ok := ctx.Get(addrKey)
		if !ok {
			panic(fmt.Sprintf("%s not found addr", ctx.ContextName()))
		}
		atomic.AddInt64(addrWrap.(*int64), 1)
		return nil, nil
	}
}

func ChangeCtxStepFunc(addr *int64, args ...any) func(ctx flow.StepCtx) (any, error) {
	return func(ctx flow.StepCtx) (any, error) {
		if len(args) > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		ctx.Set(addrKey, addr)
		println("change ctx")
		addrWrap, ok := ctx.Get(addrKey)
		if !ok {
			panic(fmt.Sprintf("%s not found addr", ctx.ContextName()))
		}
		atomic.AddInt64(addrWrap.(*int64), 1)
		return nil, nil
	}
}

func GenerateStepIncAddr(i int, args ...any) func(ctx flow.StepCtx) (any, error) {
	return func(ctx flow.StepCtx) (any, error) {
		if len(args) > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Printf("%d.step finish\n", i)
		addrWrap, ok := ctx.Get(addrKey)
		if !ok {
			panic(fmt.Sprintf("%s not found addr", ctx.ContextName()))
		}
		atomic.AddInt64(addrWrap.(*int64), 1)
		return i, nil
	}
}

//func ExposeAddrFunc(addr *int64) func(ctx flow.StepCtx) (any, error) {
//	return func(ctx flow.StepCtx) (any, error) {
//		ctx.Exposed(addrKey, addr)
//		atomic.AddInt64(addr, 1)
//		return nil, nil
//	}
//}

func getUnExist(ctx flow.StepCtx) (any, error) {
	println("start")
	ctx.Get("unExist")
	println("end")
	return nil, nil
}

func invalidUse(ctx flow.StepCtx) (any, error) {
	return nil, nil
}

//func getAllAndSet(value string, history ...string) func(ctx flow.StepCtx) (any, error) {
//	return func(ctx flow.StepCtx) (any, error) {
//		ctx.set("all", value)
//		ks := flow.newRoutineUnsafeSet[string]()
//		vs := flow.newRoutineUnsafeSet[string]()
//		m := ctx.GetAll("all")
//		for k, v := range m {
//			ks.Add(k)
//			vs.Add(v.(string))
//		}
//		for _, v := range history {
//			if !vs.Contains(v) {
//				panic(fmt.Sprintf("ctx[%s] not contain %s", ctx.ContextName(), v))
//			}
//		}
//		fmt.Printf("ctx[%s] get all keys=%s\n", ctx.ContextName(), strings.Join(ks.Slice(), ", "))
//		return nil, nil
//	}
//}

func TestSearch(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestSearch")
	process := workflow.Process("TestSearch1")
	process.NameStep(invalidUse, "1")
	process.NameStep(invalidUse, "2", "1")
	process.NameStep(invalidUse, "3", "1")
	process.NameStep(getUnExist, "4", "2", "3")
	features := flow.DoneFlow("TestSearch", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.GetName(), explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
}

func TestPriorityWithSelf(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestPriorityWithSelf")
	process := workflow.Process("test1")
	process.NameStep(ChangeCtxStepFunc(&ctx1), "0")
	process.NameStep(ChangeCtxStepFunc(&ctx1), "1")
	process.NameStep(GenerateStepIncAddr(1), "2", "1")
	process.NameStep(ChangeCtxStepFunc(&ctx2), "3", "1")
	step := process.NameStep(GenerateStepIncAddr(1), "4", "2", "3")
	defer func() {
		if r := recover(); r != nil {
			t.Logf("%v", r)
		}
	}()
	step.Priority(map[string]any{addrKey: "4"})
}

func TestPriorityCheck(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestPriorityCheck")
	process := workflow.Process("TestPriorityCheck")
	process.NameStep(ChangeCtxStepFunc(&ctx1), "0")
	process.NameStep(ChangeCtxStepFunc(&ctx1), "1")
	process.NameStep(GenerateStepIncAddr(1), "2", "1")
	process.NameStep(ChangeCtxStepFunc(&ctx2), "3", "1")
	step := process.NameStep(GenerateStepIncAddr(1), "4", "2", "3")
	defer func() {
		if r := recover(); r != nil {
			t.Logf("%v", r)
		}
	}()
	step.Priority(map[string]any{addrKey: "0"})
}

func TestPriority(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestPriority")
	process := workflow.Process("TestPriority")
	process.AfterStep(true, ErrorResultPrinter)
	process.NameStep(ChangeCtxStepFunc(&ctx1), "1")
	process.NameStep(GenerateStepIncAddr(1), "2", "1")
	process.NameStep(ChangeCtxStepFunc(&ctx2), "3", "1")
	step := process.NameStep(GenerateStepIncAddr(1), "4", "2", "3")
	step.Priority(map[string]any{addrKey: "3"})
	features := flow.DoneFlow("TestPriority", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.GetName(), explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
	if ctx1 != 2 {
		t.Errorf("execute 2 step, but ctx1 = %d", ctx1)
	}
	if atomic.LoadInt64(&ctx2) != 2 {
		t.Errorf("execute 2 step, but ctx2 = %d", ctx2)
	}
}

//func TestExpose(t *testing.T) {
//	defer resetCtx()
//	workflow := flow.RegisterFlow("TestExpose")
//	process := workflow.Process("TestExpose1", nil)
//	process.AfterStep(true, ErrorResultPrinter)
//	process.NameStep(ExposeAddrFunc(&ctx1), "1")
//	process.NameStep(GenerateStepIncAddr(2,"ms"), "2")
//	process.NameStep(GenerateStepIncAddr(3,"ms"), "3")
//	process = workflow.Process("TestExpose2", nil)
//	process.NameStep(ExposeAddrFunc(&ctx2), "11")
//	process.NameStep(GenerateStepIncAddr(12,"ms"), "12")
//	process.NameStep(GenerateStepIncAddr(13,"ms"), "13")
//	features := flow.DoneFlow("TestExpose", nil)
//	for _, feature := range features.Futures() {
//		explain := strings.Join(feature.ExplainStatus(), ", ")
//		fmt.Printf("process[%s] explain=%s\n", feature.GetName(), explain)
//		if !feature.Success() {
//			t.Errorf("process[%s] fail", feature.GetName())
//		}
//	}
//	if ctx1 != 3 {
//		t.Errorf("execute 3 step, but ctx1 = %d", ctx1)
//	}
//	if atomic.LoadInt64(&ctx2) != 3 {
//		t.Errorf("execute 3 step, but ctx2 = %d", ctx2)
//	}
//}

func TestPtrReuse(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestPtrReuse")
	process1 := workflow.Process("TestPtrReuse1")
	process1.NameStep(GenerateStepIncAddr(1), "1")
	process1.NameStep(GenerateStepIncAddr(2), "2", "1")
	process1.NameStep(GenerateStepIncAddr(3), "3", "2")
	process1 = workflow.Process("TestPtrReuse2")
	process1.NameStep(GenerateStepIncAddr(11), "11")
	process1.NameStep(GenerateStepIncAddr(12), "12", "11")
	process1.NameStep(GenerateStepIncAddr(13), "13", "12")
	features := flow.DoneFlow("TestPtrReuse", map[string]any{addrKey: &ctx1})
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.GetName(), explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
	if ctx1 != 6 {
		t.Errorf("execute 6 step, but ctx1 = %d", ctx1)
	}
}

func TestWaitToDoneInMultiple(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestWaitToDoneInMultiple")
	process1 := workflow.Process("TestWaitToDoneInMultiple1")
	process1.NameStep(GenerateStepIncAddr(1), "1")
	process1.NameStep(GenerateStepIncAddr(2), "2", "1")
	process1.NameStep(GenerateStepIncAddr(3), "3", "2")
	process2 := workflow.Process("TestWaitToDoneInMultiple2")
	process2.NameStep(GenerateStepIncAddr(11), "11")
	process2.NameStep(GenerateStepIncAddr(12), "12", "11")
	process2.NameStep(GenerateStepIncAddr(13), "13", "12")
	process2.NameStep(GenerateSleep(100*time.Millisecond), "14", "13")
	features := flow.DoneFlow("TestWaitToDoneInMultiple", map[string]any{addrKey: &ctx1})
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.GetName(), explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
	if ctx1 != 7 {
		t.Errorf("execute 7 step, but ctx1 = %d", ctx1)
	}
}

func TestWorkFlowCtx(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestWorkFlowCtx")
	process1 := workflow.Process("TestWorkFlowCtx1")
	process1.NameStep(GenerateStepIncAddr(1), "1")
	process1.NameStep(GenerateStepIncAddr(2), "2", "1")
	process1.NameStep(GenerateStepIncAddr(3), "3", "2")
	process2 := workflow.Process("TestWorkFlowCtx2")
	process2.NameStep(GenerateStepIncAddr(11), "11")
	process2.NameStep(GenerateStepIncAddr(12), "12", "11")
	process2.NameStep(GenerateStepIncAddr(13), "13", "12")
	features := flow.DoneFlow("TestWorkFlowCtx", map[string]any{addrKey: &ctx1})
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.GetName(), explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
	if ctx1 != 6 {
		t.Errorf("execute 6 step, but ctx1 = %d", ctx1)
	}
}

func TestProcessAndWorkflowCtx(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestProcessAndWorkflowCtx")
	process := workflow.Process("TestProcessAndWorkflowCtx1")
	process.AfterStep(true, ErrorResultPrinter)
	process.NameStep(GenerateStepIncAddr(1), "1")
	process.NameStep(GenerateStepIncAddr(2), "2", "1")
	process.NameStep(GenerateStepIncAddr(3), "3", "2")
	process = workflow.Process("TestProcessAndWorkflowCtx2")
	process.AfterStep(true, ErrorResultPrinter)
	process.NameStep(GenerateStepIncAddr(11), "11")
	process.NameStep(GenerateStepIncAddr(12), "12", "11")
	process.NameStep(GenerateStepIncAddr(13), "13", "12")
	features := flow.DoneFlow("TestProcessAndWorkflowCtx", map[string]any{addrKey: &ctx1})
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.GetName(), explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
	if ctx1 != 6 {
		t.Errorf("execute 6 step, but ctx2 = %d", ctx2)
	}
}

func TestStepCtx(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestStepCtx")
	process := workflow.Process("TestStepCtx1")
	process.NameStep(ChangeCtxStepFunc(&ctx3), "1")
	process.NameStep(GenerateStepIncAddr(2), "2", "1")
	process.NameStep(GenerateStepIncAddr(3), "3", "2")
	process = workflow.Process("TestStepCtx2")
	process.NameStep(ChangeCtxStepFunc(&ctx4), "11")
	process.NameStep(GenerateStepIncAddr(12), "12", "11")
	process.NameStep(GenerateStepIncAddr(13), "13", "12")
	features := flow.DoneFlow("TestStepCtx", map[string]any{addrKey: &ctx1})
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.GetName(), explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
	if ctx1 != 0 {
		t.Errorf("workflow ctx has effective with duplicate key in step")
	}
	if atomic.LoadInt64(&ctx2) != 0 {
		t.Errorf("produce ctx has effective with duplicate key in step")
	}
	if atomic.LoadInt64(&ctx3) != 3 {
		t.Errorf("execute 3 step, but ctx3 = %d", ctx3)
	}
	if atomic.LoadInt64(&ctx4) != 3 {
		t.Errorf("execute 3 step, but ctx4 = %d", ctx4)
	}
}

func TestDependStepCtx(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestDependStepCtx")
	process := workflow.Process("TestDependStepCtx1")
	process.AfterStep(true, ErrorResultPrinter)
	process.NameStep(ChangeCtxStepFunc(&ctx3), "1")
	process.NameStep(ChangeCtxStepFunc(&ctx5), "2", "1")
	process.NameStep(GenerateStepIncAddr(3), "3", "2")
	process = workflow.Process("TestDependStepCtx2")
	process.NameStep(ChangeCtxStepFunc(&ctx4), "11")
	process.NameStep(ChangeCtxStepFunc(&ctx6), "12", "11")
	process.NameStep(GenerateStepIncAddr(13), "13", "12")
	features := flow.DoneFlow("TestDependStepCtx", map[string]any{addrKey: &ctx1})
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.GetName(), explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.GetName())
		}
	}
	if ctx1 != 0 {
		t.Errorf("workflow ctx has effective with duplicate key in step")
	}
	if atomic.LoadInt64(&ctx2) != 0 {
		t.Errorf("produce ctx has effective with duplicate key in step")
	}
	if atomic.LoadInt64(&ctx3) != 1 {
		t.Errorf("execute 1 step, but ctx3 = %d", ctx3)
	}
	if atomic.LoadInt64(&ctx4) != 1 {
		t.Errorf("execute 1 step, but ctx4 = %d", ctx4)
	}
	if atomic.LoadInt64(&ctx5) != 2 {
		t.Errorf("execute 2 step, but ctx3 = %d", ctx5)
	}
	if atomic.LoadInt64(&ctx6) != 2 {
		t.Errorf("execute 2 step, but ctx4 = %d", ctx6)
	}
}

func TestFlowMultipleAsyncExecute(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestFlowMultipleExecute")
	process := workflow.Process("TestFlowMultipleExecute1")
	process.AfterStep(true, ErrorResultPrinter)
	process.NameStep(GenerateStepIncAddr(1), "1")
	process.NameStep(GenerateStepIncAddr(2), "2", "1")
	process.NameStep(GenerateStepIncAddr(3), "3", "2")
	process = workflow.Process("TestFlowMultipleExecute2")
	process.NameStep(GenerateStepIncAddr(11), "11")
	process.NameStep(GenerateStepIncAddr(12), "12", "11")
	process.NameStep(GenerateStepIncAddr(13), "13", "12")
	flow1 := flow.AsyncFlow("TestFlowMultipleExecute", map[string]any{addrKey: &ctx1})
	flow2 := flow.AsyncFlow("TestFlowMultipleExecute", map[string]any{addrKey: &ctx2})
	flow3 := flow.AsyncFlow("TestFlowMultipleExecute", map[string]any{addrKey: &ctx3})
	flow4 := flow.AsyncFlow("TestFlowMultipleExecute", map[string]any{addrKey: &ctx4})
	flow5 := flow.AsyncFlow("TestFlowMultipleExecute", map[string]any{addrKey: &ctx5})
	for _, flowing := range []flow.FlowController{flow1, flow2, flow3, flow4, flow5} {
		for _, feature := range flowing.Done() {
			explain := strings.Join(feature.ExplainStatus(), ", ")
			fmt.Printf("process[%s] explain=%s\n", feature.GetName(), explain)
			if !feature.Success() {
				t.Errorf("process[%s] fail", feature.GetName())
			}
		}
	}
	if atomic.LoadInt64(&ctx1) != 6 {
		t.Errorf("execute 6 step, but ctx1 = %d", ctx1)
	}
	if atomic.LoadInt64(&ctx2) != 6 {
		t.Errorf("execute 2 step, but ctx2 = %d", ctx3)
	}
	if atomic.LoadInt64(&ctx3) != 6 {
		t.Errorf("execute 2 step, but ctx3 = %d", ctx3)
	}
	if atomic.LoadInt64(&ctx4) != 6 {
		t.Errorf("execute 2 step, but ctx4 = %d", ctx4)
	}
	if atomic.LoadInt64(&ctx5) != 6 {
		t.Errorf("execute 2 step, but ctx5 = %d", ctx5)
	}
}

//func TestGetAll(t *testing.T) {
//	workflow := flow.RegisterFlow("TestGetAll")
//	process := workflow.Process("TestGetAll")
//	process.AfterStep(true, ErrorResultPrinter)
//	process.NameStep(getAllAndSet("1", "0"), "1")
//	process.NameStep(getAllAndSet("2", "0", "1"), "2", "1")
//	process.NameStep(getAllAndSet("3", "0", "1", "2"), "3", "2")
//	result := flow.DoneFlow("TestGetAll", map[string]any{"all": "0"})
//	if !result.Success() {
//		t.Errorf("flow[%s] failed, explain=%v", result.GetName(), result.Exceptions())
//	}
//	for _, feature := range result.Futures() {
//		t.Logf("process[%s] explain=%v", feature.GetName(), feature.ExplainStatus())
//		if !feature.Success() {
//			t.Errorf("process[%s] failed, exceptions=%v", feature.GetName(), feature.Exceptions())
//		}
//	}
//}

func TestContextNameCorrect(t *testing.T) {
	workflow := flow.RegisterFlow("TestContextNameCorrect")
	workflow.BeforeFlow(false, func(info *flow.WorkFlow) (keepOn bool, err error) {
		if info.GetName() != "TestContextNameCorrect" {
			fmt.Printf("beforeflow workflow context name incorrect, workflow's name = %s\n", info.GetName())
		} else {
			fmt.Printf("workflow's context name='%s' before flow\n", info.GetName())
		}
		return true, nil
	})
	workflow.AfterFlow(false, func(info *flow.WorkFlow) (keepOn bool, err error) {
		if info.GetName() != "TestContextNameCorrect" {
			fmt.Printf("afterflow workflow context name incorrect, workflow's name = %s\n", info.GetName())
		} else {
			fmt.Printf("workflow's context name='%s' after flow\n", info.GetName())
		}
		return true, nil
	})
	workflow.BeforeProcess(false, func(info *flow.Process) (keepOn bool, err error) {
		if info.ContextName() != "TestContextNameCorrect-Process" {
			fmt.Printf("beforeprocess process context name incorrect, workflow's name = %s\n", info.ContextName())
		} else {
			fmt.Printf("process's context name='%s' before process\n", info.ContextName())
		}
		return true, nil
	})
	workflow.AfterProcess(false, func(info *flow.Process) (keepOn bool, err error) {
		if info.ContextName() != "TestContextNameCorrect-Process" {
			fmt.Printf("afterprocess process context name incorrect, workflow's name = %s\n", info.ContextName())
		} else {
			fmt.Printf("process's context name='%s' after process\n", info.ContextName())
		}
		return true, nil
	})
	workflow.BeforeStep(false, func(info *flow.Step) (keepOn bool, err error) {
		if info.ContextName() != "Step1" {
			fmt.Printf("before step context name incorrect, step's name = %s\n", info.ContextName())
		} else {
			fmt.Printf("step's context name='%s' before step\n", info.ContextName())
		}
		return true, nil
	})
	workflow.AfterStep(false, func(info *flow.Step) (keepOn bool, err error) {
		if info.ContextName() != "Step1" {
			fmt.Printf("afterstep step context name incorrect, step's name = %s\n", info.ContextName())
		} else {
			fmt.Printf("step's context name='%s' after step\n", info.ContextName())
		}
		return true, nil
	})
	process := workflow.Process("TestContextNameCorrect-Process")
	process.NameStep(func(ctx flow.StepCtx) (any, error) {
		if ctx.ContextName() != "Step1" {
			fmt.Printf("run step's context name is incorrect, step's name = %s\n", ctx.ContextName())
			return nil, fmt.Errorf("step context name incorrect")
		}
		fmt.Printf("step name = '%s', while step running\n", ctx.ContextName())
		return nil, nil
	}, "Step1")
	result := flow.DoneFlow("TestContextNameCorrect", nil)
	if !result.Success() {
		t.Errorf("flow[%s] failed, explain=%v", result.GetName(), result.Exceptions())
	}
	result = flow.DoneFlow("TestContextNameCorrect", nil)
	if !result.Success() {
		t.Errorf("flow[%s] failed, explain=%v", result.GetName(), result.Exceptions())
	}
}
