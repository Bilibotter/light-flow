package test

import (
	"gitee.com/MetaphysicCoding/light-flow"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

type Internal struct {
	Name int
}

type Src struct {
	Name     string
	Id       int
	Start    time.Time
	Internal Internal
	SameName string
	Exist    bool
}

type Dst struct {
	Name     string
	Id       int
	Start    time.Time
	Internal Internal
	SameName int
	UnExist  bool
}

func TestGetFuncName(t *testing.T) {
	if light_flow.GetFuncName(TestGetFuncName) != "TestGetFuncName" {
		t.Errorf("get TestGetFuncName name error")
	}
	if light_flow.GetFuncName(light_flow.GetStructName) != "GetStructName" {
		t.Errorf("get GetStructName name error")
	}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("test used get func name to anonymous function, panic: %v", r)
		}
	}()
	if strings.HasPrefix(light_flow.GetFuncName(func() {}), "func") {
		t.Errorf("get anonymous function name error")
	}
}

func TestGetStructName(t *testing.T) {
	d := Dst{}
	dP := &Dst{}
	step := light_flow.FlowStep{}
	stepP := &light_flow.FlowStep{}
	if light_flow.GetStructName(d) != "Dst" {
		t.Errorf("get Dst struct name error")
	}
	if light_flow.GetStructName(dP) != "*Dst" {
		t.Errorf("get Dst pointer struct name error")
	}
	if light_flow.GetStructName(step) != "FlowStep" {
		t.Errorf("get FlowStep struct name error")
	}
	if light_flow.GetStructName(stepP) != "*FlowStep" {
		t.Errorf("get FlowStep pointer struct name error")
	}
}

func TestSet(t *testing.T) {
	s := light_flow.NewRoutineUnsafeSet()
	for i := 0; i < 1000; i++ {
		s.Add(strconv.Itoa(i))
	}
	for i := 0; i < 1000; i++ {
		if !s.Contains(strconv.Itoa(i)) {
			t.Errorf("set not contains %d", i)
			break
		}
		if s.Contains(strconv.Itoa(i + 1000)) {
			t.Errorf("set contains %d", i+1000)
			break
		}
	}
}

func TestPopStatus(t *testing.T) {
	status := int64(0)
	light_flow.AppendStatus(&status, light_flow.Cancel)
	light_flow.AppendStatus(&status, light_flow.Panic)
	light_flow.AppendStatus(&status, light_flow.Failed)
	light_flow.AppendStatus(&status, light_flow.Error)
	light_flow.AppendStatus(&status, light_flow.Timeout)
	light_flow.AppendStatus(&status, light_flow.Stop)

	if !slices.Contains(light_flow.ExplainStatus(status), "Cancel") {
		t.Errorf("cancel appended but not exist")
	}
	light_flow.PopStatus(&status, light_flow.Cancel)
	if slices.Contains(light_flow.ExplainStatus(status), "Cancel") {
		t.Errorf("cancel status pop error")
	}

	if !slices.Contains(light_flow.ExplainStatus(status), "Panic") {
		t.Errorf("panic appended but not exist")
	}
	light_flow.PopStatus(&status, light_flow.Panic)
	if slices.Contains(light_flow.ExplainStatus(status), "Panic") {
		t.Errorf("panic status pop error")
	}

	if !slices.Contains(light_flow.ExplainStatus(status), "Failed") {
		t.Errorf("failed appended but not exist")
	}
	light_flow.PopStatus(&status, light_flow.Failed)
	if slices.Contains(light_flow.ExplainStatus(status), "Failed") {
		t.Errorf("failed status pop error")
	}

	if !slices.Contains(light_flow.ExplainStatus(status), "Timeout") {
		t.Errorf("timeout appended bu not exist")
	}
	light_flow.PopStatus(&status, light_flow.Timeout)
	if slices.Contains(light_flow.ExplainStatus(status), "Timeout") {
		t.Errorf("timeout status pop error")
	}

	if !slices.Contains(light_flow.ExplainStatus(status), "Stop") {
		t.Errorf("error appended bu not exist")
	}
	light_flow.PopStatus(&status, light_flow.Stop)
	if slices.Contains(light_flow.ExplainStatus(status), "Stop") {
		t.Errorf("error status pop error")
	}

	if !slices.Contains(light_flow.ExplainStatus(status), "Error") {
		t.Errorf("error appended bu not exist")
	}
	light_flow.PopStatus(&status, light_flow.Error)
	if slices.Contains(light_flow.ExplainStatus(status), "Error") {
		t.Errorf("error status pop error")
	}
}

func TestExplainStatus1(t *testing.T) {
	if !slices.Contains(light_flow.ExplainStatus(light_flow.Cancel), "Cancel") {
		t.Errorf("cancel status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(light_flow.Pause), "Pause") {
		t.Errorf("pause status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(light_flow.Running), "Running") {
		t.Errorf("running status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(light_flow.Success), "Success") {
		t.Errorf("success status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(light_flow.Failed), "Failed") {
		t.Errorf("failed status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(light_flow.Timeout), "Timeout") {
		t.Errorf("timeout status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(light_flow.Panic), "Panic") {
		t.Errorf("panic status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(light_flow.Pending), "Pending") {
		t.Errorf("pending status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(light_flow.Error), "Error") {
		t.Errorf("error status explain error")
	}
}

func TestExplainStatus2(t *testing.T) {
	status := int64(0)
	light_flow.AppendStatus(&status, light_flow.Cancel)
	light_flow.AppendStatus(&status, light_flow.Panic)
	light_flow.AppendStatus(&status, light_flow.Failed)
	light_flow.AppendStatus(&status, light_flow.Timeout)
	light_flow.AppendStatus(&status, light_flow.Stop)
	light_flow.AppendStatus(&status, light_flow.Success)
	if slices.Contains(light_flow.ExplainStatus(status), "Success") {
		t.Errorf("explain success while error occur")
	}
	if !slices.Contains(light_flow.ExplainStatus(status), "Failed") {
		t.Errorf("failed status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(status), "Timeout") {
		t.Errorf("timeout status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(status), "Panic") {
		t.Errorf("panic status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(status), "Cancel") {
		t.Errorf("cancel status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(status), "Stop") {
		t.Errorf("stop status explain error")
	}

	status = 0
	light_flow.AppendStatus(&status, light_flow.Pause)
	light_flow.AppendStatus(&status, light_flow.Running)
	light_flow.AppendStatus(&status, light_flow.Pending)
	if !light_flow.IsStatusNormal(status) {
		t.Errorf("normal status judge error")
	}
	if !slices.Contains(light_flow.ExplainStatus(status), "Pause") {
		t.Errorf("pause status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(status), "Running") {
		t.Errorf("running status explain error")
	}
	if !slices.Contains(light_flow.ExplainStatus(status), "Pending") {
		t.Errorf("pending status explain error")
	}
	light_flow.AppendStatus(&status, light_flow.Success)
	if !slices.Contains(light_flow.ExplainStatus(status), "Success") {
		t.Errorf("success status explain error")
	}
}

func TestIsStatusNormal(t *testing.T) {
	normal := []int64{light_flow.Pending, light_flow.Running, light_flow.Pause, light_flow.Success}
	abnormal := []int64{light_flow.Cancel, light_flow.Timeout, light_flow.Panic, light_flow.Failed}
	for _, status := range normal {
		if !light_flow.IsStatusNormal(status) {
			t.Errorf("normal status %d judge to abnormal", status)
		}
	}
	for _, status := range abnormal {
		if light_flow.IsStatusNormal(status) {
			t.Errorf("abnormal status %d judge to normal", status)
		}
	}
}

func TestCreateStruct(t *testing.T) {
	src := Src{
		Internal: Internal{Name: 12},
		Name:     "src",
		Id:       12,
		Start:    time.Now(),
		SameName: "same",
		Exist:    true,
	}
	dst := light_flow.CreateStruct[Dst](&src)
	if dst.Name != src.Name {
		t.Errorf("dst.Name != src.Name")
	}
	if dst.Id != src.Id {
		t.Errorf("dst.Id != src.Id")
	}
	if dst.Start != src.Start {
		t.Errorf("dst.Start != src.Start")
	}
	if dst.Internal != src.Internal {
		t.Errorf("dst.Internal != src.Internal")
	}
	if dst.SameName != 0 || dst.UnExist != false {
		t.Errorf("dst.SameName != 0 || dst.UnExist != false")
	}
	return
}

func TestCopyProperties(t *testing.T) {
	src := Src{
		Internal: Internal{Name: 12},
		Name:     "src",
		Id:       12,
		Start:    time.Now(),
		SameName: "same",
		Exist:    true,
	}
	dst := Dst{}
	light_flow.CopyProperties(&src, &dst)
	if dst.Name != src.Name {
		t.Errorf("dst.Name != src.Name")
	}
	if dst.Id != src.Id {
		t.Errorf("dst.Id != src.Id")
	}
	if dst.Start != src.Start {
		t.Errorf("dst.Start != src.Start")
	}
	if dst.Internal != src.Internal {
		t.Errorf("dst.Internal != src.Internal")
	}
	if dst.SameName != 0 || dst.UnExist != false {
		t.Errorf("dst.SameName != 0 || dst.UnExist != false")
	}
	return
}
