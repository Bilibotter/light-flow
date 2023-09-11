package test

import (
	"gitee.com/MetaphysicCoding/light-flow"
	"slices"
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
}

func TestExplainStatus2(t *testing.T) {
	status := int64(0)
	light_flow.AppendStatus(&status, light_flow.Cancel)
	light_flow.AppendStatus(&status, light_flow.Panic)
	light_flow.AppendStatus(&status, light_flow.Failed)
	light_flow.AppendStatus(&status, light_flow.Timeout)
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
