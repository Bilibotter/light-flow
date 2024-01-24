package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDependMergedWithEmptyHead(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestDependMergedWithEmptyHead1")
	process1 := factory1.Process("TestDependMergedWithEmptyHead1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "2")
	process1.AliasStep(GenerateStep(4), "4", "3")
	factory2 := flow.RegisterFlow("TestDependMergedWithEmptyHead2")
	process2 := factory2.Process("TestDependMergedWithEmptyHead2")
	process2.Merge("TestDependMergedWithEmptyHead1")
	process2.AliasStep(GenerateStep(5), "5", "4")
	features := flow.DoneFlow("TestDependMergedWithEmptyHead2", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestDependMergedWithEmptyHead1", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 9 {
		t.Errorf("execute 9 step, but current = %d", current)
	}
}

func TestDependMergedWithNotEmptyHead(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestDependMergedWithNotEmptyHead1")
	process1 := factory1.Process("TestDependMergedWithNotEmptyHead1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "2")
	process1.AliasStep(GenerateStep(4), "4", "3")
	factory2 := flow.RegisterFlow("TestDependMergedWithNotEmptyHead2")
	process2 := factory2.Process("TestDependMergedWithNotEmptyHead2")
	process2.AliasStep(GenerateStep(0), "0")
	process2.Merge("TestDependMergedWithNotEmptyHead1")
	process2.AliasStep(GenerateStep(5), "5", "4")
	features := flow.DoneFlow("TestDependMergedWithNotEmptyHead2", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestDependMergedWithNotEmptyHead1", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 10 {
		t.Errorf("execute 10 step, but current = %d", current)
	}
}

func TestMergeEmpty(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeEmpty1")
	factory1.Process("TestMergeEmpty1")
	factory2 := flow.RegisterFlow("TestMergeEmpty2")
	process2 := factory2.Process("TestMergeEmpty2")
	process2.AliasStep(GenerateStep(1), "1")
	process2.AliasStep(GenerateStep(2), "2", "1")
	process2.AliasStep(GenerateStep(3), "3", "2")
	process2.AliasStep(GenerateStep(4), "4", "3")
	process2.Merge("TestMergeEmpty1")
	features := flow.DoneFlow("TestMergeEmpty2", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestEmptyMerge(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestEmptyMerge1")
	process1 := factory1.Process("TestEmptyMerge1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "2")
	process1.AliasStep(GenerateStep(4), "4", "3")
	factory2 := flow.RegisterFlow("TestEmptyMerge2")
	process2 := factory2.Process("TestEmptyMerge2")
	process2.Merge("TestEmptyMerge1")
	features := flow.DoneFlow("TestEmptyMerge2", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestEmptyMerge1", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 8 {
		t.Errorf("execute 8 step, but current = %d", current)
	}
}

func TestMergeAbsolutelyDifferent(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeAbsolutelyDifferent1")
	process1 := factory1.Process("TestMergeAbsolutelyDifferent1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "2")
	process1.AliasStep(GenerateStep(4), "4", "3")
	factory2 := flow.RegisterFlow("TestMergeAbsolutelyDifferent2")
	process2 := factory2.Process("TestMergeAbsolutelyDifferent2")
	process2.AliasStep(GenerateStep(11), "11")
	process2.AliasStep(GenerateStep(12), "12", "11")
	process2.AliasStep(GenerateStep(13), "13", "12")
	process2.AliasStep(GenerateStep(14), "14", "13")
	process2.Merge("TestMergeAbsolutelyDifferent1")
	features := flow.DoneFlow("TestMergeAbsolutelyDifferent2", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 8 {
		t.Errorf("execute 8 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestMergeAbsolutelyDifferent1", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 12 {
		t.Errorf("execute 12 step, but current = %d", current)
	}
}

func TestMergeLayerSame(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeLayerSame1")
	process1 := factory1.Process("TestMergeLayerSame1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "1")
	process1.AliasStep(GenerateStep(4), "4", "3")
	factory2 := flow.RegisterFlow("TestMergeLayerSame2")
	process2 := factory2.Process("TestMergeLayerSame2")
	process2.AliasStep(GenerateStep(1), "1")
	process2.AliasStep(GenerateStep(3), "3", "1")
	process2.AliasStep(GenerateStep(4), "4", "3")
	process2.AliasStep(GenerateStep(5), "5", "4")
	process2.Merge("TestMergeLayerSame1")
	features := flow.DoneFlow("TestMergeLayerSame2", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestMergeLayerSame1", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 9 {
		t.Errorf("execute 9 step, but current = %d", current)
	}
}

func TestMergeLayerDec(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeLayerDec1")
	process1 := factory1.Process("TestMergeLayerDec1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "1")
	process1.AliasStep(GenerateStep(4), "4", "3")
	factory2 := flow.RegisterFlow("TestMergeLayerDec2")
	process2 := factory2.Process("TestMergeLayerDec2")
	process2.AliasStep(GenerateStep(1), "1")
	process2.AliasStep(GenerateStep(4), "4", "1")
	process2.AliasStep(GenerateStep(5), "5", "4")
	process2.AliasStep(GenerateStep(6), "6", "5")
	process2.Merge("TestMergeLayerDec1")
	features := flow.DoneFlow("TestMergeLayerDec2", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 6 {
		t.Errorf("execute 6 step, but current = %d", current)
	}
	features = flow.DoneFlow("TestMergeLayerDec1", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 10 {
		t.Errorf("execute 10 step, but current = %d", current)
	}
}

func TestMergeLayerInc(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeLayerInc1")
	process1 := factory1.Process("TestMergeLayerInc1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "1")
	process1.AliasStep(GenerateStep(4), "4", "3")
	factory2 := flow.RegisterFlow("TestMergeLayerInc2")
	process2 := factory2.Process("TestMergeLayerInc2")
	process2.AliasStep(GenerateStep(1), "1")
	process2.AliasStep(GenerateStep(3), "3", "1")
	process2.AliasStep(GenerateStep(5), "5", "3")
	process2.Merge("TestMergeLayerInc1")
	features := flow.DoneFlow("TestMergeLayerInc2", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
}

func TestMergeSomeSame(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeSomeSame1")
	process1 := factory1.Process("TestMergeSomeSame1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "1")
	process1.AliasStep(GenerateStep(4), "4", "3")
	factory2 := flow.RegisterFlow("TestMergeSomeSame2")
	process2 := factory2.Process("TestMergeSomeSame2")
	process2.AliasStep(GenerateStep(1), "1")
	process2.AliasStep(GenerateStep(5), "5", "1")
	process2.AliasStep(GenerateStep(3), "3", "5")
	process2.AliasStep(GenerateStep(4), "4", "5")
	process2.Merge("TestMergeSomeSame1")
	features := flow.DoneFlow("TestMergeSomeSame2", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 5 {
		t.Errorf("execute 5 step, but current = %d", current)
	}
}

func TestMergeSame(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeSame1")
	process1 := factory1.Process("TestMergeSame1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "2")
	process1.AliasStep(GenerateStep(4), "4", "3")
	factory2 := flow.RegisterFlow("TestMergeSame2")
	process2 := factory2.Process("TestMergeSame2")
	process2.AliasStep(GenerateStep(1), "1")
	process2.AliasStep(GenerateStep(2), "2", "1")
	process2.AliasStep(GenerateStep(3), "3", "2")
	process2.AliasStep(GenerateStep(4), "4", "3")
	process2.Merge("TestMergeSame1")
	features := flow.DoneFlow("TestMergeSame2", nil)
	for _, feature := range features.Futures() {
		explain := strings.Join(feature.ExplainStatus(), ", ")
		fmt.Printf("process[%s] explain=%s\n", feature.Name, explain)
		if !feature.Success() {
			t.Errorf("process[%s] fail", feature.Name)
		}
	}
	if atomic.LoadInt64(&current) != 4 {
		t.Errorf("execute 4 step, but current = %d", current)
	}
}

func TestMergeCircle(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestMergeCircle1")
	process1 := factory1.Process("TestMergeCircle1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	factory2 := flow.RegisterFlow("TestMergeCircle2")
	process2 := factory2.Process("TestMergeCircle2")
	process2.AliasStep(GenerateStep(2), "2")
	process2.AliasStep(GenerateStep(1), "1", "2")
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
	process1 := factory1.Process("TestMergeLongCircle1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "2")
	process1.AliasStep(GenerateStep(4), "4", "3")
	process1.AliasStep(GenerateStep(5), "5", "4")
	factory2 := flow.RegisterFlow("TestMergeLongCircle2")
	process2 := factory2.Process("TestMergeLongCircle2")
	process2.AliasStep(GenerateStep(2), "2")
	process2.AliasStep(GenerateStep(5), "5", "2")
	process2.AliasStep(GenerateStep(3), "3", "5")
	process2.AliasStep(GenerateStep(4), "4", "3")
	defer func() {
		if r := recover(); r != nil {
			t.Logf("circle detect success info: %v", r)
		} else {
			t.Errorf("circle detect fail")
		}
	}()
	process2.Merge("TestMergeLongCircle1")
}

func TestAddWaitBeforeAfterMergeWithCircle(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestAddWaitBeforeAfterMergeWithCircle1")
	process1 := factory1.Process("TestAddWaitBeforeAfterMergeWithCircle1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "2")
	process1.AliasStep(GenerateStep(4), "4", "3")
	process1.AliasStep(GenerateStep(5), "5", "4")
	factory2 := flow.RegisterFlow("TestAddWaitBeforeAfterMergeWithCircle2")
	process2 := factory2.Process("TestAddWaitBeforeAfterMergeWithCircle2")
	tmp := process2.AliasStep(GenerateStep(5), "5")
	process2.Merge("TestAddWaitBeforeAfterMergeWithCircle1")
	defer func() {
		if r := recover(); r != nil {
			t.Logf("circle detect success info: %v", r)
		} else {
			t.Errorf("circle detect fail")
		}
	}()
	tmp.Next(GenerateStep(2), "2")
	return
}

func TestAddWaitAllAfterMergeWithCircle(t *testing.T) {
	defer resetCurrent()
	factory1 := flow.RegisterFlow("TestAddWaitAllAfterMergeWithCircle1")
	process1 := factory1.Process("TestAddWaitAllAfterMergeWithCircle1")
	process1.AliasStep(GenerateStep(1), "1")
	process1.AliasStep(GenerateStep(2), "2", "1")
	process1.AliasStep(GenerateStep(3), "3", "2")
	process1.AliasStep(GenerateStep(4), "4", "3")
	process1.AliasStep(GenerateStep(5), "5", "4")
	factory2 := flow.RegisterFlow("TestAddWaitAllAfterMergeWithCircle2")
	process2 := factory2.Process("TestAddWaitAllAfterMergeWithCircle2")
	process2.AliasStep(GenerateStep(5), "5")
	process2.Merge("TestAddWaitAllAfterMergeWithCircle1")
	defer func() {
		if r := recover(); r != nil {
			t.Logf("circle detect success info: %v", r)
		} else {
			t.Errorf("circle detect fail")
		}
	}()
	process2.Tail(GenerateStep(2), "2")
	return
}
