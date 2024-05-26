package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type EqualityImpl struct {
	age  int
	name string
}

func (ei EqualityImpl) Equal(other any) bool {
	e, ok := other.(EqualityImpl)
	if !ok {
		return false
	}
	return ei.age == e.age && ei.name == e.name
}

func CheckResult(t *testing.T, check int64, statuses ...*flow.StatusEnum) func(*flow.WorkFlow) (keepOn bool, err error) {
	return func(workFlow *flow.WorkFlow) (keepOn bool, err error) {
		t.Logf("start check result")
		if current != check {
			t.Errorf("execute %d step, but current = %d\n", check, current)
		}
		for _, status := range statuses {
			if !workFlow.Has(status) {
				t.Errorf("workFlow has not %s status\n", status.Message())
			}
		}
		t.Logf("status expalin=%s", strings.Join(workFlow.ExplainStatus(), ","))
		return true, nil
	}
}

func GenerateGetResult(step string, check any) func(ctx flow.StepCtx) (any, error) {
	return func(ctx flow.StepCtx) (any, error) {
		wrap, ok := ctx.GetResult(step)
		if !ok {
			panic(fmt.Sprintf("result[%s] not found", step))
		}
		if !reflect.DeepEqual(wrap, check) {
			panic(fmt.Sprintf("result[%s] = %v, want %v", step, wrap, check))
		}
		atomic.AddInt64(&current, 1)
		return check, nil
	}
}

func GenerateConditionStep(value ...any) func(ctx flow.StepCtx) (any, error) {
	return func(ctx flow.StepCtx) (any, error) {
		for _, v := range value {
			ctx.SetCond(v)
		}
		atomic.AddInt64(&current, 1)
		return "disruptions", nil
	}
}

func TestMultipleConditionGroup(t *testing.T) {
	defer resetCurrent()

	var v1, v2 interface{}
	v1 = 1
	v2 = 1.0
	// true true
	workflow := flow.RegisterFlow("TestMultipleConditionGroup1")
	process := workflow.Process("TestMultipleConditionGroup1")
	process.NameStep(GenerateConditionStep(v1, v2), "1")
	step2 := process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", v1)
	step2.NEQ("1", 2.0)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestMultipleConditionGroup1", map[string]any{"1": -1, "2": -1})
	resetCurrent()

	v1 = 1
	v2 = 1.0
	// true false
	workflow = flow.RegisterFlow("TestMultipleConditionGroup2")
	process = workflow.Process("TestMultipleConditionGroup2")
	process.NameStep(GenerateConditionStep(v1, v2), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", v1)
	step2.NEQ("1", 1.0)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestMultipleConditionGroup2", map[string]any{"1": -1, "2": -1})
	resetCurrent()

	v1 = 1
	v2 = 1.0
	// true false
	workflow = flow.RegisterFlow("TestMultipleConditionGroup3")
	process = workflow.Process("TestMultipleConditionGroup3")
	process.NameStep(GenerateConditionStep(v1, v2), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", 2)
	step2.NEQ("1", 2.0)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestMultipleConditionGroup3", map[string]any{"1": -1, "2": -1})
	resetCurrent()

	v1 = 1
	v2 = 1.0
	// false false
	workflow = flow.RegisterFlow("TestMultipleConditionGroup4")
	process = workflow.Process("TestMultipleConditionGroup4")
	process.NameStep(GenerateConditionStep(v1, v2), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", 2)
	step2.NEQ("1", 1.0)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestMultipleConditionGroup4", map[string]any{"1": -1, "2": -1})
	resetCurrent()
}

func TestContinuousCondition(t *testing.T) {
	defer resetCurrent()

	var v1, v2 interface{}
	v1 = 1
	v2 = 1.0
	// true true
	workflow := flow.RegisterFlow("TestContinuousCondition1")
	process := workflow.Process("TestContinuousCondition1")
	process.NameStep(GenerateConditionStep(v1, v2), "1")
	step2 := process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", v1).NEQ(2.0)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestContinuousCondition1", map[string]any{"1": -1, "2": -1})
	resetCurrent()

	v1 = 1
	v2 = 1.0
	// true false
	workflow = flow.RegisterFlow("TestContinuousCondition2")
	process = workflow.Process("TestContinuousCondition2")
	process.NameStep(GenerateConditionStep(v1, v2), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", v1).NEQ(1.0)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestContinuousCondition2", map[string]any{"1": -1, "2": -1})
	resetCurrent()

	v1 = 1
	v2 = 1.0
	// true false
	workflow = flow.RegisterFlow("TestContinuousCondition3")
	process = workflow.Process("TestContinuousCondition3")
	process.NameStep(GenerateConditionStep(v1, v2), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", 2).NEQ(2.0)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestContinuousCondition3", map[string]any{"1": -1, "2": -1})
	resetCurrent()

	v1 = 1
	v2 = 1.0
	// false false
	workflow = flow.RegisterFlow("TestContinuousCondition4")
	process = workflow.Process("TestContinuousCondition4")
	process.NameStep(GenerateConditionStep(v1, v2), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", 2).NEQ(1.0)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestContinuousCondition4", map[string]any{"1": -1, "2": -1})
	resetCurrent()
}

func TestDependItself(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestDependItself")
	process := workflow.Process("TestDependItself")
	step := process.NameStep(GenerateGetResult("1", "disruptions"), "1")
	defer func() {
		if r := recover(); r != nil {
			t.Logf("success")
		} else {
			t.Errorf("step depend itself without panic")
		}
	}()
	step.EQ("1", "dependitself")
}

func TestGetResultAfterSetCondition(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestGetResultAfterSetCondition")
	process := workflow.Process("TestGetResultAfterSetCondition")
	process.NameStep(GenerateConditionStep("1", 1), "1")
	process.NameStep(GenerateGetResult("1", "disruptions"), "2", "1")
	workflow.AfterFlow(false, CheckResult(t, 2, flow.Success))
	flow.DoneFlow("TestGetResultAfterSetCondition", map[string]any{"1": "inputdisruptions"})
}

func TestReachCondition(t *testing.T) {
	defer resetCurrent()
	var check interface{}

	check = "reach"
	workflow := flow.RegisterFlow("TestReachCondition1")
	process := workflow.Process("TestReachCondition1")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 := process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachCondition1", map[string]any{"1": "disruptions", "2": "disruptions"})
	resetCurrent()

	check = 1
	workflow = flow.RegisterFlow("TestReachCondition2")
	process = workflow.Process("TestReachCondition2")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachCondition2", map[string]any{"1": -1, "2": -1})
	resetCurrent()

	check = true
	workflow = flow.RegisterFlow("TestReachCondition3")
	process = workflow.Process("TestReachCondition3")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachCondition3", map[string]any{"1": false, "2": false})
	resetCurrent()

	check = float32(0.001)
	workflow = flow.RegisterFlow("TestReachCondition4")
	process = workflow.Process("TestReachCondition4")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachCondition4", map[string]any{"1": -0.002, "2": -0.002})
	resetCurrent()

	timeLayout := "2006-01-02 15:04:05"
	timeStr1 := "2022-01-15 15:04:05"
	timeStr2 := "2022-01-15 15:04:07"
	check, err := time.Parse(timeLayout, timeStr1)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr1)
	}
	disruptions, err := time.Parse(timeLayout, timeStr2)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr2)
	}
	workflow = flow.RegisterFlow("TestReachCondition5")
	process = workflow.Process("TestReachCondition5")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachCondition5", map[string]any{"1": disruptions, "2": disruptions})
	resetCurrent()

	check = uint(1)
	workflow = flow.RegisterFlow("TestReachCondition6")
	process = workflow.Process("TestReachCondition6")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachCondition6", map[string]any{"1": uint(0), "2": uint(0)})
	resetCurrent()

	check = EqualityImpl{age: 18, name: "xxx"}
	notEqual := EqualityImpl{age: 19, name: "xxxx"}
	workflow = flow.RegisterFlow("TestReachCondition7")
	process = workflow.Process("TestReachCondition7")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachCondition7", map[string]any{"1": notEqual, "2": notEqual})
}

func TestNotReachCondition(t *testing.T) {
	defer resetCurrent()
	var check, nomatch interface{}
	check = "reach"
	nomatch = "noreach"
	workflow := flow.RegisterFlow("TestNotReachCondition1")
	process := workflow.Process("TestNotReachCondition1")
	process.NameStep(GenerateConditionStep(nomatch), "1")
	step2 := process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachCondition1", map[string]any{"1": "disruptions", "2": "disruptions"})
	resetCurrent()

	check = int(1)
	nomatch = int(0)
	workflow = flow.RegisterFlow("TestNotReachCondition2")
	process = workflow.Process("TestNotReachCondition2")
	process.NameStep(GenerateConditionStep(nomatch), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachCondition2", map[string]any{"1": int(0), "2": int(0)})
	resetCurrent()

	check = true
	nomatch = false
	workflow = flow.RegisterFlow("TestNotReachCondition3")
	process = workflow.Process("TestNotReachCondition3")
	process.NameStep(GenerateConditionStep(nomatch), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachCondition3", map[string]any{"1": nomatch, "2": nomatch})
	resetCurrent()

	check = float32(0.001)
	nomatch = float32(0.002)
	workflow = flow.RegisterFlow("TestNotReachCondition4")
	process = workflow.Process("TestNotReachCondition4")
	process.NameStep(GenerateConditionStep(nomatch), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachCondition4", map[string]any{"1": -0.002, "2": -0.002})
	resetCurrent()

	timeLayout := "2006-01-02 15:04:05"
	timeStr1 := "2022-01-15 15:04:05"
	timeStr2 := "2022-01-15 15:04:07"
	check, err := time.Parse(timeLayout, timeStr1)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr1)
	}
	disruptions, err := time.Parse(timeLayout, timeStr2)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr2)
	}
	workflow = flow.RegisterFlow("TestNotReachCondition5")
	process = workflow.Process("TestNotReachCondition5")
	process.NameStep(GenerateConditionStep(disruptions), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachCondition5", map[string]any{"1": disruptions, "2": disruptions})
	resetCurrent()

	check = uint(1)
	nomatch = uint(0)
	workflow = flow.RegisterFlow("TestNotReachCondition6")
	process = workflow.Process("TestNotReachCondition6")
	process.NameStep(GenerateConditionStep(nomatch), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachCondition6", map[string]any{"1": uint(0), "2": uint(0)})
	resetCurrent()

	check = EqualityImpl{age: 18, name: "xxx"}
	nomatch = EqualityImpl{age: 19, name: "xxxx"}
	workflow = flow.RegisterFlow("TestNotReachCondition7")
	process = workflow.Process("TestNotReachCondition7")
	process.NameStep(GenerateConditionStep(nomatch), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachCondition7", map[string]any{"1": EqualityImpl{age: 19, name: "xxxx"}, "2": EqualityImpl{age: 19, name: "xxxx"}})
}

func TestMissingCondition(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMissingCondition1")
	process := workflow.Process("TestMissingCondition1")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 := process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", "reach")
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestMissingCondition1", map[string]any{"1": "disruptions", "2": "disruptions"})
	resetCurrent()

	workflow = flow.RegisterFlow("TestMissingCondition2")
	process = workflow.Process("TestMissingCondition2")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", int(1))
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestMissingCondition2", map[string]any{"1": 1, "2": 1})
	resetCurrent()

	workflow = flow.RegisterFlow("TestMissingCondition3")
	process = workflow.Process("TestMissingCondition3")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", true)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestMissingCondition3", map[string]any{"1": true, "2": true})
	resetCurrent()

	workflow = flow.RegisterFlow("TestMissingCondition4")
	process = workflow.Process("TestMissingCondition4")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", uint(1))
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestMissingCondition4", map[string]any{"1": uint(1), "2": uint(1)})
	resetCurrent()

	workflow = flow.RegisterFlow("TestMissingCondition5")
	process = workflow.Process("TestMissingCondition5")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", EqualityImpl{age: 18, name: "xxx"})
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestMissingCondition5", map[string]any{"1": EqualityImpl{age: 18, name: "xxx"}, "2": EqualityImpl{age: 18, name: "xxx"}})
	resetCurrent()

	workflow = flow.RegisterFlow("TestMissingCondition6")
	process = workflow.Process("TestMissingCondition6")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", float32(18.0))
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestMissingCondition6", map[string]any{"1": float32(18.0), "2": float32(18.0)})
	resetCurrent()

	timeLayout := "2006-01-02 15:04:05"
	timeStr := "2022-01-15 15:04:05"
	times, err := time.Parse(timeLayout, timeStr)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr)
	}
	workflow = flow.RegisterFlow("TestMissingCondition7")
	process = workflow.Process("TestMissingCondition7")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", times)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestMissingCondition7", map[string]any{"1": times, "2": times})
	resetCurrent()
}

func TestMissingNeqCondition(t *testing.T) {
	defer resetCurrent()
	workflow := flow.RegisterFlow("TestMissingNeqCondition1")
	process := workflow.Process("TestMissingNeqCondition1")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 := process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", "reach")
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestMissingNeqCondition1", map[string]any{"1": "disruptions", "2": "disruptions"})
	resetCurrent()

	workflow = flow.RegisterFlow("TestMissingNeqCondition2")
	process = workflow.Process("TestMissingNeqCondition2")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", int(1))
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestMissingNeqCondition2", map[string]any{"1": 1, "2": 1})
	resetCurrent()

	workflow = flow.RegisterFlow("TestMissingNeqCondition3")
	process = workflow.Process("TestMissingNeqCondition3")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", true)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestMissingNeqCondition3", map[string]any{"1": true, "2": true})
	resetCurrent()

	workflow = flow.RegisterFlow("TestMissingNeqCondition4")
	process = workflow.Process("TestMissingNeqCondition4")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", uint(1))
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestMissingNeqCondition4", map[string]any{"1": uint(1), "2": uint(1)})
	resetCurrent()

	workflow = flow.RegisterFlow("TestMissingNeqCondition5")
	process = workflow.Process("TestMissingNeqCondition5")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", EqualityImpl{age: 18, name: "xxx"})
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestMissingNeqCondition5", map[string]any{"1": EqualityImpl{age: 18, name: "xxx"}, "2": EqualityImpl{age: 18, name: "xxx"}})
	resetCurrent()

	workflow = flow.RegisterFlow("TestMissingNeqCondition6")
	process = workflow.Process("TestMissingNeqCondition6")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", float32(18.0))
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestMissingNeqCondition6", map[string]any{"1": float32(18.0), "2": float32(18.0)})
	resetCurrent()

	timeLayout := "2006-01-02 15:04:05"
	timeStr := "2022-01-15 15:04:05"
	times, err := time.Parse(timeLayout, timeStr)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr)
	}
	workflow = flow.RegisterFlow("TestMissingNeqCondition7")
	process = workflow.Process("TestMissingNeqCondition7")
	process.NameStep(GenerateNoDelayStep(1), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", times)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestMissingNeqCondition7", map[string]any{"1": times, "2": times})
	resetCurrent()
}

func TestReachConditionAllType(t *testing.T) {
	defer resetCurrent()
	timeLayout := "2006-01-02 15:04:05"
	timeStr := "2022-01-15 15:04:05"

	v1 := "reach"
	v2 := EqualityImpl{age: 18, name: "xxx"}
	v3 := float32(18.0)
	v4 := true
	v5 := uint(1)
	v6 := int(1)
	v7, err := time.Parse(timeLayout, timeStr)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr)
	}
	workflow := flow.RegisterFlow("TestReachConditionAllType")
	process := workflow.Process("TestReachConditionAllType")
	process.NameStep(GenerateConditionStep(v1, v3, v5, v7, v2, v4, v6), "1")
	step2 := process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", v1, v2, v3, v4, v5, v6, v7)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachConditionAllType", nil)
}

func TestNotReachConditionAllType(t *testing.T) {
	defer resetCurrent()
	timeLayout := "2006-01-02 15:04:05"
	timeStr := "2022-01-15 15:04:05"
	v1 := "reach"
	v2 := EqualityImpl{age: 18, name: "xxx"}
	v3 := float32(18.0)
	v4 := true
	v5 := uint(1)
	v6 := int(1)
	v7, err := time.Parse(timeLayout, timeStr)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr)
	}
	vv1 := "notreach"
	vv2 := EqualityImpl{age: 18, name: "xxxx"}
	vv3 := float32(-18.0)
	vv4 := false
	vv5 := uint(2)
	vv6 := int(-1)
	timeStr = "2023-01-15 15:04:05"
	vv7, err := time.Parse(timeLayout, timeStr)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr)
	}
	workflow := flow.RegisterFlow("TestNotReachConditionAllType")
	process := workflow.Process("TestNotReachConditionAllType")
	process.NameStep(GenerateConditionStep(vv1, vv3, vv5, vv7, vv2, vv4, vv6), "1")
	step2 := process.NameStep(GenerateNoDelayStep(2), "2")
	step2.EQ("1", v1, v2, v3, v4, v5, v6, v7)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachConditionAllType", nil)
}

func TestReachNeqCondition(t *testing.T) {
	defer resetCurrent()
	var check interface{}

	check = "reach"
	workflow := flow.RegisterFlow("TestReachNeqCondition1")
	process := workflow.Process("TestReachNeqCondition1")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 := process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", "notreach")
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachNeqCondition1", map[string]any{"1": "disruptions", "2": "disruptions"})
	resetCurrent()

	check = 1
	workflow = flow.RegisterFlow("TestReachNeqCondition2")
	process = workflow.Process("TestReachNeqCondition2")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", -1)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachNeqCondition2", map[string]any{"1": -1, "2": -1})
	resetCurrent()

	check = true
	workflow = flow.RegisterFlow("TestReachNeqCondition3")
	process = workflow.Process("TestReachNeqCondition3")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", false)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachNeqCondition3", map[string]any{"1": false, "2": false})
	resetCurrent()

	check = float32(0.001)
	workflow = flow.RegisterFlow("TestReachNeqCondition4")
	process = workflow.Process("TestReachNeqCondition4")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", -0.001)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachNeqCondition4", map[string]any{"1": -0.002, "2": -0.002})
	resetCurrent()

	timeLayout := "2006-01-02 15:04:05"
	timeStr1 := "2022-01-15 15:04:05"
	timeStr2 := "2022-01-15 15:04:07"
	check, err := time.Parse(timeLayout, timeStr1)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr1)
	}
	disruptions, err := time.Parse(timeLayout, timeStr2)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr2)
	}
	workflow = flow.RegisterFlow("TestReachNeqCondition5")
	process = workflow.Process("TestReachNeqCondition5")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", disruptions)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachNeqCondition5", map[string]any{"1": disruptions, "2": disruptions})
	resetCurrent()

	check = uint(1)
	workflow = flow.RegisterFlow("TestReachNeqCondition6")
	process = workflow.Process("TestReachNeqCondition6")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", uint(2))
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachNeqCondition6", map[string]any{"1": uint(0), "2": uint(0)})
	resetCurrent()

	check = EqualityImpl{age: 18, name: "xxx"}
	notEqual := EqualityImpl{age: 19, name: "xxxx"}
	workflow = flow.RegisterFlow("TestReachNeqCondition7")
	process = workflow.Process("TestReachNeqCondition7")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", notEqual)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestReachNeqCondition7", map[string]any{"1": notEqual, "2": notEqual})
}

func TestNotReachNeqCondition(t *testing.T) {
	defer resetCurrent()
	var check interface{}

	check = "reach"
	workflow := flow.RegisterFlow("TestNotReachNeqCondition1")
	process := workflow.Process("TestNotReachNeqCondition1")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 := process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachNeqCondition1", map[string]any{"1": "disruptions", "2": "disruptions"})
	resetCurrent()

	check = 1
	workflow = flow.RegisterFlow("TestNotReachNeqCondition2")
	process = workflow.Process("TestNotReachNeqCondition2")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachNeqCondition2", map[string]any{"1": -1, "2": -1})
	resetCurrent()

	check = true
	workflow = flow.RegisterFlow("TestNotReachNeqCondition3")
	process = workflow.Process("TestNotReachNeqCondition3")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachNeqCondition3", map[string]any{"1": false, "2": false})
	resetCurrent()

	check = float32(0.001)
	workflow = flow.RegisterFlow("TestNotReachNeqCondition4")
	process = workflow.Process("TestNotReachNeqCondition4")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachNeqCondition4", map[string]any{"1": -0.002, "2": -0.002})
	resetCurrent()

	timeLayout := "2006-01-02 15:04:05"
	timeStr1 := "2022-01-15 15:04:05"
	timeStr2 := "2022-01-15 15:04:07"
	check, err := time.Parse(timeLayout, timeStr1)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr1)
	}
	disruptions, err := time.Parse(timeLayout, timeStr2)
	if err != nil {
		t.Errorf("parse time %s fail", timeStr2)
	}
	workflow = flow.RegisterFlow("TestNotReachNeqCondition5")
	process = workflow.Process("TestNotReachNeqCondition5")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachNeqCondition5", map[string]any{"1": disruptions, "2": disruptions})
	resetCurrent()

	check = uint(1)
	workflow = flow.RegisterFlow("TestNotReachNeqCondition6")
	process = workflow.Process("TestNotReachNeqCondition6")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachNeqCondition6", map[string]any{"1": uint(0), "2": uint(0)})
	resetCurrent()

	check = EqualityImpl{age: 18, name: "xxx"}
	notEqual := EqualityImpl{age: 19, name: "xxxx"}
	workflow = flow.RegisterFlow("TestNotReachNeqCondition7")
	process = workflow.Process("TestNotReachNeqCondition7")
	process.NameStep(GenerateConditionStep(check), "1")
	step2 = process.NameStep(GenerateNoDelayStep(2), "2")
	step2.NEQ("1", check)
	process.NameStep(GenerateNoDelayStep(3), "3", "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Success))
	flow.DoneFlow("TestNotReachNeqCondition7", map[string]any{"1": notEqual, "2": notEqual})
}
