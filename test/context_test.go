package test

import (
	"fmt"
	"gitee.com/MetaphysicCoding/light-flow"
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

func GenerateSleep(duration time.Duration) func(ctx *light_flow.Context) (any, error) {
	return func(ctx *light_flow.Context) (any, error) {
		time.Sleep(duration)
		addrWrap, ok := ctx.Get(addrKey)
		if !ok {
			panic("not found addr")
		}
		atomic.AddInt64(addrWrap.(*int64), 1)
		return nil, nil
	}
}

func ChangeCtxStepFunc(addr *int64) func(ctx *light_flow.Context) (any, error) {
	return func(ctx *light_flow.Context) (any, error) {
		time.Sleep(100 * time.Millisecond)
		ctx.Set(addrKey, addr)
		println("change ctx")
		addrWrap, ok := ctx.Get(addrKey)
		if !ok {
			panic("not found addr")
		}
		atomic.AddInt64(addrWrap.(*int64), 1)
		return nil, nil
	}
}

func GenerateStepIncAddr(i int) func(ctx *light_flow.Context) (any, error) {
	return func(ctx *light_flow.Context) (any, error) {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("%d.step finish\n", i)
		addrWrap, ok := ctx.Get(addrKey)
		if !ok {
			panic("not found addr")
		}
		atomic.AddInt64(addrWrap.(*int64), 1)
		return i, nil
	}
}

func ExposeAddrFunc(addr *int64) func(ctx *light_flow.Context) (any, error) {
	return func(ctx *light_flow.Context) (any, error) {
		ctx.Exposed(addrKey, addr)
		atomic.AddInt64(addr, 1)
		return nil, nil
	}
}

func getUnexist(ctx *light_flow.Context) (any, error) {
	println("start")
	ctx.Get("unexist")
	println("end")
	return nil, nil
}

func invalidUse(ctx *light_flow.Context) (any, error) {
	return nil, nil
}

func TestSearch(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", invalidUse)
	process.AddStepWithAlias("2", invalidUse, "1")
	process.AddStepWithAlias("3", invalidUse, "1")
	process.AddStepWithAlias("4", getUnexist, "2", "3")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
}

func TestPriorityWithSelf(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
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
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 2 {
		t.Errorf("excute 2 step, but ctx1 = %d", ctx1)
	}
	if ctx2 != 2 {
		t.Errorf("excute 2 step, but ctx2 = %d", ctx2)
	}
}

func TestPriorityCheck(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
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
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 2 {
		t.Errorf("excute 2 step, but ctx1 = %d", ctx1)
	}
	if ctx2 != 2 {
		t.Errorf("excute 2 step, but ctx2 = %d", ctx2)
	}
}

func TestPriority(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", ChangeCtxStepFunc(&ctx1))
	process.AddStepWithAlias("2", GenerateStepIncAddr(1), "1")
	process.AddStepWithAlias("3", ChangeCtxStepFunc(&ctx2), "1")
	step := process.AddStepWithAlias("4", GenerateStepIncAddr(1), "2", "3")
	step.AddPriority(map[string]any{addrKey: "3"})
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 2 {
		t.Errorf("excute 2 step, but ctx1 = %d", ctx1)
	}
	if ctx2 != 2 {
		t.Errorf("excute 2 step, but ctx2 = %d", ctx2)
	}
}

func TestExpose(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.AddStepWithAlias("1", ExposeAddrFunc(&ctx1))
	process.AddStepWithAlias("2", GenerateStepIncAddr(2))
	process.AddStepWithAlias("3", GenerateStepIncAddr(3))
	process = workflow.AddProcess("test2", nil)
	process.AddStepWithAlias("11", ExposeAddrFunc(&ctx2))
	process.AddStepWithAlias("12", GenerateStepIncAddr(12))
	process.AddStepWithAlias("13", GenerateStepIncAddr(13))
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 3 {
		t.Errorf("excute 3 step, but ctx1 = %d", ctx1)
	}
	if ctx2 != 3 {
		t.Errorf("excute 3 step, but ctx2 = %d", ctx2)
	}
}

func TestPtrReuse(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](map[string]any{addrKey: &ctx1})
	process1 := workflow.AddProcess("test1", nil)
	process1.AddStepWithAlias("1", GenerateStepIncAddr(1))
	process1.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process1.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process1 = workflow.AddProcess("test2", nil)
	process1.AddStepWithAlias("11", GenerateStepIncAddr(11))
	process1.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process1.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 6 {
		t.Errorf("excute 6 step, but ctx1 = %d", ctx1)
	}
}

func TestWaitToDoneInMultiple(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](map[string]any{addrKey: &ctx1})
	process1 := workflow.AddProcess("test1", nil)
	process1.AddStepWithAlias("1", GenerateStepIncAddr(1))
	process1.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process1.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process2 := workflow.AddProcess("test2", nil)
	process2.AddStepWithAlias("11", GenerateStepIncAddr(11))
	process2.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process2.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	process2.AddStepWithAlias("14", GenerateSleep(1*time.Second), "13")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 7 {
		t.Errorf("excute 7 step, but ctx1 = %d", ctx1)
	}
}

func TestWorkFlowCtx(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](map[string]any{addrKey: &ctx1})
	process1 := workflow.AddProcess("test1", nil)
	process1.AddStepWithAlias("1", GenerateStepIncAddr(1))
	process1.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process1.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process2 := workflow.AddProcess("test2", nil)
	process2.AddStepWithAlias("11", GenerateStepIncAddr(11))
	process2.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process2.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 6 {
		t.Errorf("excute 6 step, but ctx1 = %d", ctx1)
	}
}

func TestMultipleNormalprocess(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](nil)
	process := workflow.AddProcess("test1", nil)
	process.SupplyCtxByMap(map[string]any{addrKey: &ctx1})
	process.AddStepWithAlias("1", GenerateStepIncAddr(1))
	process.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process = workflow.AddProcess("test2", nil)
	process.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	process.AddStepWithAlias("11", GenerateStepIncAddr(11))
	process.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 3 {
		t.Errorf("excute 3 step, but ctx1 = %d", ctx1)
	}
	if ctx2 != 3 {
		t.Errorf("excute 3 step, but ctx2 = %d", ctx2)
	}
}

func TestprocessAndWorkflowCtx(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](map[string]any{addrKey: &ctx1})
	process := workflow.AddProcess("test1", nil)
	process.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	process.AddStepWithAlias("1", GenerateStepIncAddr(1))
	process.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process = workflow.AddProcess("test2", nil)
	process.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	process.AddStepWithAlias("11", GenerateStepIncAddr(11))
	process.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 0 {
		t.Errorf("workflow ctx has effective with duplicate key in process")
	}
	if ctx2 != 6 {
		t.Errorf("excute 6 step, but ctx2 = %d", ctx2)
	}
}

func TestStepCtx(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](map[string]any{addrKey: &ctx1})
	process := workflow.AddProcess("test1", nil)
	process.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	process.AddStepWithAlias("1", ChangeCtxStepFunc(&ctx3))
	process.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	process.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process = workflow.AddProcess("test2", nil)
	process.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	process.AddStepWithAlias("11", ChangeCtxStepFunc(&ctx4))
	process.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	process.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 0 {
		t.Errorf("workflow ctx has effective with duplicate key in step")
	}
	if ctx2 != 0 {
		t.Errorf("produce ctx has effective with duplicate key in step")
	}
	if ctx3 != 3 {
		t.Errorf("excute 3 step, but ctx3 = %d", ctx3)
	}
	if ctx4 != 3 {
		t.Errorf("excute 3 step, but ctx4 = %d", ctx4)
	}
}

func TestDependStepCtx(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow[any](map[string]any{addrKey: &ctx1})
	process := workflow.AddProcess("test1", nil)
	process.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	process.AddStepWithAlias("1", ChangeCtxStepFunc(&ctx3))
	process.AddStepWithAlias("2", ChangeCtxStepFunc(&ctx5), "1")
	process.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	process = workflow.AddProcess("test2", nil)
	process.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	process.AddStepWithAlias("11", ChangeCtxStepFunc(&ctx4))
	process.AddStepWithAlias("12", ChangeCtxStepFunc(&ctx6), "11")
	process.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.Done()
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if ctx1 != 0 {
		t.Errorf("workflow ctx has effective with duplicate key in step")
	}
	if ctx2 != 0 {
		t.Errorf("produce ctx has effective with duplicate key in step")
	}
	if ctx3 != 1 {
		t.Errorf("excute 1 step, but ctx3 = %d", ctx3)
	}
	if ctx4 != 1 {
		t.Errorf("excute 1 step, but ctx4 = %d", ctx4)
	}
	if ctx5 != 2 {
		t.Errorf("excute 2 step, but ctx3 = %d", ctx5)
	}
	if ctx6 != 2 {
		t.Errorf("excute 2 step, but ctx4 = %d", ctx6)
	}
}
