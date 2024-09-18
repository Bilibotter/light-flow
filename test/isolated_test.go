package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"sync/atomic"
	"testing"
)

func CheckGetEndValues(check ...string) func(info flow.Step) (result any, err error) {
	return func(info flow.Step) (result any, err error) {
		values := info.EndValues("Step")
		if len(values) != len(check) {
			panic(fmt.Sprintf("values length[%d] not equal check length[%d]", len(values), len(check)))
		}
		for _, k := range check {
			if value, exist := values[k]; exist {
				if value.(string) != k {
					fmt.Printf("[Step: %s ] key[%s] is %#v, expected get %s\n\n", info.Name(), k, value, k)
					panic(fmt.Sprintf("[Step: %s ] key[%s] is %#v, expected get %s", info.Name(), k, value, k))
				}
			} else {
				fmt.Printf("[Step: %s ] has no key[%s]\n\n", info.Name(), k)
				panic(fmt.Sprintf("[Step: %s ] has no key[%s]", info.Name(), k))
			}
		}
		atomic.AddInt64(&current, 1)
		return "result", nil
	}
}

func CheckGetByStep(check ...string) func(info flow.Process) (keepOn bool, err error) {
	return func(info flow.Process) (keepOn bool, err error) {
		for _, k := range check {
			if value, exist := info.GetByStepName(k, k); exist {
				if value.(string) != k {
					fmt.Printf("[Step: %s ] key[%s] is %#v, expected get %s\n\n", info.Name(), k, value, k)
					panic(fmt.Sprintf("[Step: %s ] key[%s] is %#v, expected get %s", info.Name(), k, value, k))
				}
			} else {
				fmt.Printf("[Step: %s ] has no key[%s]\n\n", info.Name(), k)
				panic(fmt.Sprintf("[Step: %s ] has no key[%s]", info.Name(), k))
			}
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func StepCallbackCheck(check ...string) func(info flow.Step) (keepOn bool, err error) {
	return func(info flow.Step) (keepOn bool, err error) {
		for _, s := range check {
			if value, exist := info.Get(s); exist {
				if value.(string) != s {
					fmt.Printf("[Step: %s ] key[%s] is %#v, expected get %s\n\n", info.Name(), s, value, s)
					panic(fmt.Sprintf("[Step: %s ] key[%s] is %#v, expected get %s", info.Name(), s, value, s))
				}
			} else {
				fmt.Printf("[Step: %s ] has no key[%s]\n\n", info.Name(), s)
				panic(fmt.Sprintf("[Step: %s ] has no key[%s]", info.Name(), s))
			}
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}
func StepResultCheck(info flow.Step) (keepOn bool, err error) {
	result, exist := info.Result(info.Name())
	if !exist {
		panic(fmt.Sprintf("[Step: %s ] reuslt is missing.", info.Name()))
	}
	if result.(string) != "result" {
		panic(fmt.Sprintf("[Step: %s ] reuslt not equal to \"result\"", info.Name()))
	} else {
		fmt.Printf("[Step: %s ] reuslt = %#v\n\n", info.Name(), result)
	}
	atomic.AddInt64(&current, 1)
	return true, nil
}

func StepCtxFunc(input map[string]any, check ...string) func(ctx flow.Step) (any, error) {
	return func(ctx flow.Step) (any, error) {
		fmt.Printf("[Step: %s ] runnning \n", ctx.Name())
		for _, s := range check {
			if value, exist := ctx.Get(s); exist {
				if value.(string) != s {
					panic(fmt.Sprintf("[Step: %s ] key[%s] is %#v, expected get %s", ctx.Name(), s, value, s))
				}
			} else {
				panic(fmt.Sprintf("[Step: %s ] has no key[%s]", ctx.Name(), s))
			}
		}
		for k, v := range input {
			ctx.Set(k, v)
		}
		atomic.AddInt64(&current, 1)
		return "result", nil
	}
}

func ProcCheckFunc(check ...string) func(info flow.Process) (keepOn bool, err error) {
	return func(info flow.Process) (keepOn bool, err error) {
		for _, s := range check {
			if value, exist := info.Get(s); exist {
				if value.(string) != s {
					fmt.Printf("[Process: %s] key[%s] is %#v, expected get %s\n\n", info.Name(), s, value, s)
					panic(fmt.Sprintf("[Process: %s] key[%s] is %#v, expected get %s", info.Name(), s, value, s))
				}
			} else {
				fmt.Printf("[Process: %s] has no key[%s]\n\n", info.Name(), s)
				panic(fmt.Sprintf("[Process: %s] has no key[%s]", info.Name(), s))
			}
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func ProcCtxFunc(input map[string]any) func(info flow.Process) (keepOn bool, err error) {
	return func(info flow.Process) (keepOn bool, err error) {
		for k, v := range input {
			info.Set(k, v)
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func SetCtxStepFunc(input map[string]any) func(ctx flow.Step) (any, error) {
	return func(ctx flow.Step) (any, error) {
		fmt.Printf("Context[ %s ] set value\n", ctx.Name())
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
	workflow.AfterFlow(false, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestGetEndIgnoreProc", nil)
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
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestCollectGetEndValues", nil)
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
	workflow.AfterFlow(false, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestMultipleGetEndValues", nil)
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
	workflow.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestGetByStepName", nil)
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
	workflow.AfterFlow(false, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestInputVisibleToStepAndProc", map[string]any{"Step0": "Step0"})
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
	workflow.AfterFlow(false, CheckResult(t, 7, flow.Success))
	flow.DoneFlow("TestAfterProcAndStepCtxConnected", nil)
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
	workflow.AfterFlow(false, CheckResult(t, 7, flow.Success))
	flow.DoneFlow("TestProcAndStepCtxConnected", nil)
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
	workflow.AfterFlow(false, CheckResult(t, 7, flow.Success))
	flow.DoneFlow("TestProcContextAndResultIsolated", nil)
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
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestResultAndStepCtxIsolated", map[string]any{"Step1": "Step1"})
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
	workflow.AfterFlow(false, CheckResult(t, 6, flow.Success))
	flow.DoneFlow("TestResultAndStepCtxIsolatedAfterMerge2", map[string]any{"Step1": "Step1"})
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
	workflow.AfterFlow(false, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestConnectionWhileMergeBreakOrder2", nil)
}
