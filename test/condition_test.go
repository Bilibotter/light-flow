package test

import (
	"github.com/Bilibotter/light-flow/flow"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func LTECondition(meta *flow.StepMeta, suffixes ...string) *flow.StepMeta {
	suffix := ""
	if len(suffixes) > 0 {
		suffix = suffixes[0]
	}
	t, err := time.Parse(time.RFC3339, "2024-09-05T15:04:05Z")
	if err != nil {
		panic(err.Error())
	}
	t1, err1 := time.Parse(time.RFC3339, "2024-09-04T15:04:05Z")
	if err1 != nil {
		panic(err1.Error())
	}
	meta.LTE(suffix+"int", 2).
		LTE(suffix+"int8", int8(3)).
		LTE(suffix+"int16", int16(4)).
		LTE(suffix+"int32", int32(5)).
		LTE(suffix+"int64", int64(6)).
		LTE(suffix+"uint", uint(7)).
		LTE(suffix+"uint8", uint8(8)).
		LTE(suffix+"uint16", uint16(9)).
		LTE(suffix+"uint32", uint32(10)).
		LTE(suffix+"uint64", uint64(11)).
		LTE(suffix+"float32", float32(12.1)).
		LTE(suffix+"float64", float64(13.2)).
		LTE(suffix+"*Comparable", &ComparableImpl{Age: 17}).
		LTE(suffix+"int", 1).
		LTE(suffix+"int8", int8(2)).
		LTE(suffix+"int16", int16(3)).
		LTE(suffix+"int32", int32(4)).
		LTE(suffix+"int64", int64(5)).
		LTE(suffix+"uint", uint(6)).
		LTE(suffix+"uint8", uint8(7)).
		LTE(suffix+"uint16", uint16(8)).
		LTE(suffix+"uint32", uint32(9)).
		LTE(suffix+"uint64", uint64(10)).
		LTE(suffix+"float32", float32(11.1)).
		LTE(suffix+"float64", float64(12.2)).
		LTE(suffix+"string", "hello").
		LTE(suffix+"bool", true).
		LTE(suffix+"time", t).
		LTE(suffix+"*time", &t).
		LTE(suffix+"time", t1).
		LTE(suffix+"*time", &t1).
		LTE(suffix+"Equality", EqualityImpl{Age: 13, Name: "Alice"}).
		LTE(suffix+"*Equality", &EqualityImpl{Age: 14, Name: "Bob"})
	return meta
}

func GTECondition(meta *flow.StepMeta, suffixes ...string) *flow.StepMeta {
	suffix := ""
	if len(suffixes) > 0 {
		suffix = suffixes[0]
	}
	t1, err1 := time.Parse(time.RFC3339, "2024-09-03T15:04:05Z")
	if err1 != nil {
		panic(err1.Error())
	}
	t2, err2 := time.Parse(time.RFC3339, "2024-09-04T15:04:05Z")
	if err2 != nil {
		panic(err2.Error())
	}
	meta.GTE(suffix+"int", 0).
		GTE(suffix+"int8", int8(1)).
		GTE(suffix+"int16", int16(2)).
		GTE(suffix+"int32", int32(3)).
		GTE(suffix+"int64", int64(4)).
		GTE(suffix+"uint", uint(5)).
		GTE(suffix+"uint8", uint8(6)).
		GTE(suffix+"uint16", uint16(7)).
		GTE(suffix+"uint32", uint32(8)).
		GTE(suffix+"uint64", uint64(9)).
		GTE(suffix+"float32", float32(10.1)).
		GTE(suffix+"float64", float64(11.2)).
		GTE(suffix+"time", t1).
		GTE(suffix+"*time", &t1).
		GTE(suffix+"*Comparable", &ComparableImpl{Age: 15}).
		GTE(suffix+"int", 1).
		GTE(suffix+"int8", int8(2)).
		GTE(suffix+"int16", int16(3)).
		GTE(suffix+"int32", int32(4)).
		GTE(suffix+"int64", int64(5)).
		GTE(suffix+"uint", uint(6)).
		GTE(suffix+"uint8", uint8(7)).
		GTE(suffix+"uint16", uint16(8)).
		GTE(suffix+"uint32", uint32(9)).
		GTE(suffix+"uint64", uint64(10)).
		GTE(suffix+"float32", float32(11.1)).
		GTE(suffix+"float64", float64(12.2)).
		GTE(suffix+"string", "hello").
		GTE(suffix+"bool", true).
		GTE(suffix+"time", t2).
		GTE(suffix+"*time", &t2).
		GTE(suffix+"Equality", EqualityImpl{Age: 13, Name: "Alice"}).
		GTE(suffix+"*Equality", &EqualityImpl{Age: 14, Name: "Bob"})
	return meta
}

func LTCondition(meta *flow.StepMeta, suffixes ...string) *flow.StepMeta {
	suffix := ""
	if len(suffixes) > 0 {
		suffix = suffixes[0]
	}
	t, err := time.Parse(time.RFC3339, "2024-09-05T15:04:05Z")
	if err != nil {
		panic(err.Error())
	}
	meta.LT(suffix+"int", 2).
		LT(suffix+"int8", int8(3)).
		LT(suffix+"int16", int16(4)).
		LT(suffix+"int32", int32(5)).
		LT(suffix+"int64", int64(6)).
		LT(suffix+"uint", uint(7)).
		LT(suffix+"uint8", uint8(8)).
		LT(suffix+"uint16", uint16(9)).
		LT(suffix+"uint32", uint32(10)).
		LT(suffix+"uint64", uint64(11)).
		LT(suffix+"float32", float32(12.1)).
		LT(suffix+"float64", float64(13.2)).
		LT(suffix+"time", t).
		LT(suffix+"*time", &t).
		LT(suffix+"*Comparable", &ComparableImpl{Age: 17})
	return meta
}

func GTCondition(meta *flow.StepMeta, suffixes ...string) *flow.StepMeta {
	suffix := ""
	if len(suffixes) > 0 {
		suffix = suffixes[0]
	}
	t, err := time.Parse(time.RFC3339, "2024-09-03T15:04:05Z")
	if err != nil {
		panic(err.Error())
	}
	meta.GT(suffix+"int", 0).
		GT(suffix+"int8", int8(1)).
		GT(suffix+"int16", int16(2)).
		GT(suffix+"int32", int32(3)).
		GT(suffix+"int64", int64(4)).
		GT(suffix+"uint", uint(5)).
		GT(suffix+"uint8", uint8(6)).
		GT(suffix+"uint16", uint16(7)).
		GT(suffix+"uint32", uint32(8)).
		GT(suffix+"uint64", uint64(9)).
		GT(suffix+"float32", float32(10.1)).
		GT(suffix+"float64", float64(11.2)).
		GT(suffix+"time", t).
		GT(suffix+"*time", &t).
		GT(suffix+"*Comparable", &ComparableImpl{Age: 15})
	return meta
}

func EQCondition(meta *flow.StepMeta, suffixes ...string) *flow.StepMeta {
	suffix := ""
	if len(suffixes) > 0 {
		suffix = suffixes[0]
	}
	t, err := time.Parse(time.RFC3339, "2024-09-04T15:04:05Z")
	if err != nil {
		panic(err.Error())
	}
	meta.EQ(suffix+"int", 1).
		EQ(suffix+"int8", int8(2)).
		EQ(suffix+"int16", int16(3)).
		EQ(suffix+"int32", int32(4)).
		EQ(suffix+"int64", int64(5)).
		EQ(suffix+"uint", uint(6)).
		EQ(suffix+"uint8", uint8(7)).
		EQ(suffix+"uint16", uint16(8)).
		EQ(suffix+"uint32", uint32(9)).
		EQ(suffix+"uint64", uint64(10)).
		EQ(suffix+"float32", float32(11.1)).
		EQ(suffix+"float64", float64(12.2)).
		EQ(suffix+"string", "hello").
		EQ(suffix+"bool", true).
		EQ(suffix+"time", t).
		EQ(suffix+"*time", &t).
		EQ(suffix+"Equality", EqualityImpl{Age: 13, Name: "Alice"}).
		EQ(suffix+"*Equality", &EqualityImpl{Age: 14, Name: "Bob"})
	return meta
}

func NEQCondition(meta *flow.StepMeta, suffixes ...string) *flow.StepMeta {
	suffix := ""
	if len(suffixes) > 0 {
		suffix = suffixes[0]
	}
	t, err := time.Parse(time.RFC3339, "2024-09-04T15:04:05Z")
	if err != nil {
		panic(err.Error())
	}
	t = t.Add(time.Minute)
	meta.NEQ(suffix+"int", 1+1).
		NEQ(suffix+"int8", int8(2+1)).
		NEQ(suffix+"int16", int16(3+1)).
		NEQ(suffix+"int32", int32(4+1)).
		NEQ(suffix+"int64", int64(5+1)).
		NEQ(suffix+"uint", uint(6+1)).
		NEQ(suffix+"uint8", uint8(7+1)).
		NEQ(suffix+"uint16", uint16(8+1)).
		NEQ(suffix+"uint32", uint32(9+1)).
		NEQ(suffix+"uint64", uint64(10+1)).
		NEQ(suffix+"float32", float32(11.1+1)).
		NEQ(suffix+"float64", float64(12.2+1)).
		NEQ(suffix+"string", "hello+1").
		NEQ(suffix+"bool", false).
		NEQ(suffix+"time", t).
		NEQ(suffix+"*time", &t).
		NEQ(suffix+"Equality", EqualityImpl{Age: 14, Name: "Alice"}).
		NEQ(suffix+"*Equality", &EqualityImpl{Age: 15, Name: "Bob"})
	return meta
}

func TestUnComparablePanic(t *testing.T) {
	wf := flow.RegisterFlow("TestUnComparablePanic")
	proc := wf.Process("TestUnComparablePanic")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected a panic")
		} else if _, ok := r.(runtime.Error); !ok {
			t.Error("Expected a runtime error")
		} else {
			t.Logf("%v", r)
		}
	}()
	step.EQ("notExist", []byte("123"))
}

func TestEQ(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestEQ0")
	proc := wf.Process("TestEQ0")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	EQCondition(step)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestEQ0", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestEQ1")
	proc = wf.Process("TestEQ1")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.EQ("notExistKey", "foo")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestEQ1", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestEQ2")
	proc = wf.Process("TestEQ2")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.EQ("notExistKey", "foo").SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestEQ2", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestEQ3")
	proc = wf.Process("TestEQ3")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	EQCondition(step).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestEQ3", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestEQ4")
	proc = wf.Process("TestEQ4")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.EQ("int", 2)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestEQ4", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestEQ5")
	proc = wf.Process("TestEQ5")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.EQ("notExistKey", "foo").SkipWithDependents("4")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestEQ5", nil)
}

func TestNEQ0(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestNEQ0")
	proc := wf.Process("TestNEQ0")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	NEQCondition(step)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestNEQ0", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestNEQ1")
	proc = wf.Process("TestNEQ1")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.NEQ("notExistKey", "foo")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestNEQ1", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestNEQ2")
	proc = wf.Process("TestNEQ2")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.NEQ("int", 1).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestNEQ2", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestNEQ3")
	proc = wf.Process("TestNEQ3")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	NEQCondition(step).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestNEQ3", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestNEQ4")
	proc = wf.Process("TestNEQ4")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.NEQ("int", 1)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestNEQ4", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestNEQ5")
	proc = wf.Process("TestNEQ5")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.NEQ("int", 1).SkipWithDependents("4")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestNEQ5", nil)
}

func TestLT(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestLT0")
	proc := wf.Process("TestLT0")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	LTCondition(step)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestLT0", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestLT1")
	proc = wf.Process("TestLT1")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.LT("notExistKey", "foo")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestLT1", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestLT2")
	proc = wf.Process("TestLT2")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.LT("int", 0).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestLT2", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestLT3")
	proc = wf.Process("TestLT3")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	LTCondition(step).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestLT3", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestLT4")
	proc = wf.Process("TestLT4")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.LT("int", 1)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestLT4", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestLT5")
	proc = wf.Process("TestLT5")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.LT("int", 1).SkipWithDependents("4")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestLT5", nil)
}

func TestLTE(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestLTE0")
	proc := wf.Process("TestLTE0")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	LTECondition(step)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestLTE0", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestLTE1")
	proc = wf.Process("TestLTE1")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.LTE("notExistKey", "foo")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestLTE1", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestLTE2")
	proc = wf.Process("TestLTE2")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.LTE("int", 0).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestLTE2", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestLTE3")
	proc = wf.Process("TestLTE3")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	LTECondition(step).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestLTE3", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestLTE4")
	proc = wf.Process("TestLTE4")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.LTE("int", -1)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestLTE4", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestLTE5")
	proc = wf.Process("TestLTE5")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.LTE("int", -1).SkipWithDependents("4")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestLTE5", nil)
}

func TestGT(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestGT0")
	proc := wf.Process("TestGT0")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	GTCondition(step)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestGT0", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestGT1")
	proc = wf.Process("TestGT1")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.GT("notExistKey", "foo")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestGT1", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestGT2")
	proc = wf.Process("TestGT2")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.GT("int", 3).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestGT2", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestGT3")
	proc = wf.Process("TestGT3")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	GTCondition(step).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestGT3", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestGT4")
	proc = wf.Process("TestGT4")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.GT("int", 3)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestGT4", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestGT5")
	proc = wf.Process("TestGT5")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.GT("int", 3).SkipWithDependents("4")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestGT5", nil)
}

func TestGTE(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestGTE0")
	proc := wf.Process("TestGTE0")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	GTECondition(step)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestGTE0", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestGTE1")
	proc = wf.Process("TestGTE1")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.GTE("notExistKey", "foo")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestGTE1", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestGTE2")
	proc = wf.Process("TestGTE2")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.GTE("int", 3).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestGTE2", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestGTE3")
	proc = wf.Process("TestGTE3")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	GTECondition(step).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestGTE3", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestGTE4")
	proc = wf.Process("TestGTE4")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.GTE("int", 3)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestGTE4", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestGTE5")
	proc = wf.Process("TestGTE5")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.GTE("int", 3).SkipWithDependents("4")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestGTE5", nil)
}

func TestCondition(t *testing.T) {
	meet := func(step flow.Step) bool {
		return step.Name() == "2"
	}
	panicNotMeet := func(step flow.Step) bool {
		panic("not meet")
	}
	notMeet := func(step flow.Step) bool {
		return step.Name() != "2"
	}
	defer resetCurrent()
	wf := flow.RegisterFlow("TestCondition0")
	proc := wf.Process("TestCondition0")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(meet)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestCondition0", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestCondition1")
	proc = wf.Process("TestCondition1")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(panicNotMeet)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestCondition1", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestCondition2")
	proc = wf.Process("TestCondition2")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(notMeet).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestCondition2", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestCondition3")
	proc = wf.Process("TestCondition3")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(meet).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestCondition3", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestCondition4")
	proc = wf.Process("TestCondition4")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(notMeet)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestCondition4", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestCondition5")
	proc = wf.Process("TestCondition5")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(notMeet).SkipWithDependents("4")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestCondition5", nil)
}

func TestOR(t *testing.T) {
	meet := func(step flow.Step) bool {
		return step.Name() == "2"
	}
	notMeet := func(step flow.Step) bool {
		return step.Name() != "2"
	}
	defer resetCurrent()
	wf := flow.RegisterFlow("TestOR0")
	proc := wf.Process("TestOR0")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(meet).OR().Condition(meet)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestOR0", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestOR1")
	proc = wf.Process("TestOR1")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(notMeet).OR().Condition(notMeet)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestOR1", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestOR2")
	proc = wf.Process("TestOR2")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(meet).OR().Condition(notMeet)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestOR2", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestOR3")
	proc = wf.Process("TestOR3")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(notMeet).OR().Condition(meet)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestOR3", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestOR5")
	proc = wf.Process("TestOR5")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(notMeet).OR().Condition(meet).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestOR5", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestOR6")
	proc = wf.Process("TestOR6")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(notMeet).OR().Condition(notMeet).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestOR6", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestOR7")
	proc = wf.Process("TestOR7")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(notMeet).OR().Condition(notMeet).SkipWithDependents("4")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestOR7", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestOR8")
	proc = wf.Process("TestOR8")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(notMeet).OR().Condition(notMeet).OR().Condition(notMeet).OR().Condition(notMeet).OR().Condition(meet)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestOR8", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestOR9")
	proc = wf.Process("TestOR9")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.Condition(meet).OR()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestOR9", nil)
}

func TestIllegalOR(t *testing.T) {
	f1 := func() {
		t.Logf("TestIllegalOR0")
		defer func() {
			if r := recover(); r != nil {
				if !strings.Contains(r.(string), "condition failed") {
					t.Errorf("illegal OR should panic with condition failed, but got %v", r)
				}
			} else {
				t.Errorf("illegal OR should panic with condition failed, but success")
			}
		}()
		wf := flow.RegisterFlow("TestIllegalOR0")
		proc := wf.Process("TestIllegalOR0")
		step := proc.NameStep(Fx[flow.Step](t).Step(), "1")
		step.OR()
	}
	f1()
	f2 := func() {
		t.Logf("TestIllegalOR1")
		defer func() {
			if r := recover(); r != nil {
				if !strings.Contains(r.(string), "condition failed") {
					t.Errorf("illegal OR should panic with condition failed, but got %v", r)
				}
			} else {
				t.Errorf("illegal OR should panic with condition failed, but success")
			}
		}()
		wf := flow.RegisterFlow("TestIllegalOR1")
		proc := wf.Process("TestIllegalOR1")
		step := proc.NameStep(Fx[flow.Step](t).Step(), "1")
		EQCondition(step).OR().OR()
	}
	f2()
}

func TestComparePanic(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	wf := flow.RegisterFlow("TestComparePanic")
	proc := wf.Process("TestComparePanic")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	EQCondition(step).EQ("PanicComparable", &PanicComparableImpl{Content: "panic"})
	wf.AfterFlow(true, CheckResult(t, 11, flow.Success))
	atomic.StoreInt64(&letGo, 1)
	flow.DoneFlow("TestComparePanic", nil)

	resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	wf = flow.RegisterFlow("TestComparePanic2")
	proc = wf.Process("TestComparePanic2")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	NEQCondition(step).NEQ("PanicComparable", PanicComparableImpl{Content: "panic"})
	wf.AfterFlow(true, CheckResult(t, 11, flow.Success))
	atomic.StoreInt64(&letGo, 1)
	flow.DoneFlow("TestComparePanic2", nil)

	resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	wf = flow.RegisterFlow("TestComparePanic3")
	proc = wf.Process("TestComparePanic3")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	LTCondition(step).LT("PanicComparable", PanicComparableImpl{Content: "panic"})
	wf.AfterFlow(true, CheckResult(t, 11, flow.Success))
	atomic.StoreInt64(&letGo, 1)
	flow.DoneFlow("TestComparePanic3", nil)

	resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	wf = flow.RegisterFlow("TestComparePanic4")
	proc = wf.Process("TestComparePanic4")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	LTECondition(step).LTE("PanicComparable", PanicComparableImpl{Content: "panic"})
	wf.AfterFlow(true, CheckResult(t, 11, flow.Success))
	atomic.StoreInt64(&letGo, 1)
	flow.DoneFlow("TestComparePanic4", nil)

	resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	wf = flow.RegisterFlow("TestComparePanic5")
	proc = wf.Process("TestComparePanic5")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	GTCondition(step).GT("PanicComparable", PanicComparableImpl{Content: "panic"})
	wf.AfterFlow(true, CheckResult(t, 11, flow.Success))
	atomic.StoreInt64(&letGo, 1)
	flow.DoneFlow("TestComparePanic5", nil)

	resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	wf = flow.RegisterFlow("TestComparePanic6")
	proc = wf.Process("TestComparePanic6")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	GTECondition(step).GTE("PanicComparable", PanicComparableImpl{Content: "panic"})
	wf.AfterFlow(true, CheckResult(t, 11, flow.Success))
	atomic.StoreInt64(&letGo, 1)
	flow.DoneFlow("TestComparePanic6", nil)
}

func TestConditionRecover(t *testing.T) {
	flow.RegisterType[EqualityImpl]()
	flow.RegisterType[ComparableImpl]()
	flow.RegisterType[PanicComparableImpl]()
	defer resetCurrent()
	wf := flow.RegisterFlow("TestConditionRecover0")
	wf.EnableRecover()
	proc := wf.Process("TestConditionRecover0")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Recover().Step(), "2", "1")
	EQCondition(step)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success)).When(flow.Recovering)
	ff := flow.DoneFlow("TestConditionRecover0", nil)
	CheckResult(t, 3, flow.Error)(any(ff).(flow.WorkFlow))
	Recover("TestConditionRecover0")

	resetCurrent()
	wf = flow.RegisterFlow("TestConditionRecover1")
	wf.EnableRecover()
	proc = wf.Process("TestConditionRecover1")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	EQCondition(step)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Recover().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 1, flow.Success)).When(flow.Recovering)
	ff = flow.DoneFlow("TestConditionRecover1", nil)
	CheckResult(t, 4, flow.Error)(any(ff).(flow.WorkFlow))
	Recover("TestConditionRecover1")

	resetCurrent()
	wf = flow.RegisterFlow("TestConditionRecover2")
	wf.EnableRecover()
	proc = wf.Process("TestConditionRecover2")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Recover().Step(), "2", "1")
	EQCondition(step).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2", "3")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success)).When(flow.Recovering)
	ff = flow.DoneFlow("TestConditionRecover2", nil)
	CheckResult(t, 3, flow.Error)(any(ff).(flow.WorkFlow))
	Recover("TestConditionRecover2")
}

func TestConditionMerge(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestConditionMerge0")
	proc := wf.Process("MeetCondition")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	EQCondition(step)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")

	proc = wf.Process("NotMeetCondition")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.EQ("notExistKey", 1)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")

	wf = flow.RegisterFlow("TestConditionMerge1")
	proc = wf.Process("TestConditionMerge10")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	NEQCondition(step)
	proc.Merge("MeetCondition")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestConditionMerge1", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestConditionMerge2")
	proc = wf.Process("TestConditionMerge20")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	NEQCondition(step)
	proc.Merge("NotMeetCondition")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestConditionMerge2", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestConditionMerge3")
	proc = wf.Process("TestConditionMerge30")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.EQ("notExistKey", 1)
	proc.Merge("MeetCondition")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestConditionMerge3", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestConditionMerge4")
	proc = wf.Process("TestConditionMerge40")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.EQ("notExistKey", 1)
	proc.Merge("NotMeetCondition")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestConditionMerge4", nil)
}

func TestConditionMergeWithSkip(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestConditionMergeWithSkip0")
	proc := wf.Process("SkipDepends")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2")
	step.EQ("notExistKey", 1).SkipWithDependents()
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "2")

	proc = wf.Process("NotSkipDepends")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2")
	step.EQ("notExistKey", 1)
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "2")

	proc = wf.Process("SkipDependsWithExclude")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2")
	step.EQ("notExistKey", 1).SkipWithDependents("4")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "2")

	wf = flow.RegisterFlow("TestConditionMergeWithSkip1")
	proc = wf.Process("TestConditionMergeWithSkip10")
	proc.Merge("SkipDepends")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "5", "1")
	wf.AfterFlow(true, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestConditionMergeWithSkip1", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestConditionMergeWithSkip2")
	proc = wf.Process("TestConditionMergeWithSkip20")
	proc.Merge("NotSkipDepends")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "5", "1")
	wf.AfterFlow(true, CheckResult(t, 4, flow.Success))
	flow.DoneFlow("TestConditionMergeWithSkip2", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestConditionMergeWithSkip3")
	proc = wf.Process("TestConditionMergeWithSkip30")
	proc.Merge("SkipDependsWithExclude")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "5", "1")
	wf.AfterFlow(true, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestConditionMergeWithSkip3", nil)
}

func TestConditionMergeWithAdd(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestConditionMergeWithAdd0")
	proc := wf.Process("TestConditionMergeWithAdd0")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	EQCondition(step)
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	step.EQ("notExistKey", 1)
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "1")
	step.EQ("notExistKey", 1).SkipWithDependents()
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "5", "1")
	step.EQ("notExistKey", 1).SkipWithDependents("7")
	proc.NameStep(Fx[flow.Step](t).Inc(100).Step(), "6", "4")
	proc.NameStep(Fx[flow.Step](t).Inc(10).Step(), "7", "5")
	proc.NameStep(Fx[flow.Step](t).Inc(10).Step(), "8", "2")
	wf.AfterFlow(true, CheckResult(t, 22, flow.Success))
	flow.DoneFlow("TestConditionMergeWithAdd0", nil)

	resetCurrent()
	wf = flow.RegisterFlow("TestConditionMergeWithAdd1")
	proc = wf.Process("TestConditionMergeWithAdd1")
	proc.NameStep(Fx[flow.Step](t).Inc(1000).Step(), "9")
	proc.Merge("TestConditionMergeWithAdd0")
	wf.AfterFlow(true, CheckResult(t, 1022, flow.Success))
	flow.DoneFlow("TestConditionMergeWithAdd1", nil)
}

func TestConditionMultiMerge(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestConditionMultiMerge0")
	proc := wf.Process("TestConditionMultiMerge0")
	proc.NameStep(Fx[flow.Step](t).Cond().Step(), "1")
	step := proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.SkipWithDependents("3")
	EQCondition(step)

	wf = flow.RegisterFlow("TestConditionMultiMerge1")
	proc = wf.Process("TestConditionMultiMerge1")
	proc.Merge("TestConditionMultiMerge0")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	NEQCondition(step)

	wf = flow.RegisterFlow("TestConditionMultiMerge2")
	proc = wf.Process("TestConditionMultiMerge2")
	proc.Merge("TestConditionMultiMerge1")
	step = proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	step.EQ("notExistKey", 1)
	proc.NameStep(Fx[flow.Step](t).Inc(10).Step(), "3", "2")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "2")
	wf.AfterFlow(true, CheckResult(t, 11, flow.Success))
	flow.DoneFlow("TestConditionMultiMerge2", nil)
}
