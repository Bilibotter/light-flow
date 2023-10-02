package test

import (
	"fmt"
	flow "gitee.com/MetaphysicCoding/light-flow"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDependMergedWithEmptyHead(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestDependMergedWithEmptyHead1")
	process1 := factory1.AddProcess("TestDependMergedWithEmptyHead1", nil)
	process1.AddStepWithAlias("1", GenerateStep(1))
	process1.AddStepWithAlias("2", GenerateStep(2), "1")
	process1.AddStepWithAlias("3", GenerateStep(3), "2")
	process1.AddStepWithAlias("4", GenerateStep(4), "3")
	factory2 := flow.RegisterFlow("TestDependMergedWithEmptyHead2")
	process2 := factory2.AddProcess("TestDependMergedWithEmptyHead2", nil)
	process2.Merge("TestDependMergedWithEmptyHead1")
	process2.AddStepWithAlias("5", GenerateStep(5), "4")
	features := flow.DoneFlow("TestDependMergedWithEmptyHead2", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestDependMergedWithEmptyHead1", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 9 {
		t.Errorf("execute 9 step, but current = %d", current)
	}
}

func TestDependMergedWithNotEmptyHead(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestDependMergedWithNotEmptyHead1")
	process1 := factory1.AddProcess("TestDependMergedWithNotEmptyHead1", nil)
	process1.AddStepWithAlias("1", GenerateStep(1))
	process1.AddStepWithAlias("2", GenerateStep(2), "1")
	process1.AddStepWithAlias("3", GenerateStep(3), "2")
	process1.AddStepWithAlias("4", GenerateStep(4), "3")
	factory2 := flow.RegisterFlow("TestDependMergedWithNotEmptyHead2")
	process2 := factory2.AddProcess("TestDependMergedWithNotEmptyHead2", nil)
	process2.AddStepWithAlias("0", GenerateStep(0))
	process2.Merge("TestDependMergedWithNotEmptyHead1")
	process2.AddStepWithAlias("5", GenerateStep(5), "4")
	features := flow.DoneFlow("TestDependMergedWithNotEmptyHead2", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestDependMergedWithNotEmptyHead1", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 10 {
		t.Errorf("execute 10 step, but current = %d", current)
	}
}

func TestMergeEmpty(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeEmpty1")
	factory1.AddProcess("TestMergeEmpty1", nil)
	factory2 := flow.RegisterFlow("TestMergeEmpty2")
	process2 := factory2.AddProcess("TestMergeEmpty2", nil)
	process2.AddStepWithAlias("1", GenerateStep(1))
	process2.AddStepWithAlias("2", GenerateStep(2), "1")
	process2.AddStepWithAlias("3", GenerateStep(3), "2")
	process2.AddStepWithAlias("4", GenerateStep(4), "3")
	process2.Merge("TestMergeEmpty1")
	features := flow.DoneFlow("TestMergeEmpty2", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestEmptyMerge(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestEmptyMerge1")
	process1 := factory1.AddProcess("TestEmptyMerge1", nil)
	process1.AddStepWithAlias("1", GenerateStep(1))
	process1.AddStepWithAlias("2", GenerateStep(2), "1")
	process1.AddStepWithAlias("3", GenerateStep(3), "2")
	process1.AddStepWithAlias("4", GenerateStep(4), "3")
	factory2 := flow.RegisterFlow("TestEmptyMerge2")
	process2 := factory2.AddProcess("TestEmptyMerge2", nil)
	process2.Merge("TestEmptyMerge1")
	features := flow.DoneFlow("TestEmptyMerge2", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestEmptyMerge1", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 8 {
		t.Errorf("execute 8 step, but current = %d", current)
	}
}

func TestMergeAbsolutelyDifferent(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeAbsolutelyDifferent1")
	process1 := factory1.AddProcess("TestMergeAbsolutelyDifferent1", nil)
	process1.AddStepWithAlias("1", GenerateStep(1))
	process1.AddStepWithAlias("2", GenerateStep(2), "1")
	process1.AddStepWithAlias("3", GenerateStep(3), "2")
	process1.AddStepWithAlias("4", GenerateStep(4), "3")
	factory2 := flow.RegisterFlow("TestMergeAbsolutelyDifferent2")
	process2 := factory2.AddProcess("TestMergeAbsolutelyDifferent2", nil)
	process2.AddStepWithAlias("11", GenerateStep(11))
	process2.AddStepWithAlias("12", GenerateStep(12), "11")
	process2.AddStepWithAlias("13", GenerateStep(13), "12")
	process2.AddStepWithAlias("14", GenerateStep(14), "13")
	process2.Merge("TestMergeAbsolutelyDifferent1")
	features := flow.DoneFlow("TestMergeAbsolutelyDifferent2", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 8 {
		t.Errorf("execute 8 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestMergeAbsolutelyDifferent1", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 12 {
		t.Errorf("execute 12 step, but current = %d", current)
	}
}

func TestMergeLayerSame(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeLayerSame1")
	process1 := factory1.AddProcess("TestMergeLayerSame1", nil)
	process1.AddStepWithAlias("1", GenerateStep(1))
	process1.AddStepWithAlias("2", GenerateStep(2), "1")
	process1.AddStepWithAlias("3", GenerateStep(3), "1")
	process1.AddStepWithAlias("4", GenerateStep(4), "3")
	factory2 := flow.RegisterFlow("TestMergeLayerSame2")
	process2 := factory2.AddProcess("TestMergeLayerSame2", nil)
	process2.AddStepWithAlias("1", GenerateStep(1))
	process2.AddStepWithAlias("3", GenerateStep(3), "1")
	process2.AddStepWithAlias("4", GenerateStep(4), "3")
	process2.AddStepWithAlias("5", GenerateStep(5), "4")
	process2.Merge("TestMergeLayerSame1")
	features := flow.DoneFlow("TestMergeLayerSame2", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestMergeLayerSame1", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 9 {
		t.Errorf("execute 9 step, but current = %d", current)
	}
}

func TestMergeLayerDec(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeLayerDec1")
	process1 := factory1.AddProcess("TestMergeLayerDec1", nil)
	process1.AddStepWithAlias("1", GenerateStep(1))
	process1.AddStepWithAlias("2", GenerateStep(2), "1")
	process1.AddStepWithAlias("3", GenerateStep(3), "1")
	process1.AddStepWithAlias("4", GenerateStep(4), "3")
	factory2 := flow.RegisterFlow("TestMergeLayerDec2")
	process2 := factory2.AddProcess("TestMergeLayerDec2", nil)
	process2.AddStepWithAlias("1", GenerateStep(1))
	process2.AddStepWithAlias("4", GenerateStep(4), "1")
	process2.AddStepWithAlias("5", GenerateStep(5), "4")
	process2.AddStepWithAlias("6", GenerateStep(6), "5")
	process2.Merge("TestMergeLayerDec1")
	features := flow.DoneFlow("TestMergeLayerDec2", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestMergeLayerDec1", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 10 {
		t.Errorf("execute 10 step, but current = %d", current)
	}
}

func TestMergeLayerInc(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeLayerInc1")
	process1 := factory1.AddProcess("TestMergeLayerInc1", nil)
	process1.AddStepWithAlias("1", GenerateStep(1))
	process1.AddStepWithAlias("2", GenerateStep(2), "1")
	process1.AddStepWithAlias("3", GenerateStep(3), "1")
	process1.AddStepWithAlias("4", GenerateStep(4), "3")
	factory2 := flow.RegisterFlow("TestMergeLayerInc2")
	process2 := factory2.AddProcess("TestMergeLayerInc2", nil)
	process2.AddStepWithAlias("1", GenerateStep(1))
	process2.AddStepWithAlias("3", GenerateStep(3), "1")
	process2.AddStepWithAlias("5", GenerateStep(5), "3")
	process2.Merge("TestMergeLayerInc1")
	features := flow.DoneFlow("TestMergeLayerInc2", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
}

func TestMergeSomeSame(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeSomeSame1")
	process1 := factory1.AddProcess("TestMergeSomeSame1", nil)
	process1.AddStepWithAlias("1", GenerateStep(1))
	process1.AddStepWithAlias("2", GenerateStep(2), "1")
	process1.AddStepWithAlias("3", GenerateStep(3), "1")
	process1.AddStepWithAlias("4", GenerateStep(4), "3")
	factory2 := flow.RegisterFlow("TestMergeSomeSame2")
	process2 := factory2.AddProcess("TestMergeSomeSame2", nil)
	process2.AddStepWithAlias("1", GenerateStep(1))
	process2.AddStepWithAlias("5", GenerateStep(5), "1")
	process2.AddStepWithAlias("3", GenerateStep(3), "5")
	process2.AddStepWithAlias("4", GenerateStep(4), "5")
	process2.Merge("TestMergeSomeSame1")
	features := flow.DoneFlow("TestMergeSomeSame2", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
}

func TestMergeSame(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeSame1")
	process1 := factory1.AddProcess("TestMergeSame1", nil)
	process1.AddStepWithAlias("1", GenerateStep(1))
	process1.AddStepWithAlias("2", GenerateStep(2), "1")
	process1.AddStepWithAlias("3", GenerateStep(3), "2")
	process1.AddStepWithAlias("4", GenerateStep(4), "3")
	factory2 := flow.RegisterFlow("TestMergeSame2")
	process2 := factory2.AddProcess("TestMergeSame2", nil)
	process2.AddStepWithAlias("1", GenerateStep(1))
	process2.AddStepWithAlias("2", GenerateStep(2), "1")
	process2.AddStepWithAlias("3", GenerateStep(3), "2")
	process2.AddStepWithAlias("4", GenerateStep(4), "3")
	process2.Merge("TestMergeSame1")
	features := flow.DoneFlow("TestMergeSame2", nil)
	for name, feature := range features {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestMergeCircle(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeCircle1")
	process1 := factory1.AddProcess("TestMergeCircle1", nil)
	process1.AddStepWithAlias("1", GenerateStep(1))
	process1.AddStepWithAlias("2", GenerateStep(2), "1")
	factory2 := flow.RegisterFlow("TestMergeCircle2")
	process2 := factory2.AddProcess("TestMergeCircle2", nil)
	process2.AddStepWithAlias("2", GenerateStep(2))
	process2.AddStepWithAlias("1", GenerateStep(1), "2")
	defer func() {
		if r := recover(); r != nil {
			t.Logf("circle detect success info: %v", r)
		} else {
			t.Errorf("circle detect fail")
		}
	}()
	process2.Merge("TestMergeCircle1")
}

func TestMergeLongCircleLongCircle(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeLongCircle1")
	process1 := factory1.AddProcess("TestMergeLongCircle1", nil)
	process1.AddStepWithAlias("1", GenerateStep(1))
	process1.AddStepWithAlias("2", GenerateStep(2), "1")
	process1.AddStepWithAlias("3", GenerateStep(3), "2")
	process1.AddStepWithAlias("4", GenerateStep(4), "3")
	process1.AddStepWithAlias("5", GenerateStep(5), "4")
	factory2 := flow.RegisterFlow("TestMergeLongCircle2")
	process2 := factory2.AddProcess("TestMergeLongCircle2", nil)
	process2.AddStepWithAlias("2", GenerateStep(2))
	process2.AddStepWithAlias("5", GenerateStep(5), "2")
	process2.AddStepWithAlias("3", GenerateStep(3), "5")
	process2.AddStepWithAlias("4", GenerateStep(4), "3")
	defer func() {
		if r := recover(); r != nil {
			t.Logf("circle detect success info: %v", r)
		} else {
			t.Errorf("circle detect fail")
		}
	}()
	process2.Merge("TestMergeLongCircle1")
}
