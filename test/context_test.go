package test

import (
	"fmt"
	flow "gitee.com/MetaphysicCoding/light-flow"
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

func GenerateSleep(duration time.Duration, args ...any) func(ctx *flow.Context) (any, error) {
	return func(ctx *flow.Context) (any, error) {
		addrWrap, ok := ctx.Get(addrKey)
		if !ok {
			panic(fmt.Sprintf("%s not found addr", ctx.GetCtxName()))
		}
		atomic.AddInt64(addrWrap.(*int64), 1)
		return nil, nil
	}
}

func ChangeCtxStepFunc(addr *int64, args ...any) func(ctx *flow.Context) (any, error) {
	return func(ctx *flow.Context) (any, error) {
		if len(args) > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		ctx.Set(addrKey, addr)
		println("change ctx")
		addrWrap, ok := ctx.Get(addrKey)
		if !ok {
			panic(fmt.Sprintf("%s not found addr", ctx.GetCtxName()))
		}
		atomic.AddInt64(addrWrap.(*int64), 1)
		return nil, nil
	}
}

func GenerateStepIncAddr(i int, args ...any) func(ctx *flow.Context) (any, error) {
	return func(ctx *flow.Context) (any, error) {
		if len(args) > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Printf("%d.step finish\n", i)
		addrWrap, ok := ctx.Get(addrKey)
		if !ok {
			panic(fmt.Sprintf("%s not found addr", ctx.GetCtxName()))
		}
		atomic.AddInt64(addrWrap.(*int64), 1)
		return i, nil
	}
}

func ExposeAddrFunc(addr *int64) func(ctx *flow.Context) (any, error) {
	return func(ctx *flow.Context) (any, error) {
		ctx.Exposed(addrKey, addr)
		atomic.AddInt64(addr, 1)
		return nil, nil
	}
}

func getUnExist(ctx *flow.Context) (any, error) {
	println("start")
	ctx.Get("unExist")
	println("end")
	return nil, nil
}

func invalidUse(ctx *flow.Context) (any, error) {
	return nil, nil
}

func getAllAndSet(value string, history ...string) func(ctx *flow.Context) (any, error) {
	return func(ctx *flow.Context) (any, error) {
		ctx.Set("all", value)
		ks := flow.NewRoutineUnsafeSet[string]()
		vs := flow.NewRoutineUnsafeSet[string]()
		m := ctx.GetAll("all")
		for k, v := range m {
			ks.Add(k)
			vs.Add(v.(string))
		}
		for _, v := range history {
			if !vs.Contains(v) {
				panic(fmt.Sprintf("ctx[%s] not contain %s", ctx.GetCtxName(), v))
			}
		}
		fmt.Printf("ctx[%s] get all keys=%s\n", ctx.GetCtxName(), strings.Join(ks.Slice(), ", "))
		return nil, nil
	}
}

func TestSearch(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestSearch")
	process := workflow.AddProcessWithConf("TestSearch1", nil)
	process.AddStepWithAlias("1", invalidUse)
	process.AddStepWithAlias("2", invalidUse, "1")
	process.AddStepWithAlias("3", invalidUse, "1")
	process.AddStepWithAlias("4", getUnExist, "2", "3")
	features := flow.DoneFlow("TestSearch", nil)
	for name, feature := range features.Features() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
}

func TestPriorityWithSelf(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestPriorityWithSelf")
	process := workflow.AddProcessWithConf("test1", nil)
	process.AddStepWithAlias("0", ChangeCtxStepFunc(&ctx1))
	process.AddStepWithAlias("1", ChangeCtxStepFunc(&ctx1))
	process.AddStepWithAlias("2", GenerateStepIncAddr(1), "1")
	process.AddStepWithAlias("3", ChangeCtxStepFunc(&ctx2), "1")
	step := process.AddStepWithAlias("4", GenerateStepIncAddr(1), "2", "3")
	defer func() {
		if r := recover(); r != nil {
			t.Logf("%v", r)
		}
	}()
	step.AddPriority(map[string]any{addrKey: "4"})
}

func TestPriorityCheck(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestPriorityCheck")
	process := workflow.AddProcessWithConf("TestPriorityCheck", nil)
	process.AddStepWithAlias("0", ChangeCtxStepFunc(&ctx1))
	process.AddStepWithAlias("1", ChangeCtxStepFunc(&ctx1))
	process.AddStepWithAlias("2", GenerateStepIncAddr(1), "1")
	process.AddStepWithAlias("3", ChangeCtxStepFunc(&ctx2), "1")
	step := process.AddStepWithAlias("4", GenerateStepIncAddr(1), "2", "3")
	defer func() {
		if r := recover(); r != nil {
			t.Logf("%v", r)
		}
	}()
	step.AddPriority(map[string]any{addrKey: "0"})
}

func TestPriority(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestPriority")
	process := workflow.AddProcessWithConf("TestPriority", nil)
	process.AddStepWithAlias("1", ChangeCtxStepFunc(&ctx1))
	process.AddStepWithAlias("2", GenerateStepIncAddr(1), "1")
	process.AddStepWithAlias("3", ChangeCtxStepFunc(&ctx2), "1")
	step := process.AddStepWithAlias("4", GenerateStepIncAddr(1), "2", "3")
	step.AddPriority(map[string]any{addrKey: "3"})
	features := flow.DoneFlow("TestPriority", nil)
	for name, feature := range features.Features() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 2 {
		t.Errorf("execute 2 step, but ctx1 = %d", ctx1)
	}
	if atomic.LoadInt64(&ctx2) != 2 {
		t.Errorf("execute 2 step, but ctx2 = %d", ctx2)
	}
}

func TestExpose(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestExpose")
	process := workflow.AddProcessWithConf("TestExpose1", nil)
	process.AddAfterStep(true, ErrorResultPrinter)
	process.AddStepWithAlias("1", ExposeAddrFunc(&ctx1))
	process.AddStepWithAlias("2", GenerateStepIncAddr(2, "ms"))
	process.AddStepWithAlias("3", GenerateStepIncAddr(3, "ms"))
	process = workflow.AddProcessWithConf("TestExpose2", nil)
	process.AddStepWithAlias("11", ExposeAddrFunc(&ctx2))
	process.AddStepWithAlias("12", GenerateStepIncAddr(12, "ms"))
	process.AddStepWithAlias("13", GenerateStepIncAddr(13, "ms"))
	features := flow.DoneFlow("TestExpose", nil)
	for name, feature := range features.Features() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 3 {
		t.Errorf("execute 3 step, but ctx1 = %d", ctx1)
	}
	if atomic.LoadInt64(&ctx2) != 3 {
		t.Errorf("execute 3 step, but ctx2 = %d", ctx2)
	}
}

func TestPtrReuse(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestPtrReuse")
	process1 := workflow.AddProcessWithConf("TestPtrReuse1", nil)
	process1.AddStepWithAlias("1", GenerateStepIncAddr(1))
	process1.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process1.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process1 = workflow.AddProcessWithConf("TestPtrReuse2", nil)
	process1.AddStepWithAlias("11", GenerateStepIncAddr(11))
	process1.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process1.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := flow.DoneFlow("TestPtrReuse", map[string]any{addrKey: &ctx1})
	for name, feature := range features.Features() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 6 {
		t.Errorf("execute 6 step, but ctx1 = %d", ctx1)
	}
}

func TestWaitToDoneInMultiple(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestWaitToDoneInMultiple")
	process1 := workflow.AddProcessWithConf("TestWaitToDoneInMultiple1", nil)
	process1.AddStepWithAlias("1", GenerateStepIncAddr(1))
	process1.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process1.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process2 := workflow.AddProcessWithConf("TestWaitToDoneInMultiple2", nil)
	process2.AddStepWithAlias("11", GenerateStepIncAddr(11))
	process2.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process2.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	process2.AddStepWithAlias("14", GenerateSleep(100*time.Millisecond), "13")
	features := flow.DoneFlow("TestWaitToDoneInMultiple", map[string]any{addrKey: &ctx1})
	for name, feature := range features.Features() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 7 {
		t.Errorf("execute 7 step, but ctx1 = %d", ctx1)
	}
}

func TestWorkFlowCtx(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestWorkFlowCtx")
	process1 := workflow.AddProcessWithConf("TestWorkFlowCtx1", nil)
	process1.AddStepWithAlias("1", GenerateStepIncAddr(1))
	process1.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process1.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process2 := workflow.AddProcessWithConf("TestWorkFlowCtx2", nil)
	process2.AddStepWithAlias("11", GenerateStepIncAddr(11))
	process2.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process2.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := flow.DoneFlow("TestWorkFlowCtx", map[string]any{addrKey: &ctx1})
	for name, feature := range features.Features() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 6 {
		t.Errorf("execute 6 step, but ctx1 = %d", ctx1)
	}
}

func TestProcessAndWorkflowCtx(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestProcessAndWorkflowCtx")
	process := workflow.AddProcessWithConf("TestProcessAndWorkflowCtx1", nil)
	process.AddAfterStep(true, ErrorResultPrinter)
	process.AddStepWithAlias("1", GenerateStepIncAddr(1))
	process.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process = workflow.AddProcessWithConf("TestProcessAndWorkflowCtx2", nil)
	process.AddAfterStep(true, ErrorResultPrinter)
	process.AddStepWithAlias("11", GenerateStepIncAddr(11))
	process.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := flow.DoneFlow("TestProcessAndWorkflowCtx", map[string]any{addrKey: &ctx1})
	for name, feature := range features.Features() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 6 {
		t.Errorf("execute 6 step, but ctx2 = %d", ctx2)
	}
}

func TestStepCtx(t *testing.T) {
	defer resetCtx()
	workflow := flow.RegisterFlow("TestStepCtx")
	process := workflow.AddProcessWithConf("TestStepCtx1", nil)
	process.AddStepWithAlias("1", ChangeCtxStepFunc(&ctx3))
	process.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process = workflow.AddProcessWithConf("TestStepCtx2", nil)
	process.AddStepWithAlias("11", ChangeCtxStepFunc(&ctx4))
	process.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := flow.DoneFlow("TestStepCtx", map[string]any{addrKey: &ctx1})
	for name, feature := range features.Features() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
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
	process := workflow.AddProcessWithConf("TestDependStepCtx1", nil)
	process.AddAfterStep(true, ErrorResultPrinter)
	process.AddStepWithAlias("1", ChangeCtxStepFunc(&ctx3))
	process.AddStepWithAlias("2", ChangeCtxStepFunc(&ctx5), "1")
	process.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process = workflow.AddProcessWithConf("TestDependStepCtx2", nil)
	process.AddStepWithAlias("11", ChangeCtxStepFunc(&ctx4))
	process.AddStepWithAlias("12", ChangeCtxStepFunc(&ctx6), "11")
	process.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := flow.DoneFlow("TestDependStepCtx", map[string]any{addrKey: &ctx1})
	for name, feature := range features.Features() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
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
	process := workflow.AddProcessWithConf("TestFlowMultipleExecute1", nil)
	process.AddAfterStep(true, ErrorResultPrinter)
	process.AddStepWithAlias("1", GenerateStepIncAddr(1))
	process.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process = workflow.AddProcessWithConf("TestFlowMultipleExecute2", nil)
	process.AddStepWithAlias("11", GenerateStepIncAddr(11))
	process.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	flow1 := flow.AsyncFlow("TestFlowMultipleExecute", map[string]any{addrKey: &ctx1})
	flow2 := flow.AsyncFlow("TestFlowMultipleExecute", map[string]any{addrKey: &ctx2})
	flow3 := flow.AsyncFlow("TestFlowMultipleExecute", map[string]any{addrKey: &ctx3})
	flow4 := flow.AsyncFlow("TestFlowMultipleExecute", map[string]any{addrKey: &ctx4})
	flow5 := flow.AsyncFlow("TestFlowMultipleExecute", map[string]any{addrKey: &ctx5})
	for _, flowing := range []flow.FlowController{flow1, flow2, flow3, flow4, flow5} {
		for name, feature := range flowing.Done() {
			explain := strings.Join(feature.ExplainStatus(), ", ")
			fmt.Printf("process[%s] explain=%s\n", name, explain)
			if !feature.Success() {
				t.Errorf("process[%s] fail", name)
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

func TestGetAll(t *testing.T) {
	workflow := flow.RegisterFlow("TestGetAll")
	process := workflow.AddProcess("TestGetAll")
	process.AddAfterStep(true, ErrorResultPrinter)
	process.AddStepWithAlias("1", getAllAndSet("1", "0"))
	process.AddStepWithAlias("2", getAllAndSet("2", "0", "1"), "1")
	process.AddStepWithAlias("3", getAllAndSet("3", "0", "1", "2"), "2")
	result := flow.DoneFlow("TestGetAll", map[string]any{"all": "0"})
	if !result.Success() {
		t.Errorf("flow[%s] failed, explain=%v", result.GetName(), result.Exceptions())
	}
	for name, feature := range result.Features() {
		t.Logf("process[%s] explain=%v", name, feature.ExplainStatus())
		if !feature.Success() {
			t.Errorf("process[%s] failed, exceptions=%v", name, feature.Exceptions())
		}
	}
}
