package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"strings"
	"sync/atomic"
	"testing"
)

func CheckGetEndValues(check ...string) func(info flow.StepCtx) (result any, err error) {
	return func(info flow.StepCtx) (result any, err error) {
		values := info.GetEndValues("Step")
		if len(values) != len(check) {
			panic(fmt.Sprintf("values length[%d] not equal check length[%d]", len(values), len(check)))
		}
		for _, k := range check {
			if value, exist := values[k]; exist {
				if value.(string) != k {
					fmt.Printf("step[%s] key[%s] is %#v, expected get %s\n\n", info.ContextName(), k, value, k)
					panic(fmt.Sprintf("step[%s] key[%s] is %#v, expected get %s", info.ContextName(), k, value, k))
				}
			} else {
				fmt.Printf("step[%s] has no key[%s]\n\n", info.ContextName(), k)
				panic(fmt.Sprintf("step[%s] has no key[%s]", info.ContextName(), k))
			}
		}
		atomic.AddInt64(&current, 1)
		return "result", nil
	}
}

func CheckGetByStep(check ...string) func(info *flow.Process) (keepOn bool, err error) {
	return func(info *flow.Process) (keepOn bool, err error) {
		for _, k := range check {
			if value, exist := info.GetByStepName(k, k); exist {
				if value.(string) != k {
					fmt.Printf("step[%s] key[%s] is %#v, expected get %s\n\n", info.ContextName(), k, value, k)
					panic(fmt.Sprintf("step[%s] key[%s] is %#v, expected get %s", info.ContextName(), k, value, k))
				}
			} else {
				fmt.Printf("step[%s] has no key[%s]\n\n", info.ContextName(), k)
				panic(fmt.Sprintf("step[%s] has no key[%s]", info.ContextName(), k))
			}
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func StepCallbackCheck(check ...string) func(info *flow.Step) (keepOn bool, err error) {
	return func(info *flow.Step) (keepOn bool, err error) {
		for _, s := range check {
			if value, exist := info.Get(s); exist {
				if value.(string) != s {
					fmt.Printf("step[%s] key[%s] is %#v, expected get %s\n\n", info.ContextName(), s, value, s)
					panic(fmt.Sprintf("step[%s] key[%s] is %#v, expected get %s", info.ContextName(), s, value, s))
				}
			} else {
				fmt.Printf("step[%s] has no key[%s]\n\n", info.ContextName(), s)
				panic(fmt.Sprintf("step[%s] has no key[%s]", info.ContextName(), s))
			}
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}
func StepResultCheck(info *flow.Step) (keepOn bool, err error) {
	result, exist := info.GetResult(info.Name)
	if !exist {
		panic(fmt.Sprintf("step[%s] reuslt is missing.", info.Name))
	}
	if result.(string) != "result" {
		panic(fmt.Sprintf("step[%s] reuslt not equal to \"result\"", info.Name))
	} else {
		fmt.Printf("step[%s] reuslt = %#v\n\n", info.Name, result)
	}
	atomic.AddInt64(&current, 1)
	return true, nil
}

func StepCtxFunc(input map[string]any, check ...string) func(ctx flow.StepCtx) (any, error) {
	return func(ctx flow.StepCtx) (any, error) {
		fmt.Printf("step[%s] runnning \n", ctx.ContextName())
		for _, s := range check {
			if value, exist := ctx.Get(s); exist {
				if value.(string) != s {
					panic(fmt.Sprintf("step[%s] key[%s] is %#v, expected get %s", ctx.ContextName(), s, value, s))
				}
			} else {
				panic(fmt.Sprintf("step[%s] has no key[%s]", ctx.ContextName(), s))
			}
		}
		for k, v := range input {
			ctx.Set(k, v)
		}
		atomic.AddInt64(&current, 1)
		return "result", nil
	}
}

func ProcCheckFunc(check ...string) func(info *flow.Process) (keepOn bool, err error) {
	return func(info *flow.Process) (keepOn bool, err error) {
		for _, s := range check {
			if value, exist := info.Get(s); exist {
				if value.(string) != s {
					fmt.Printf("process[%s] key[%s] is %#v, expected get %s\n\n", info.ContextName(), s, value, s)
					panic(fmt.Sprintf("process[%s] key[%s] is %#v, expected get %s", info.ContextName(), s, value, s))
				}
			} else {
				fmt.Printf("process[%s] has no key[%s]\n\n", info.ContextName(), s)
				panic(fmt.Sprintf("process[%s] has no key[%s]", info.ContextName(), s))
			}
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func ProcCtxFunc(input map[string]any) func(info *flow.Process) (keepOn bool, err error) {
	return func(info *flow.Process) (keepOn bool, err error) {
		for k, v := range input {
			info.Set(k, v)
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func SetCtxStepFunc(input map[string]any) func(ctx flow.StepCtx) (any, error) {
	return func(ctx flow.StepCtx) (any, error) {
		fmt.Printf("ctx[%s] set value\n", ctx.ContextName())
		for k, v := range input {
			ctx.Set(k, v)
		}
		atomic.AddInt64(&current, 1)
		return "result", nil
	}
}

func TestGetEndIgnoreProc(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestGetEndIgnoreProc")
	process := workflow.Process("TestGetEndIgnoreProc")
	process.BeforeProcess(true, ProcCtxFunc(map[string]any{"Step": "TestGetEndIgnoreProc"}))
	process.NameStep(CheckGetEndValues(), "check")
	result := flow.DoneFlow("TestGetEndIgnoreProc", nil)
	if !result.Success() {
		for _, f := range result.Futures() {
			if !f.Success() {
				t.Errorf("process[%s] failed, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			} else {
				t.Errorf("process[%s] success, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			}
		}
	}
	if atomic.LoadInt64(&current) != 2 {
		t.Errorf("execute 2 step, but current = %d", current)
	}
}

func TestCollectGetEndValues(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestCollectGetEndValues")
	process := workflow.Process("TestCollectGetEndValues")
	process.NameStep(SetCtxStepFunc(map[string]any{"Step": "Error"}), "Step0")
	process.NameStep(SetCtxStepFunc(map[string]any{"Step": "Error"}), "Step1", "Step0")
	process.NameStep(SetCtxStepFunc(map[string]any{"Step": "Error"}), "Step2", "Step0")
	process.NameStep(SetCtxStepFunc(map[string]any{"Step": "Error"}), "Step3", "Step0")
	process.Tail(SetCtxStepFunc(map[string]any{"Step": "Step4"}), "Step4")
	process.Tail(CheckGetEndValues("Step4"), "check")
	result := flow.DoneFlow("TestCollectGetEndValues", nil)
	if !result.Success() {
		for _, f := range result.Futures() {
			if !f.Success() {
				t.Errorf("process[%s] failed, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			}
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
}

func TestMultipleGetEndValues(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleGetEndValues")
	process := workflow.Process("TestMultipleGetEndValues")
	process.NameStep(SetCtxStepFunc(map[string]any{"Step": "Error"}), "Step0")
	process.NameStep(SetCtxStepFunc(map[string]any{"Step": "Step1"}), "Step1", "Step0")
	process.NameStep(SetCtxStepFunc(map[string]any{"Step": "Step2"}), "Step2", "Step0")
	process.NameStep(SetCtxStepFunc(map[string]any{"Step": "Step3"}), "Step3", "Step0")
	process.Tail(CheckGetEndValues("Step1", "Step2", "Step3"), "check")
	result := flow.DoneFlow("TestMultipleGetEndValues", nil)
	if !result.Success() {
		for _, f := range result.Futures() {
			if !f.Success() {
				t.Errorf("process[%s] failed, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			}
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
}

func TestGetByStepName(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestGetByStepName")
	process := workflow.Process("TestGetByStepName")
	process.AfterProcess(true, CheckGetByStep("Step0", "Step1", "Step2"))
	step := process.NameStep(SetCtxStepFunc(map[string]any{"Step0": "Step0"}), "Step0")
	step.
		Next(SetCtxStepFunc(map[string]any{"Step1": "Step1"}), "Step1").
		Next(SetCtxStepFunc(map[string]any{"Step2": "Step2"}), "Step2")
	result := flow.DoneFlow("TestGetByStepName", nil)
	if !result.Success() {
		for _, f := range result.Futures() {
			if !f.Success() {
				t.Errorf("process[%s] failed, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			}
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestInputVisibleToStepAndProc(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestInputVisibleToStepAndProc")
	process := workflow.Process("TestInputVisibleToStepAndProc")
	process.BeforeProcess(true, ProcCheckFunc("Step0"))
	process.AfterProcess(true, ProcCheckFunc("Step0"))
	process.BeforeStep(true, StepCallbackCheck("Step0"))
	process.AfterStep(true, StepCallbackCheck("Step0"))
	process.NameStep(StepCtxFunc(map[string]any{}, "Step0"), "Step1")
	result := flow.DoneFlow("TestInputVisibleToStepAndProc", map[string]any{"Step0": "Step0"})
	if !result.Success() {
		for _, f := range result.Futures() {
			if !f.Success() {
				t.Errorf("process[%s] failed, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			}
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
}

func TestAfterProcAndStepCtxConnected(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestAfterProcAndStepCtxConnected")
	process := workflow.Process("TestAfterProcAndStepCtxConnected")
	process.AfterProcess(true, ProcCheckFunc("Step0", "Step1", "Step2", "Step3"))
	process.AfterStep(true, StepResultCheck)
	step := process.NameStep(SetCtxStepFunc(map[string]any{"Step0": "Step0", "Step1": "Error", "Step2": "Error", "Step3": "Error"}), "Step1")
	step.
		Next(SetCtxStepFunc(map[string]any{"Step1": "Step1", "Step2": "Step2"}), "Step2").
		Same(SetCtxStepFunc(map[string]any{"Step3": "Step3"}), "Step3")
	result := flow.DoneFlow("TestAfterProcAndStepCtxConnected", nil)
	if !result.Success() {
		for _, f := range result.Futures() {
			if !f.Success() {
				t.Errorf("process[%s] failed, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			}
		}
		t.Errorf("flow[%s] failed", result.GetName())
	}
	if atomic.LoadInt64(&current) != 7 {
		t.Errorf("execute 7 step, but current = %d", current)
	}
}

func TestProcAndStepCtxConnected(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcAndStepCtxConnected")
	process := workflow.Process("TestProcAndStepCtxConnected")
	process.BeforeProcess(false, ProcCtxFunc(map[string]any{
		"Step0": "Step0", "Step1": "Step1", "Step2": "Step2", "Step3": "Step3",
	}))
	process.AfterStep(true, StepResultCheck)
	step := process.NameStep(StepCtxFunc(map[string]any{"Step1": "Step1"}, "Step0", "Step1"), "Step1")
	step.
		Next(StepCtxFunc(map[string]any{"Step2": "Step2"}, "Step0", "Step1", "Step2"), "Step2").
		Next(StepCtxFunc(map[string]any{"Step3": "Step3"}, "Step0", "Step1", "Step2", "Step3"), "Step3")
	result := flow.DoneFlow("TestProcAndStepCtxConnected", nil)
	if !result.Success() {
		for _, f := range result.Futures() {
			if !f.Success() {
				t.Errorf("process[%s] failed, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			}
		}
		t.Errorf("flow[%s] failed", result.GetName())
	}
	if atomic.LoadInt64(&current) != 7 {
		t.Errorf("execute 7 step, but current = %d", current)
	}
}

func TestProcContextAndResultIsolated(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcContextAndResultIsolated")
	process := workflow.Process("TestProcContextAndResultIsolated")
	process.BeforeProcess(false, ProcCtxFunc(map[string]any{"Step1": "Step1"}))
	process.AfterStep(true, StepResultCheck)
	step := process.NameStep(SetCtxStepFunc(map[string]any{"Step2": "Step2"}), "Step1")
	step.
		Next(SetCtxStepFunc(map[string]any{"Step3": "Step3"}), "Step2").
		Next(SetCtxStepFunc(map[string]any{"Step4": "Step4"}), "Step3")
	result := flow.DoneFlow("TestProcContextAndResultIsolated", nil)
	if !result.Success() {
		for _, f := range result.Futures() {
			if !f.Success() {
				t.Errorf("process[%s] failed, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			}
		}
		t.Errorf("flow[%s] failed", result.GetName())
	}
	if atomic.LoadInt64(&current) != 7 {
		t.Errorf("execute 7 step, but current = %d", current)
	}
}

func TestResultAndStepCtxIsolated(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestResultAndStepCtxIsolated")
	process := workflow.Process("TestResultAndStepCtxIsolated")
	process.AfterStep(true, StepResultCheck)
	step := process.NameStep(SetCtxStepFunc(map[string]any{"Step2": "Step2"}), "Step1")
	step.
		Next(SetCtxStepFunc(map[string]any{"Step3": "Step3"}), "Step2").
		Next(SetCtxStepFunc(map[string]any{"Step4": "Step4"}), "Step3")
	result := flow.DoneFlow("TestResultAndStepCtxIsolated", map[string]any{"Step1": "Step1"})
	if !result.Success() {
		for _, f := range result.Futures() {
			if !f.Success() {
				t.Errorf("process[%s] failed, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			}
		}
		t.Errorf("flow[%s] failed", result.GetName())
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
}

func TestResultAndStepCtxIsolatedAfterMerge(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestResultAndStepCtxIsolatedAfterMerge1")
	process := workflow.Process("TestResultAndStepCtxIsolatedAfterMerge1")
	step := process.NameStep(SetCtxStepFunc(map[string]any{"Step2": "Step2"}), "Step1")
	workflow = flow.RegisterFlow("TestResultAndStepCtxIsolatedAfterMerge2")
	process = workflow.Process("TestResultAndStepCtxIsolatedAfterMerge2")
	process.AfterStep(true, StepResultCheck)
	process.Merge("TestResultAndStepCtxIsolatedAfterMerge1")
	step = process.NameStep(SetCtxStepFunc(map[string]any{"Step3": "Step3"}), "Step2", "Step1")
	step.
		Next(SetCtxStepFunc(map[string]any{"Step4": "Step4"}), "Step3")
	result := flow.DoneFlow("TestResultAndStepCtxIsolatedAfterMerge2", map[string]any{"Step1": "Step1"})
	if !result.Success() {
		for _, f := range result.Futures() {
			if !f.Success() {
				t.Errorf("process[%s] failed, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			}
		}
		t.Errorf("flow[%s] failed", result.GetName())
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
}

func TestConnectionWhileMergeBreakOrder(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestConnectionWhileMergeBreakOrder1")
	process := workflow.Process("TestConnectionWhileMergeBreakOrder1")
	step := process.NameStep(StepCtxFunc(map[string]any{"Step1": "Step1", "Step3": "Error", "Step4": "Error"}), "Step1")
	step.Next(StepCtxFunc(map[string]any{"Step2": "Step2"}, "Step1", "Step3", "Step4"), "Step2")
	process.NameStep(StepCtxFunc(map[string]any{"Step3": "Step3"}), "Step3").
		Next(StepCtxFunc(map[string]any{"Step4": "Step4"}), "Step4")
	workflow = flow.RegisterFlow("TestConnectionWhileMergeBreakOrder2")
	process = workflow.Process("TestConnectionWhileMergeBreakOrder2")
	process.Merge("TestConnectionWhileMergeBreakOrder1")
	process.NameStep(StepCtxFunc(map[string]any{"Step3": "Step3"}), "Step3", "Step1")
	process.NameStep(StepCtxFunc(map[string]any{"Step2": "Step2"}), "Step2", "Step4")
	result := flow.DoneFlow("TestConnectionWhileMergeBreakOrder2", nil)
	if !result.Success() {
		for _, f := range result.Futures() {
			if !f.Success() {
				t.Errorf("process[%s] failed, explain=%s\n", f.Name, strings.Join(f.ExplainStatus(), ","))
			}
		}
		t.Errorf("flow[%s] failed", result.GetName())
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}
