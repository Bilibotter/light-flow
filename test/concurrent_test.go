package test

import (
	"fmt"
	flow "gitee.com/MetaphysicCoding/light-flow"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func GenerateNoDelayStep(i int) func(ctx flow.Context) (any, error) {
	return func(ctx flow.Context) (any, error) {
		ctx.Set("step", i)
		atomic.AddInt64(&current, 1)
		return i, nil
	}
}

func GenerateNoDelayProcessor(info *flow.StepInfo) (bool, error) {
	atomic.AddInt64(&current, 1)
	return true, nil
}

func NoDelayContextStep(ctx flow.Context) (any, error) {
	addr, _ := ctx.Get(addrKey)
	atomic.AddInt64(addr.(*int64), 1)
	ctx.Set("foo", 1)
	return nil, nil
}

func TestQuickStepConcurrent(t *testing.T) {
	workflow := flow.RegisterFlow("TestQuickStepConcurrent")
	proc := make([]*flow.ProcessMeta, 0, 100)
	for i := 0; i < 100; i++ {
		p := workflow.Process("TestQuickStepConcurrent" + strconv.Itoa(i))
		proc = append(proc, p)
	}
	for i := 0; i < 62; i++ {
		for _, p := range proc {
			p.AliasStep(strconv.Itoa(i), func(ctx flow.Context) (any, error) {
				return nil, nil
			})
		}
	}
	flow.DoneFlow("TestQuickStepConcurrent", nil)
	flow.DoneFlow("TestQuickStepConcurrent", nil)
	flow.DoneFlow("TestQuickStepConcurrent", nil)
	flow.DoneFlow("TestQuickStepConcurrent", nil)
}

func TestTestMultipleConcurrentDependContext(t *testing.T) {
	defer resetCtx()
	factory := flow.RegisterFlow("TestTestMultipleConcurrentDependContext")
	process := factory.Process("TestTestMultipleConcurrentDependContext")
	process.AliasStep("-1", ChangeCtxStepFunc(&ctx1))
	for i := 0; i < 61; i++ {
		process.AliasStep(strconv.Itoa(i), NoDelayContextStep, "-1")
	}
	features := flow.DoneFlow("TestTestMultipleConcurrentDependContext", map[string]any{addrKey: &current})
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] run fail", feature.Name)
		}
	}

	if atomic.LoadInt64(&ctx1) != 62 {
		t.Errorf("execute 62 step, but current = %d", current)
	}
}

func TestMultipleConcurrentContext(t *testing.T) {
	defer resetCurrent()
	factory := flow.RegisterFlow("TestMultipleConcurrentContext")
	process := factory.Process("TestMultipleConcurrentContext")
	for i := 0; i < 62; i++ {
		process.AliasStep(strconv.Itoa(i), NoDelayContextStep)
	}
	features := flow.DoneFlow("TestMultipleConcurrentContext", map[string]any{addrKey: &current})
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] run fail", feature.Name)
		}
	}

	if atomic.LoadInt64(&current) != 62 {
		t.Errorf("execute 62 step, but current = %d", current)
	}
}

func TestMultipleConcurrentProcess(t *testing.T) {
	defer resetCurrent()
	factory := flow.RegisterFlow("TestMultipleConcurrentProcess")
	for i := 0; i < 100; i++ {
		process := factory.Process("TestMultipleConcurrentProcess" + strconv.Itoa(i))
		process.BeforeStep(true, GenerateNoDelayProcessor)
		process.AfterStep(true, GenerateNoDelayProcessor)
		for j := 0; j < 62; j++ {
			key := strconv.Itoa(i) + "|" + strconv.Itoa(j)
			process.AliasStep(key, GenerateNoDelayStep(i*1000+j))
		}
	}
	features := flow.DoneFlow("TestMultipleConcurrentProcess", nil)
	for _, feature := range features.Futures() {
		if !feature.Success() {
			t.Errorf("process[%s] run fail", feature.Name)
		}
	}

	if atomic.LoadInt64(&current) != 62*100*3 {
		t.Errorf("execute %d step, but current = %d", 62*100*3, current)
	}
}

func TestMultipleConcurrentStepWithProcessor(t *testing.T) {
	defer resetCurrent()
	factory := flow.RegisterFlow("TestMultipleConcurrentStepWithProcessor")
	process := factory.Process("TestMultipleConcurrentStepWithProcessor")
	process.BeforeStep(true, GenerateNoDelayProcessor)
	process.AfterStep(true, GenerateNoDelayProcessor)
	for i := 0; i < 62; i++ {
		process.AliasStep(strconv.Itoa(i), GenerateNoDelayStep(i))
	}
	features := flow.DoneFlow("TestMultipleConcurrentStepWithProcessor", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] run fail", feature.Name)
		}
	}

	if atomic.LoadInt64(&current) != 62*3 {
		t.Errorf("execute %d step, but current = %d", 62*100*3, current)
	}
}

func TestMultipleConcurrentStep(t *testing.T) {
	defer resetCurrent()
	factory := flow.RegisterFlow("TestMultipleConcurrentStep")
	process := factory.Process("TestMultipleConcurrentStep")
	for i := 0; i < 62; i++ {
		process.AliasStep(strconv.Itoa(i), GenerateNoDelayStep(i))
	}
	features := flow.DoneFlow("TestMultipleConcurrentStep", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] run fail", feature.Name)
		}
	}

	if atomic.LoadInt64(&current) != 62 {
		t.Errorf("execute 62 step, but current = %d", current)
	}
}

func TestMultipleConcurrentDependStep(t *testing.T) {
	defer resetCurrent()
	factory := flow.RegisterFlow("TestMultipleConcurrentDependStep")
	process := factory.Process("TestMultipleConcurrentDependStep")
	for i := 0; i < 20; i++ {
		prev := ""
		for j := 0; j < 3; j++ {
			key := strconv.Itoa(i) + "|" + strconv.Itoa(j)
			if len(prev) == 0 {
				process.AliasStep(key, GenerateNoDelayStep(i*1000+j))
			} else {
				process.AliasStep(key, GenerateNoDelayStep(i*1000+j), prev)
			}
			prev = key
		}
	}
	features := flow.DoneFlow("TestMultipleConcurrentDependStep", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] run fail", feature.Name)
		}
	}

	if atomic.LoadInt64(&current) != 60 {
		t.Errorf("execute 60 step, but current = %d", current)
	}
}

func TestConcurrentSameFlow(t *testing.T) {
	defer resetCurrent()
	factory := flow.RegisterFlow("TestConcurrentSameFlow")
	process := factory.Process("TestConcurrentSameFlow")
	for i := 0; i < 62; i++ {
		process.AliasStep(strconv.Itoa(i), GenerateNoDelayStep(i))
	}
	flows := make([]flow.FlowController, 0, 1000)
	for i := 0; i < 1000; i++ {
		flows = append(flows, flow.AsyncFlow("TestConcurrentSameFlow", nil))
	}
	for _, flowing := range flows {
		features := flowing.Done()
		for _, feature := range features {
			if !feature.Success() {
				t.Errorf("process[%s] run fail", feature.Name)
			}
		}
	}

	if atomic.LoadInt64(&current) != 62000 {
		t.Errorf("execute 62000 step, but current = %d", current)
	}
}
