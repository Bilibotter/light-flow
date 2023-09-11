package test

import (
	"gitee.com/MetaphysicCoding/light-flow/core"
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
	if !slices.Contains(core.ExplainStatus(core.Cancel), "Cancel") {
		t.Errorf("cancel status explain error")
	}
	if !slices.Contains(core.ExplainStatus(core.Pause), "Pause") {
		t.Errorf("pause status explain error")
	}
	if !slices.Contains(core.ExplainStatus(core.Running), "Running") {
		t.Errorf("running status explain error")
	}
	if !slices.Contains(core.ExplainStatus(core.Success), "Success") {
		t.Errorf("success status explain error")
	}
	if !slices.Contains(core.ExplainStatus(core.Failed), "Failed") {
		t.Errorf("failed status explain error")
	}
	if !slices.Contains(core.ExplainStatus(core.Timeout), "Timeout") {
		t.Errorf("timeout status explain error")
	}
	if !slices.Contains(core.ExplainStatus(core.Panic), "Panic") {
		t.Errorf("panic status explain error")
	}
	if !slices.Contains(core.ExplainStatus(core.Pending), "Pending") {
		t.Errorf("pending status explain error")
	}
}

func TestExplainStatus2(t *testing.T) {
	status := int64(0)
	core.AppendStatus(&status, core.Cancel)
	core.AppendStatus(&status, core.Panic)
	core.AppendStatus(&status, core.Failed)
	core.AppendStatus(&status, core.Timeout)
	core.AppendStatus(&status, core.Success)
	if slices.Contains(core.ExplainStatus(status), "Success") {
		t.Errorf("explain success while error occur")
	}
	if !slices.Contains(core.ExplainStatus(status), "Failed") {
		t.Errorf("failed status explain error")
	}
	if !slices.Contains(core.ExplainStatus(status), "Timeout") {
		t.Errorf("timeout status explain error")
	}
	if !slices.Contains(core.ExplainStatus(status), "Panic") {
		t.Errorf("panic status explain error")
	}
	if !slices.Contains(core.ExplainStatus(status), "Cancel") {
		t.Errorf("cancel status explain error")
	}
	status = 0
	core.AppendStatus(&status, core.Pause)
	core.AppendStatus(&status, core.Running)
	core.AppendStatus(&status, core.Pending)
	if !core.IsStatusNormal(status) {
		t.Errorf("normal status judge error")
	}
	if !slices.Contains(core.ExplainStatus(status), "Pause") {
		t.Errorf("pause status explain error")
	}
	if !slices.Contains(core.ExplainStatus(status), "Running") {
		t.Errorf("running status explain error")
	}
	if !slices.Contains(core.ExplainStatus(status), "Pending") {
		t.Errorf("pending status explain error")
	}
	core.AppendStatus(&status, core.Success)
	if !slices.Contains(core.ExplainStatus(status), "Success") {
		t.Errorf("success status explain error")
	}
}

func TestIsStatusNormal(t *testing.T) {
	normal := []int64{core.Pending, core.Running, core.Pause, core.Success}
	abnormal := []int64{core.Cancel, core.Timeout, core.Panic, core.Failed}
	for _, status := range normal {
		if !core.IsStatusNormal(status) {
			t.Errorf("normal status %d judge to abnormal", status)
		}
	}
	for _, status := range abnormal {
		if core.IsStatusNormal(status) {
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
	dst := core.CreateStruct[Dst](&src)
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
	core.CopyProperties(&src, &dst)
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
