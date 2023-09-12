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

func TestPtrReuse(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow(map[string]any{addrKey: &ctx1})
	procedure1 := workflow.AddProcess("test1", nil)
	procedure1.AddStepWithAlias("1", GenerateStepIncAddr(1))
	procedure1.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	procedure1.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	procedure1 = workflow.AddProcess("test2", nil)
	procedure1.AddStepWithAlias("11", GenerateStepIncAddr(11))
	procedure1.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	procedure1.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if ctx1 != 6 {
		t.Errorf("excute 6 step, but ctx1 = %d", ctx1)
	}
}

func TestWaitToDoneInMultiple(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow(map[string]any{addrKey: &ctx1})
	procedure1 := workflow.AddProcess("test1", nil)
	procedure1.AddStepWithAlias("1", GenerateStepIncAddr(1))
	procedure1.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	procedure1.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	procedure2 := workflow.AddProcess("test2", nil)
	procedure2.AddStepWithAlias("11", GenerateStepIncAddr(11))
	procedure2.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	procedure2.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	procedure2.AddStepWithAlias("14", GenerateSleep(1*time.Second), "13")
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if ctx1 != 7 {
		t.Errorf("excute 7 step, but ctx1 = %d", ctx1)
	}
}

func TestWorkFlowCtx(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow(map[string]any{addrKey: &ctx1})
	procedure1 := workflow.AddProcess("test1", nil)
	procedure1.AddStepWithAlias("1", GenerateStepIncAddr(1))
	procedure1.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	procedure1.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	procedure2 := workflow.AddProcess("test2", nil)
	procedure2.AddStepWithAlias("11", GenerateStepIncAddr(11))
	procedure2.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	procedure2.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if ctx1 != 6 {
		t.Errorf("excute 6 step, but ctx1 = %d", ctx1)
	}
}

func TestMultipleNormalProcedure(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow(nil)
	procedure := workflow.AddProcess("test1", nil)
	procedure.SupplyCtxByMap(map[string]any{addrKey: &ctx1})
	procedure.AddStepWithAlias("1", GenerateStepIncAddr(1))
	procedure.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	procedure.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	procedure = workflow.AddProcess("test2", nil)
	procedure.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	procedure.AddStepWithAlias("11", GenerateStepIncAddr(11))
	procedure.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	procedure.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if ctx1 != 3 {
		t.Errorf("excute 3 step, but ctx1 = %d", ctx1)
	}
	if ctx2 != 3 {
		t.Errorf("excute 3 step, but ctx2 = %d", ctx2)
	}
}

func TestProcedureAndWorkflowCtx(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow(map[string]any{addrKey: &ctx1})
	procedure := workflow.AddProcess("test1", nil)
	procedure.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	procedure.AddStepWithAlias("1", GenerateStepIncAddr(1))
	procedure.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	procedure.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	procedure = workflow.AddProcess("test2", nil)
	procedure.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	procedure.AddStepWithAlias("11", GenerateStepIncAddr(11))
	procedure.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	procedure.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
		}
	}
	if ctx1 != 0 {
		t.Errorf("workflow ctx has effective with duplicate key in procedure")
	}
	if ctx2 != 6 {
		t.Errorf("excute 6 step, but ctx2 = %d", ctx2)
	}
}

func TestStepCtx(t *testing.T) {
	defer resetCtx()
	workflow := light_flow.NewWorkflow(map[string]any{addrKey: &ctx1})
	procedure := workflow.AddProcess("test1", nil)
	procedure.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	procedure.AddStepWithAlias("1", ChangeCtxStepFunc(&ctx3))
	procedure.AddStepWithAlias("2", GenerateStepIncAddr(2), "1")
	procedure.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	procedure = workflow.AddProcess("test2", nil)
	procedure.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	procedure.AddStepWithAlias("11", ChangeCtxStepFunc(&ctx4))
	procedure.AddStepWithAlias("12", GenerateStepIncAddr(12), "11")
	procedure.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
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
	workflow := light_flow.NewWorkflow(map[string]any{addrKey: &ctx1})
	procedure := workflow.AddProcess("test1", nil)
	procedure.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	procedure.AddStepWithAlias("1", ChangeCtxStepFunc(&ctx3))
	procedure.AddStepWithAlias("2", ChangeCtxStepFunc(&ctx5), "1")
	procedure.AddStepWithAlias("3", GenerateStepIncAddr(3), "2")
	procedure = workflow.AddProcess("test2", nil)
	procedure.SupplyCtxByMap(map[string]any{addrKey: &ctx2})
	procedure.AddStepWithAlias("11", ChangeCtxStepFunc(&ctx4))
	procedure.AddStepWithAlias("12", ChangeCtxStepFunc(&ctx6), "11")
	procedure.AddStepWithAlias("13", GenerateStepIncAddr(13), "12")
	features := workflow.WaitToDone()
	for name, feature := range features {
		explain := strings.Join(feature.GetStatusExplain(), ", ")
		fmt.Printf("procedure[%s] explain=%s\n", name, explain)
		if !feature.IsSuccess() {
			t.Errorf("procedure[%s] fail", name)
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
