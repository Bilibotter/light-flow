package test

import (
	"fmt"
	flow "gitee.com/MetaphysicCoding/light-flow"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func GenerateNoDelayStep(i int) func(ctx *flow.Context) (any, error) {
	return func(ctx *flow.Context) (any, error) {
		ctx.Set("step", i)
		atomic.AddInt64(&current, 1)
		return i, nil
	}
}

func GenerateNoDelayProcessor(info *flow.StepInfo) bool {
	atomic.AddInt64(&current, 1)
	return true
}

func NoDelayContextStep(ctx *flow.Context) (any, error) {
	addr, _ := ctx.Get(addrKey)
	atomic.AddInt64(addr.(*int64), 1)
	ctx.Exposed("foo", 1)
	return nil, nil
}

func TestTestMultipleConcurrentDependContext(t *testing.T) {
	defer resetCtx()
	factory := flow.AddFlowFactory("TestTestMultipleConcurrentDependContext")
	process := factory.AddProcess("TestTestMultipleConcurrentDependContext", nil)
	process.AddStepWithAlias("-1", ChangeCtxStepFunc(&ctx1))
	for i := 0; i < 10000; i++ {
		process.AddStepWithAlias(strconv.Itoa(i), NoDelayContextStep, "-1")
	}
	features := flow.DoneFlow("TestTestMultipleConcurrentDependContext", map[string]any{addrKey: &current})
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] run fail", name)
		}
	}

	if atomic.LoadInt64(&ctx1) != 10001 {
		t.Errorf("excute 10001 step, but current = %d", current)
	}
}

func TestMultipleConcurrentContext(t *testing.T) {
	defer resetCurrent()
	factory := flow.AddFlowFactory("TestMultipleConcurrentContext")
	process := factory.AddProcess("TestMultipleConcurrentContext", nil)
	for i := 0; i < 10000; i++ {
		process.AddStepWithAlias(strconv.Itoa(i), NoDelayContextStep)
	}
	features := flow.DoneFlow("TestMultipleConcurrentContext", map[string]any{addrKey: &current})
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] run fail", name)
		}
	}

	if atomic.LoadInt64(&current) != 10000 {
		t.Errorf("excute 10000 step, but current = %d", current)
	}
}

func TestMultipleConcurrentProcess(t *testing.T) {
	defer resetCurrent()
	factory := flow.AddFlowFactory("TestMultipleConcurrentProcess")
	for i := 0; i < 100; i++ {
		process := factory.AddProcess("TestMultipleConcurrentProcess"+strconv.Itoa(i), nil)
		process.AddPreProcessor(GenerateNoDelayProcessor)
		process.AddPostProcessor(GenerateNoDelayProcessor)
		for j := 0; j < 100; j++ {
			key := strconv.Itoa(i) + "|" + strconv.Itoa(j)
			process.AddStepWithAlias(key, GenerateNoDelayStep(i*1000+j))
		}
	}
	features := flow.DoneFlow("TestMultipleConcurrentProcess", nil)
	for name, feature := range features {
		if !feature.Success() {
			t.Errorf("process[%s] run fail", name)
		}
	}

	if atomic.LoadInt64(&current) != 30000 {
		t.Errorf("excute 30000 step, but current = %d", current)
	}
}

func TestMultipleConcurrentStepWithProcessor(t *testing.T) {
	defer resetCurrent()
	factory := flow.AddFlowFactory("TestMultipleConcurrentStepWithProcessor")
	process := factory.AddProcess("TestMultipleConcurrentStepWithProcessor", nil)
	process.AddPreProcessor(GenerateNoDelayProcessor)
	process.AddPostProcessor(GenerateNoDelayProcessor)
	for i := 0; i < 10000; i++ {
		process.AddStepWithAlias(strconv.Itoa(i), GenerateNoDelayStep(i))
	}
	features := flow.DoneFlow("TestMultipleConcurrentStepWithProcessor", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] run fail", name)
		}
	}

	if atomic.LoadInt64(&current) != 30000 {
		t.Errorf("excute 30000 step, but current = %d", current)
	}
}

func TestMultipleConcurrentStep(t *testing.T) {
	defer resetCurrent()
	factory := flow.AddFlowFactory("TestMultipleConcurrentStep")
	process := factory.AddProcess("TestMultipleConcurrentStep", nil)
	for i := 0; i < 10000; i++ {
		process.AddStepWithAlias(strconv.Itoa(i), GenerateNoDelayStep(i))
	}
	features := flow.DoneFlow("TestMultipleConcurrentStep", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] run fail", name)
		}
	}

	if atomic.LoadInt64(&current) != 10000 {
		t.Errorf("excute 10000 step, but current = %d", current)
	}
}

func TestMultipleConcurrentDependStep(t *testing.T) {
	defer resetCurrent()
	factory := flow.AddFlowFactory("TestMultipleConcurrentDependStep")
	process := factory.AddProcess("TestMultipleConcurrentDependStep", nil)
	for i := 0; i < 100; i++ {
		prev := ""
		for j := 0; j < 100; j++ {
			key := strconv.Itoa(i) + "|" + strconv.Itoa(j)
			if len(prev) == 0 {
				process.AddStepWithAlias(key, GenerateNoDelayStep(i*1000+j))
			} else {
				process.AddStepWithAlias(key, GenerateNoDelayStep(i*1000+j), prev)
			}
			prev = key
		}
	}
	features := flow.DoneFlow("TestMultipleConcurrentDependStep", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] run fail", name)
		}
	}

	if atomic.LoadInt64(&current) != 10000 {
		t.Errorf("excute 1000 step, but current = %d", current)
	}
}

func TestConcurrentSameFlow(t *testing.T) {
	defer resetCurrent()
	factory := flow.AddFlowFactory("TestConcurrentSameFlow")
	process := factory.AddProcess("TestConcurrentSameFlow", nil)
	for i := 0; i < 100; i++ {
		process.AddStepWithAlias(strconv.Itoa(i), GenerateNoDelayStep(i))
	}
	flows := make([]flow.WorkFlowCtrl, 0, 1000)
	for i := 0; i < 1000; i++ {
		flows = append(flows, flow.AsyncFlow("TestConcurrentSameFlow", nil))
	}
	for _, flowing := range flows {
		features := flowing.Done()
		for name, feature := range features {
			if !feature.Success() {
				t.Errorf("process[%s] run fail", name)
			}
		}
	}

	if atomic.LoadInt64(&current) != 100*1000 {
		t.Errorf("excute 100000 step, but current = %d", current)
	}
}
