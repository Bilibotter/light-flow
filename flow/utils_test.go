package flow

import (
	"bytes"
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

type FConfig struct {
}

type PConfig struct {
}

func fn1() {
	panic("fn1 invoke fail")
}

func fn2() {
	fn1()
}

func fn3() {
	fn2()
}

func fn4() {
	fn3()
}

func TestValidIdentifier(t *testing.T) {
	prefix := "AB"
	identifiers := []string{prefix, "Valid123", "1234567890", "abcdefghijklmnopqrstuvwxyz", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", "ab"}
	for _, id := range identifiers {
		if !isValidIdentifier(id) {
			t.Errorf("'%s' is not a valid identifier, but it should be.\n", id)
		}
	}
	matches := strings.Split(`" # $ % & ' ( ) * + , - . / : ; < = > ? @ [ \ ] ^ _ { | } ~ ¢ £ ¤ ¥ ₠ ₡ ₢ ₣ ₤ ₥ ₦ ₧ ₨ ₩ ₪ ₫ € ₭ ₮ ₯ ₹ ₲ ₳ ₴ ₵ ₶ ₷ ₸ § ¶ © ® ™ ℗ ∞ ∆ ∇ ∏ ∑ ∂ ∴ ∵ ∷ ∈ ∉ ∋ ∌ ∅ ∇ ∆ ∏ ∐ ∑ ( ) [ ] { } < > ' "`, " ")
	for _, match := range matches {
		if isValidIdentifier(match) {
			t.Errorf("'%s' is a valid identifier, but it should not be.\n", match)
		}
		if isValidIdentifier(prefix + match) {
			t.Errorf("'%s' is a valid identifier, but it should not be.\n", prefix+match)
		}
	}
}

func TestNamedFuncStackTrace(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			bs := stack()
			if !bytes.Contains(bs, []byte("fn1")) {
				t.Errorf("fn1 trace fail")
			}
			if !bytes.Contains(bs, []byte("fn2")) {
				t.Errorf("fn2 trace fail")
			}
			if !bytes.Contains(bs, []byte("fn3")) {
				t.Errorf("fn3 trace fail")
			}
			if !bytes.Contains(bs, []byte("fn4")) {
				t.Errorf("fn4 trace fail")
			}
			t.Logf("[Recovery] %s panic recovered:\n%s\n%s\n", time.Now().Format("2006/01/02 - 15:04:05"), r, bs)
		}
	}()
	fn4()
}

func TestGetFuncName(t *testing.T) {
	if getFuncName(TestGetFuncName) != "TestGetFuncName" {
		t.Errorf("get TestGetFuncName name error")
	}
	if getFuncName(getStructName) != "getStructName" {
		t.Errorf("get getStructName name error")
	}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("test used get func name to anonymous function, panic: %v", r)
		}
	}()
	if strings.HasPrefix(getFuncName(func() {}), "func") {
		t.Errorf("get anonymous function name error")
	}
}

func TestGetStructName(t *testing.T) {
	d := Dst{}
	dP := &Dst{}
	step := FlowMeta{}
	stepP := &FlowMeta{}
	if getStructName(d) != "Dst" {
		t.Errorf("get Dst struct name error")
	}
	if getStructName(dP) != "*Dst" {
		t.Errorf("get Dst pointer struct name error")
	}
	if getStructName(step) != "FlowMeta" {
		t.Errorf("get FlowMeta struct name error")
	}
	if getStructName(stepP) != "*FlowMeta" {
		t.Errorf("get FlowMeta pointer struct name error")
	}
}

func TestSet(t *testing.T) {
	s := newRoutineUnsafeSet[string]()
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
	s := newRoutineUnsafeSet[Dst]()
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
	status := state(0)
	status.append(Cancel)
	status.append(Panic)
	status.append(Failed)
	status.append(Error)
	status.append(Timeout)
	status.append(Stop)

	if !status.Has(Cancel) {
		t.Errorf("cancel appended but not exist")
	}
	status.clear(Cancel)
	status.append(Failed)
	if status.Has(Cancel) {
		t.Errorf("cancel status pop error")
	}

	if !status.Has(Panic) {
		t.Errorf("panic appended but not exist")
	}
	status.clear(Panic)
	status.append(Failed)
	if status.Has(Panic) {
		t.Errorf("panic status pop error")
	}

	if !status.Has(Failed) {
		t.Errorf("failed appended but not exist")
	}
	status.clear(Failed)
	if status.Has(Failed) {
		t.Errorf("failed status pop error")
	}
	status.append(Failed)
	if !status.Has(Timeout) {
		t.Errorf("timeout appended bu not exist")
	}
	status.clear(Timeout)
	status.append(Failed)
	if status.Has(Timeout) {
		t.Errorf("timeout status pop error")
	}

	if !status.Has(Stop) {
		t.Errorf("error appended bu not exist")
	}
	status.clear(Stop)
	status.append(Failed)
	if status.Has(Stop) {
		t.Errorf("error status pop error")
	}

	if !status.Has(Error) {
		t.Errorf("error appended bu not exist")
	}
	status.clear(Error)
	status.append(Failed)
	if status.Has(Error) {
		t.Errorf("error status pop error")
	}
}

func TestExplainStatus1(t *testing.T) {
	if status := state(0); !status.append(Cancel) {
		t.Errorf("cancel status add error")
	} else if !status.Has(Cancel) {
		t.Errorf("cancel status not contain after add")
	}
	if status := state(0); !status.append(Pause) {
		t.Errorf("panic status add error")
	} else if !status.Has(Pause) {
		t.Errorf("pause status not contain after add")
	}
	if status := state(0); !status.append(Success) {
		t.Errorf("success status add error")
	} else if !status.Has(Success) {
		t.Errorf("success status not contain after add")
	}
	if status := state(0); !status.append(Failed) {
		t.Errorf("failed status add error")
	} else if !status.Has(Failed) {
		t.Errorf("failed status not contain after add")
	}
	if status := state(0); !status.append(Timeout) {
		t.Errorf("timeout status add error")
	} else if !status.Has(Timeout) {
		t.Errorf("timeout status not contain after add")
	}
	if status := state(0); !status.append(Panic) {
		t.Errorf("panic status add error")
	} else if !status.Has(Panic) {
		t.Errorf("panic status not contain after add")
	}
	if status := state(0); !status.append(Error) {
		t.Errorf("error status add error")
	} else if !status.Has(Error) {
		t.Errorf("error status not contain after add")
	}
}

func TestExplainStatus2(t *testing.T) {
	status1 := state(0)
	status1.append(Cancel)
	status1.append(Panic)
	status1.append(Failed)
	status1.append(Timeout)
	status1.append(Error)
	status1.append(Stop)
	status1.append(Success)
	if status1.Success() {
		t.Errorf("explain success while error occur")
	}
	if !status1.Has(Failed) {
		t.Errorf("failed status explain error")
	}
	if !status1.Has(Timeout) {
		t.Errorf("timeout status explain error")
	}
	if !status1.Has(Panic) {
		t.Errorf("panic status explain error")
	}
	if !status1.Has(Cancel) {
		t.Errorf("cancel status explain error")
	}
	if !status1.Has(Stop) {
		t.Errorf("stop status explain error")
	}

	status1 = state(0)
	status1.append(Pause)
	status1.append(Pending)
	if !status1.Normal() {
		t.Errorf("normal status judge error")
	}
	if !status1.Has(Pause) {
		t.Errorf("pause status explain error")
	}
	status1.append(Success)
	if !status1.Has(Success) || !status1.Success() {
		t.Errorf("success status explain error")
	}
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
	copyPropertiesWithMerge(&src, &dst)
	if dst.Name != src.Name {
		t.Errorf("dst.GetCtxName != src.GetCtxName")
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
}
