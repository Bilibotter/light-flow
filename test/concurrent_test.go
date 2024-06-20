package test

import (
	flow "github.com/Bilibotter/light-flow"
	"strconv"
	"sync/atomic"
	"testing"
)

func GenerateNoDelayStep(i int) func(ctx flow.Step) (any, error) {
	return func(ctx flow.Step) (any, error) {
		ctx.Set("step", i)
		atomic.AddInt64(&current, 1)
		return i, nil
	}
}

func GenerateNoDelayProcessor(info flow.Step) (bool, error) {
	atomic.AddInt64(&current, 1)
	return true, nil
}

func NoDelayContextStep(ctx flow.Step) (any, error) {
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
			p.NameStep(func(ctx flow.Step) (any, error) {
				return nil, nil
			}, strconv.Itoa(i))
		}
	}
	flow.DoneFlow("TestQuickStepConcurrent", nil)
	flow.DoneFlow("TestQuickStepConcurrent", nil)
	flow.DoneFlow("TestQuickStepConcurrent", nil)
	flow.DoneFlow("TestQuickStepConcurrent", nil)
}

func TestTestMultipleConcurrentDependContext(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestTestMultipleConcurrentDependContext")
	process := workflow.Process("TestTestMultipleConcurrentDependContext")
	process.NameStep(ChangeCtxStepFunc(&current), "-1")
	for i := 0; i < 61; i++ {
		process.NameStep(NoDelayContextStep, strconv.Itoa(i), "-1")
	}
	workflow.AfterFlow(false, CheckResult(t, 62, flow.Success))
	flow.DoneFlow("TestTestMultipleConcurrentDependContext", map[string]any{addrKey: &current})
}

func TestMultipleConcurrentContext(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleConcurrentContext")
	process := workflow.Process("TestMultipleConcurrentContext")
	for i := 0; i < 62; i++ {
		process.NameStep(NoDelayContextStep, strconv.Itoa(i))
	}
	workflow.AfterFlow(false, CheckResult(t, 62, flow.Success))
	flow.DoneFlow("TestMultipleConcurrentContext", map[string]any{addrKey: &current})
}

func TestMultipleConcurrentProcess(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleConcurrentProcess")
	for i := 0; i < 100; i++ {
		process := workflow.Process("TestMultipleConcurrentProcess" + strconv.Itoa(i))
		process.BeforeStep(true, GenerateNoDelayProcessor)
		process.AfterStep(true, GenerateNoDelayProcessor)
		for j := 0; j < 62; j++ {
			key := strconv.Itoa(i) + "|" + strconv.Itoa(j)
			process.NameStep(GenerateNoDelayStep(i*1000+j), key)
		}
	}
	workflow.AfterFlow(false, CheckResult(t, 62*100*3, flow.Success))
	flow.DoneFlow("TestMultipleConcurrentProcess", nil)
}

func TestMultipleConcurrentStepWithProcessor(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleConcurrentStepWithProcessor")
	process := workflow.Process("TestMultipleConcurrentStepWithProcessor")
	process.BeforeStep(true, GenerateNoDelayProcessor)
	process.AfterStep(true, GenerateNoDelayProcessor)
	for i := 0; i < 62; i++ {
		process.NameStep(GenerateNoDelayStep(i), strconv.Itoa(i))
	}
	workflow.AfterFlow(false, CheckResult(t, 62*3, flow.Success))
	flow.DoneFlow("TestMultipleConcurrentStepWithProcessor", nil)
}

func TestMultipleConcurrentStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleConcurrentStep")
	process := workflow.Process("TestMultipleConcurrentStep")
	for i := 0; i < 62; i++ {
		process.NameStep(GenerateNoDelayStep(i), strconv.Itoa(i))
	}
	workflow.AfterFlow(false, CheckResult(t, 62, flow.Success))
	flow.DoneFlow("TestMultipleConcurrentStep", nil)
}

func TestMultipleConcurrentDependStep(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMultipleConcurrentDependStep")
	process := workflow.Process("TestMultipleConcurrentDependStep")
	for i := 0; i < 20; i++ {
		prev := ""
		for j := 0; j < 3; j++ {
			key := strconv.Itoa(i) + "|" + strconv.Itoa(j)
			if len(prev) == 0 {
				process.NameStep(GenerateNoDelayStep(i*1000+j), key)
			} else {
				process.NameStep(GenerateNoDelayStep(i*1000+j), key, prev)
			}
			prev = key
		}
	}
	workflow.AfterFlow(false, CheckResult(t, 60, flow.Success))
	flow.DoneFlow("TestMultipleConcurrentDependStep", nil)
}

func TestConcurrentSameFlow(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestConcurrentSameFlow")
	process := workflow.Process("TestConcurrentSameFlow")
	for i := 0; i < 62; i++ {
		process.NameStep(GenerateNoDelayStep(i), strconv.Itoa(i))
	}
	flows := make([]flow.FlowController, 0, 1000)
	for i := 0; i < 1000; i++ {
		flows = append(flows, flow.AsyncFlow("TestConcurrentSameFlow", nil))
	}
	for _, flowing := range flows {
		if !flowing.Done().Success() {
			t.Errorf("flow fail")
		}
	}

	if atomic.LoadInt64(&current) != 62000 {
		t.Errorf("execute 62000 step, but current = %d", current)
	}
}
