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
