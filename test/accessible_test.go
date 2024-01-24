package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func FlowCallback(flag bool, visible []string, notVisible []string, keys ...string) func(*flow.FlowInfo) (keepOn bool, err error) {
	return func(info *flow.FlowInfo) (keepOn bool, err error) {
		if flag {
			time.Sleep(10 * time.Millisecond)
		}
		for _, k := range visible {
			if _, ok := info.Get(k); !ok {
				panic(fmt.Sprintf("%s not found %s", info.Name, k))
			}
		}
		fmt.Printf("flow[%s] can get all keys = %s\n", info.ContextName(), strings.Join(visible, ", "))
		for _, k := range notVisible {
			if _, ok := info.Get(k); ok {
				panic(fmt.Sprintf("%s found %s", info.ContextName(), k))
			}
		}
		fmt.Printf("flow[%s] can't get all keys = %s\n", info.ContextName(), strings.Join(notVisible, ", "))
		for _, k := range keys {
			info.Set(k, k)
		}
		fmt.Printf("flow[%s] set all keys = %s\n", info.ContextName(), strings.Join(keys, ", "))
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func ProcCallback(flag bool, visible []string, notVisible []string, keys ...string) func(*flow.ProcessInfo) (keepOn bool, err error) {
	return func(info *flow.ProcessInfo) (keepOn bool, err error) {
		if flag {
			time.Sleep(10 * time.Millisecond)
		}
		for _, k := range visible {
			if _, ok := info.Get(k); !ok {
				panic(fmt.Sprintf("%s not found %s", info.ContextName(), k))
			}
		}
		fmt.Printf("process[%s] can get all keys = %s\n", info.ContextName(), strings.Join(visible, ", "))
		for _, k := range notVisible {
			if _, ok := info.Get(k); ok {
				panic(fmt.Sprintf("%s found %s", info.ContextName(), k))
			}
		}
		fmt.Printf("process[%s] can't get all keys = %s\n", info.ContextName(), strings.Join(notVisible, ", "))
		for _, k := range keys {
			info.Set(k, k)
		}
		fmt.Printf("process[%s] set all keys = %s\n", info.ContextName(), strings.Join(keys, ", "))
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func StepCallback(flag bool, visible []string, notVisible []string, keys ...string) func(*flow.StepInfo) (keepOn bool, err error) {
	return func(info *flow.StepInfo) (keepOn bool, err error) {
		if flag {
			time.Sleep(10 * time.Millisecond)
		}
		for _, k := range visible {
			if _, ok := info.Get(k); !ok {
				panic(fmt.Sprintf("%s not found %s", info.ContextName(), k))
			}
		}
		fmt.Printf("step[%s] can get all keys = %s\n", info.ContextName(), strings.Join(visible, ", "))
		for _, k := range notVisible {
			if _, ok := info.Get(k); ok {
				panic(fmt.Sprintf("%s found %s", info.ContextName(), k))
			}
		}
		fmt.Printf("step[%s] can't get all keys = %s\n", info.ContextName(), strings.Join(notVisible, ", "))
		for _, k := range keys {
			info.Set(k, k)
		}
		fmt.Printf("step[%s] set all keys = %s\n", info.ContextName(), strings.Join(keys, ", "))
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func CtxChecker(flag bool, visible []string, notVisible []string, keys ...string) func(ctx flow.Context) (any, error) {
	return func(ctx flow.Context) (any, error) {
		if flag {
			time.Sleep(10 * time.Millisecond)
		}
		for _, k := range visible {
			if _, ok := ctx.Get(k); !ok {
				panic(fmt.Sprintf("%s not found %s", ctx.ContextName(), k))
			}
		}
		fmt.Printf("ctx[%s] can get all keys = %s\n", ctx.ContextName(), strings.Join(visible, ", "))
		for _, k := range notVisible {
			if _, ok := ctx.Get(k); ok {
				panic(fmt.Sprintf("%s found %s", ctx.ContextName(), k))
			}
		}
		fmt.Printf("ctx[%s] can't get all keys = %s\n", ctx.ContextName(), strings.Join(notVisible, ", "))
		for _, k := range keys {
			ctx.Set(k, k)
		}
		fmt.Printf("ctx[%s] set all keys = %s\n", ctx.ContextName(), strings.Join(keys, ", "))
		atomic.AddInt64(&current, 1)
		return nil, nil
	}
}

func TestDependConnectAndIsolated(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestDependConnectAndIsolated")
	process := workflow.Process("TestDependConnectAndIsolated")
	process.AfterStep(true, ErrorResultPrinter)
	process.AliasStep(CtxChecker(false, []string{"1", "2", "3"}, []string{}, "a", "b", "c"), "1").
		Next(CtxChecker(false, []string{"a", "b", "c"}, []string{}, "d", "e", "f"), "2").
		Same(CtxChecker(true, []string{"a", "b", "c"}, []string{"d", "e", "f"}, "g", "h", "i"), "3")
	result := flow.DoneFlow("TestDependConnectAndIsolated", map[string]any{"1": "1", "2": "2", "3": "3"})
	if !result.Success() {
		for _, exception := range result.Exceptions() {
			t.Errorf("%s failed: %s\n", result.GetName(), exception)
		}
	}
	if atomic.LoadInt64(&current) != 3 {
		t.Errorf("execute 3 step, but current = %d", current)
	}
}

func TestFlowCallbackValid(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestFlowCallbackValid")
	workflow.BeforeFlow(true, FlowCallback(false, []string{"4", "5", "6"}, []string{}, "1", "2", "3"))
	workflow.AfterFlow(true, FlowCallback(false, []string{"1", "2", "3"}, []string{}))
	process := workflow.Process("TestFlowCallbackValid")
	process.AliasStep(CtxChecker(false, []string{"1", "2", "3"}, []string{}, "a", "b", "c"), "1").
		Next(CtxChecker(false, []string{"a", "b", "c"}, []string{}, "d", "e", "f"), "2").
		Same(CtxChecker(true, []string{"a", "b", "c"}, []string{"d", "e", "f"}, "g", "h", "i"), "3")
	result := flow.DoneFlow("TestFlowCallbackValid", map[string]any{"4": "1", "5": "2", "6": "3"})
	if !result.Success() {
		for _, exception := range result.Exceptions() {
			t.Errorf("%s failed: %s\n", result.GetName(), exception)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
}

func TestProcCallbackValid(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestProcCallbackValid")
	process := workflow.Process("TestProcCallbackValid")
	process.BeforeProcess(true, ProcCallback(false, []string{"4", "5", "6"}, []string{}, "1", "2", "3"))
	process.AfterProcess(true, ProcCallback(false, []string{"1", "2", "3"}, []string{}))
	process.AliasStep(CtxChecker(false, []string{"1", "2", "3"}, []string{}, "a", "b", "c"), "1").
		Next(CtxChecker(false, []string{"a", "b", "c"}, []string{}, "d", "e", "f"), "2").
		Same(CtxChecker(true, []string{"a", "b", "c"}, []string{"d", "e", "f"}, "g", "h", "i"), "3")
	result := flow.DoneFlow("TestProcCallbackValid", map[string]any{"4": "1", "5": "2", "6": "3"})
	if !result.Success() {
		for _, exception := range result.Exceptions() {
			t.Errorf("%s failed: %s\n", result.GetName(), exception)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
}

func TestStepCallbackValid(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestStepCallbackValid")
	process := workflow.Process("TestStepCallbackValid")
	process.AfterStep(true, ErrorResultPrinter)
	process.BeforeStep(true, StepCallback(false, []string{"4", "5", "6"}, []string{}, "1", "2", "3"))
	process.AfterStep(true, StepCallback(false, []string{"1", "2", "3"}, []string{}))
	process.AliasStep(CtxChecker(false, []string{"1", "2", "3"}, []string{}, "a", "b", "c"), "1").
		Next(CtxChecker(false, []string{"a", "b", "c"}, []string{}, "d", "e", "f"), "2").
		Same(CtxChecker(true, []string{"a", "b", "c"}, []string{"d", "e", "f"}, "g", "h", "i"), "3")
	result := flow.DoneFlow("TestStepCallbackValid", map[string]any{"4": "1", "5": "2", "6": "3"})
	if !result.Success() {
		for _, exception := range result.Exceptions() {
			t.Errorf("%s failed: %s\n", result.GetName(), exception)
		}
	}
	if atomic.LoadInt64(&current) != 9 {
		t.Errorf("execute 9 step, but current = %d", current)
	}
}

func TestAllCallbackConnect(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestAllCallbackConnect")
	workflow.BeforeFlow(true, FlowCallback(false, []string{"1"}, []string{}, "2"))
	workflow.AfterFlow(true, FlowCallback(false, []string{"1", "2"}, []string{"3", "4", "5", "6"}))
	process := workflow.Process("TestAllCallbackConnect")
	process.AfterStep(true, ErrorResultPrinter)
	process.BeforeProcess(true, ProcCallback(false, []string{"1", "2"}, []string{}, "3"))
	process.AfterProcess(true, ProcCallback(false, []string{"1", "2", "3"}, []string{"4", "6"}))
	process.BeforeStep(true, StepCallback(false, []string{"1", "2", "3"}, []string{}, "4"))
	process.AfterStep(true, StepCallback(false, []string{"1", "2", "3", "4", "6"}, []string{}))
	process.AliasStep(CtxChecker(false, []string{"1", "2", "3", "4"}, []string{}, "6"), "1")
	result := flow.DoneFlow("TestAllCallbackConnect", map[string]any{"1": "1"})
	if !result.Success() {
		for _, exception := range result.Exceptions() {
			t.Errorf("%s failed: %s\n", result.GetName(), exception)
		}
	}
	if atomic.LoadInt64(&current) != 7 {
		t.Errorf("execute 7 step, but current = %d", current)
	}
}
