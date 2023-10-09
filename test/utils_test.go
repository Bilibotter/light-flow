package test

import (
	light_flow "gitee.com/MetaphysicCoding/light-flow"
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
	step := light_flow.RunStep{}
	stepP := &light_flow.RunStep{}
	if light_flow.GetStructName(d) != "Dst" {
		t.Errorf("get Dst struct name error")
	}
	if light_flow.GetStructName(dP) != "*Dst" {
		t.Errorf("get Dst pointer struct name error")
	}
	if light_flow.GetStructName(step) != "RunStep" {
		t.Errorf("get RunStep struct name error")
	}
	if light_flow.GetStructName(stepP) != "*RunStep" {
		t.Errorf("get RunStep pointer struct name error")
	}
}

func TestSet(t *testing.T) {
	s := light_flow.NewRoutineUnsafeSet[string]()
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

func TestComparableSet(t *testing.T) {
	s := light_flow.NewRoutineUnsafeSet[Dst]()
	d1 := Dst{
		Name:     "",
		Id:       1,
		Start:    time.Now(),
		Internal: Internal{},
		SameName: 0,
		UnExist:  false,
	}
	s.Add(d1)
	if !s.Contains(d1) {
		t.Errorf("set not contains d1")
	} else {
		t.Logf("set contains d1")
	}
	d2 := d1
	d2.Id = 2
	if s.Contains(d2) {
		t.Errorf("set contains d2")
	} else {
		t.Logf("set not contains d2")
	}
}

func TestPopStatus(t *testing.T) {
	status := light_flow.Status(0)
	status.AppendStatus(light_flow.Cancel)
	status.AppendStatus(light_flow.Panic)
	status.AppendStatus(light_flow.Failed)
	status.AppendStatus(light_flow.Error)
	status.AppendStatus(light_flow.Timeout)
	status.AppendStatus(light_flow.Stop)

	if !status.Contain(light_flow.Cancel) {
		t.Errorf("cancel appended but not exist")
	}
	status.Pop(light_flow.Cancel)
	if status.Contain(light_flow.Cancel) {
		t.Errorf("cancel status pop error")
	}

	if !status.Contain(light_flow.Panic) {
		t.Errorf("panic appended but not exist")
	}
	status.Pop(light_flow.Panic)
	if status.Contain(light_flow.Panic) {
		t.Errorf("panic status pop error")
	}

	if !status.Contain(light_flow.Failed) {
		t.Errorf("failed appended but not exist")
	}
	status.Pop(light_flow.Failed)
	if status.Contain(light_flow.Failed) {
		t.Errorf("failed status pop error")
	}

	if !status.Contain(light_flow.Timeout) {
		t.Errorf("timeout appended bu not exist")
	}
	status.Pop(light_flow.Timeout)
	if status.Contain(light_flow.Timeout) {
		t.Errorf("timeout status pop error")
	}

	if !status.Contain(light_flow.Stop) {
		t.Errorf("error appended bu not exist")
	}
	status.Pop(light_flow.Stop)
	if status.Contain(light_flow.Stop) {
		t.Errorf("error status pop error")
	}

	if !status.Contain(light_flow.Error) {
		t.Errorf("error appended bu not exist")
	}
	status.Pop(light_flow.Error)
	if status.Contain(light_flow.Error) {
		t.Errorf("error status pop error")
	}
}

func TestExplainStatus1(t *testing.T) {
	if status := light_flow.Status(0); !status.AppendStatus(light_flow.Cancel) {
		t.Errorf("cancel status append error")
	} else if !status.Contain(light_flow.Cancel) {
		t.Errorf("cancel status not cotain after append")
	}
	if status := light_flow.Status(0); !status.AppendStatus(light_flow.Pause) {
		t.Errorf("panic status append error")
	} else if !status.Contain(light_flow.Pause) {
		t.Errorf("pause status not cotain after append")
	}
	if status := light_flow.Status(0); !status.AppendStatus(light_flow.Running) {
		t.Errorf("running status append error")
	} else if !status.Contain(light_flow.Running) {
		t.Errorf("running status not cotain after append")
	}
	if status := light_flow.Status(0); !status.AppendStatus(light_flow.Success) {
		t.Errorf("success status append error")
	} else if !status.Contain(light_flow.Success) {
		t.Errorf("success status not cotain after append")
	}
	if status := light_flow.Status(0); !status.AppendStatus(light_flow.Failed) {
		t.Errorf("failed status append error")
	} else if !status.Contain(light_flow.Failed) {
		t.Errorf("failed status not cotain after append")
	}
	if status := light_flow.Status(0); !status.AppendStatus(light_flow.Timeout) {
		t.Errorf("timeout status append error")
	} else if !status.Contain(light_flow.Timeout) {
		t.Errorf("timeout status not cotain after append")
	}
	if status := light_flow.Status(0); !status.AppendStatus(light_flow.Panic) {
		t.Errorf("panic status append error")
	} else if !status.Contain(light_flow.Panic) {
		t.Errorf("panic status not cotain after append")
	}
	if status := light_flow.Status(0); !status.AppendStatus(light_flow.Error) {
		t.Errorf("error status append error")
	} else if !status.Contain(light_flow.Error) {
		t.Errorf("error status not cotain after append")
	}
}

func TestExplainStatus2(t *testing.T) {
	status := light_flow.Status(0)
	status.AppendStatus(light_flow.Cancel)
	status.AppendStatus(light_flow.Panic)
	status.AppendStatus(light_flow.Failed)
	status.AppendStatus(light_flow.Timeout)
	status.AppendStatus(light_flow.Error)
	status.AppendStatus(light_flow.Stop)
	status.AppendStatus(light_flow.Success)
	if status.Success() {
		t.Errorf("explain success while error occur")
	}
	if !status.Contain(light_flow.Failed) {
		t.Errorf("failed status explain error")
	}
	if !status.Contain(light_flow.Timeout) {
		t.Errorf("timeout status explain error")
	}
	if !status.Contain(light_flow.Panic) {
		t.Errorf("panic status explain error")
	}
	if !status.Contain(light_flow.Cancel) {
		t.Errorf("cancel status explain error")
	}
	if !status.Contain(light_flow.Stop) {
		t.Errorf("stop status explain error")
	}

	status = light_flow.Status(0)
	status.AppendStatus(light_flow.Pause)
	status.AppendStatus(light_flow.Running)
	status.AppendStatus(light_flow.Pending)
	if !status.Normal() {
		t.Errorf("normal status judge error")
	}
	if !status.Contain(light_flow.Pause) {
		t.Errorf("pause status explain error")
	}
	if !status.Contain(light_flow.Running) {
		t.Errorf("running status explain error")
	}
	status.AppendStatus(light_flow.Success)
	if !status.Contain(light_flow.Success) || !status.Success() {
		t.Errorf("success status explain error")
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
