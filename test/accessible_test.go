package test

import (
	"fmt"
	"github.com/Bilibotter/light-flow/flow"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func FlowCallback(flag bool, visible []string, notVisible []string, keys ...string) func(flow.WorkFlow) (keepOn bool, err error) {
	return func(info flow.WorkFlow) (keepOn bool, err error) {
		if flag {
			time.Sleep(10 * time.Millisecond)
		}
		atomic.AddInt64(&current, 1)
		println("Invoke flow callback")
		return true, nil
	}
}

func ProcCallback(flag bool, visible []string, notVisible []string, keys ...string) func(flow.Process) (keepOn bool, err error) {
	return func(info flow.Process) (keepOn bool, err error) {
		if flag {
			time.Sleep(10 * time.Millisecond)
		}
		for _, k := range visible {
			if _, ok := info.Get(k); !ok {
				panic(fmt.Sprintf("%s not found %s", info.Name(), k))
			}
		}
		fmt.Printf("[Process: %s] can get all keys = %s\n", info.Name(), strings.Join(visible, ", "))
		for _, k := range notVisible {
			if _, ok := info.Get(k); ok {
				panic(fmt.Sprintf("%s found %s", info.Name(), k))
			}
		}
		fmt.Printf("[Process: %s] can't get all keys = %s\n", info.Name(), strings.Join(notVisible, ", "))
		for _, k := range keys {
			info.Set(k, k)
		}
		fmt.Printf("[Process: %s] set all keys = %s\n", info.Name(), strings.Join(keys, ", "))
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func StepCallback(flag bool, visible []string, notVisible []string, keys ...string) func(flow.Step) (keepOn bool, err error) {
	return func(info flow.Step) (keepOn bool, err error) {
		if flag {
			time.Sleep(10 * time.Millisecond)
		}
		fmt.Printf("invoke [Step: %s] callback, current=%d\n", info.Name(), atomic.LoadInt64(&current))
		for _, k := range visible {
			if _, ok := info.Get(k); !ok {
				panic(fmt.Sprintf("%s not found %s", info.Name(), k))
			}
		}
		fmt.Printf("[Step: %s] can get all keys = %s\n", info.Name(), strings.Join(visible, ", "))
		for _, k := range notVisible {
			if _, ok := info.Get(k); ok {
				panic(fmt.Sprintf("%s found %s", info.Name(), k))
			}
		}
		fmt.Printf("[Step: %s] can't get all keys = %s\n", info.Name(), strings.Join(notVisible, ", "))
		for _, k := range keys {
			info.Set(k, k)
		}
		fmt.Printf("[Step: %s] set all keys = %s\n", info.Name(), strings.Join(keys, ", "))
		atomic.AddInt64(&current, 1)
		println()
		return true, nil
	}
}

func CtxChecker(flag bool, visible []string, notVisible []string, keys ...string) func(ctx flow.Step) (any, error) {
	return func(ctx flow.Step) (any, error) {
		fmt.Printf("Context[ %s ] start check context, current=%d\n", ctx.Name(), atomic.LoadInt64(&current))
		if flag {
			time.Sleep(10 * time.Millisecond)
		}
		for _, k := range visible {
			if _, ok := ctx.Get(k); !ok {
				panic(fmt.Sprintf("%s not found %s", ctx.Name(), k))
			}
		}
		fmt.Printf("Context[ %s ] can get all keys = %s\n", ctx.Name(), strings.Join(visible, ", "))
		for _, k := range notVisible {
			if _, ok := ctx.Get(k); ok {
				panic(fmt.Sprintf("%s found %s", ctx.Name(), k))
			}
		}
		fmt.Printf("Context[ %s ] can't get all keys = %s\n", ctx.Name(), strings.Join(notVisible, ", "))
		for _, k := range keys {
			ctx.Set(k, k)
		}
		fmt.Printf("Context[ %s ] set all keys = %s\n", ctx.Name(), strings.Join(keys, ", "))
		atomic.AddInt64(&current, 1)
		println()
		return nil, nil
	}
}

func TestDependConnectAndIsolated(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestDependConnectAndIsolated")
	process := workflow.Process("TestDependConnectAndIsolated")
	process.AfterStep(true, ErrorResultPrinter)
	process.NamedStep(CtxChecker(false, []string{"1", "2", "3"}, []string{}, "a", "b", "c"), "1").
		Next(CtxChecker(false, []string{"a", "b", "c"}, []string{}, "d", "e", "f"), "2").
		Same(CtxChecker(true, []string{"a", "b", "c"}, []string{"d", "e", "f"}, "g", "h", "i"), "3")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestDependConnectAndIsolated", map[string]any{"1": "1", "2": "2", "3": "3"})
}

func TestFlowCallbackValid(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestFlowCallbackValid")
	workflow.BeforeFlow(true, FlowCallback(false, []string{"4", "5", "6"}, []string{}, "1", "2", "3"))
	workflow.AfterFlow(true, FlowCallback(false, []string{"1", "2", "3"}, []string{}))
	process := workflow.Process("TestFlowCallbackValid")
	process.NamedStep(CtxChecker(false, []string{}, []string{}, "a", "b", "c"), "1").
		Next(CtxChecker(false, []string{"a", "b", "c"}, []string{}, "d", "e", "f"), "2").
		Same(CtxChecker(true, []string{"a", "b", "c"}, []string{"d", "e", "f"}, "g", "h", "i"), "3")
	workflow.AfterFlow(false, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestFlowCallbackValid", map[string]any{"4": "1", "5": "2", "6": "3"})
}

func TestProcCallbackValid(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcCallbackValid")
	process := workflow.Process("TestProcCallbackValid")
	process.BeforeProcess(true, ProcCallback(false, []string{"4", "5", "6"}, []string{}, "1", "2", "3"))
	process.AfterProcess(true, ProcCallback(false, []string{"1", "2", "3"}, []string{}))
	process.NamedStep(CtxChecker(false, []string{"1", "2", "3"}, []string{}, "a", "b", "c"), "1").
		Next(CtxChecker(false, []string{"a", "b", "c"}, []string{}, "d", "e", "f"), "2").
		Same(CtxChecker(true, []string{"a", "b", "c"}, []string{"d", "e", "f"}, "g", "h", "i"), "3")
	workflow.AfterFlow(false, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestProcCallbackValid", map[string]any{"4": "1", "5": "2", "6": "3"})
}

func TestStepCallbackValid(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestStepCallbackValid")
	process := workflow.Process("TestStepCallbackValid")
	process.AfterStep(true, ErrorResultPrinter)
	process.BeforeStep(true, StepCallback(false, []string{"4", "5", "6"}, []string{}, "1", "2", "3"))
	process.AfterStep(true, StepCallback(false, []string{"1", "2", "3"}, []string{}))
	process.NamedStep(CtxChecker(false, []string{"1", "2", "3"}, []string{}, "a", "b", "c"), "1").
		Next(CtxChecker(false, []string{"a", "b", "c"}, []string{}, "d", "e", "f"), "2").
		Same(CtxChecker(true, []string{"a", "b", "c"}, []string{"d", "e", "f"}, "g", "h", "i"), "3")
	workflow.AfterFlow(false, CheckResult(t, 9, flow.Success))
	flow.DoneFlow("TestStepCallbackValid", map[string]any{"4": "1", "5": "2", "6": "3"})
}

func TestAllCallbackConnect(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestAllCallbackConnect")
	process := workflow.Process("TestAllCallbackConnect")
	process.AfterStep(true, ErrorResultPrinter)
	process.BeforeProcess(true, ProcCallback(false, []string{"1"}, []string{}, "3"))
	process.AfterProcess(true, ProcCallback(false, []string{"1", "3", "4", "6"}, []string{}))
	process.BeforeStep(true, StepCallback(false, []string{"3"}, []string{}, "4"))
	process.AfterStep(true, StepCallback(false, []string{"1", "3", "4", "6"}, []string{}))
	process.NamedStep(CtxChecker(false, []string{"1", "3", "4"}, []string{}, "6"), "1")
	workflow.AfterFlow(false, CheckResult(t, 5, flow.Success))
	flow.DoneFlow("TestAllCallbackConnect", map[string]any{"1": "1"})
}
