package light_flow

import (
	"testing"
	"time"
)

type Person struct {
	Name string
	Age  int
}

type Named interface {
	name() string
}

func (p *Person) name() string {
	return p.Name
}

func TestAllType(t *testing.T) {
	RegisterType[Person]()
	named := &Person{Name: "Alice", Age: 30}
	ot1 := outcome{
		Result: "Anne",
		Named: []evalValue{
			{
				Value:   "named",
				Matches: newRoutineUnsafeSet("named"),
			},
		},
		Unnamed: []evalValue{
			{
				Value:   "unamed",
				Matches: newRoutineUnsafeSet("unamed"),
			},
		},
	}
	otv1 := outcomeValue{
		O:     ot1,
		Value: "Amy",
	}
	m1 := map[string]any{}
	m1["uint"] = uint(128)
	m1["uint8"] = uint8(1)
	m1["uint16"] = uint16(1)
	m1["uint32"] = uint32(1)
	m1["uint64"] = uint64(1)
	m1["int"] = int(128)
	m1["int8"] = int8(1)
	m1["int16"] = int16(1)
	m1["int32"] = int32(1)
	m1["int64"] = int64(1)
	m1["float32"] = float32(1)
	m1["float64"] = float64(1)
	m1["string"] = "foo"
	m1["bool"] = true
	m1["Person"] = Person{Name: "Alice", Age: 30}
	m1["*Person"] = &Person{Name: "Anne", Age: 18}
	now := time.Now()
	m1["time.Time"] = time.Now()
	m1["*time.Time"] = &now
	m1["Named"] = named
	m1["outcome"] = ot1
	m1["*outcome"] = &ot1
	m1["outcomeValue"] = otv1
	m1["*outcomeValue"] = &otv1
	bs, err := serialize(m1)
	if err != nil {
		t.Errorf("serialize error: %v", err)
		return
	}
	m2, err := deserialize[map[string]any](bs)
	if err != nil {
		t.Errorf("deserialize error: %v", err)
		return
	}
	if u, ok := m2["uint"].(uint); !ok {
		t.Errorf("uint transfer to %T，value=%#v\n", m2["uint"], m2["uint"])
	} else if u != m1["uint"].(uint) {
		t.Errorf("uint serialize failed\n")
	}
	if _, ok := m2["uint8"].(uint8); !ok {
		t.Errorf("uint8 transfer to %T, value=%#v\n", m2["uint8"], m2["uint8"])
	} else if m1["uint8"].(uint8) != m2["uint8"].(uint8) {
		t.Errorf("uint8 serialize failed")
	}
	if _, ok := m2["uint16"].(uint16); !ok {
		t.Errorf("uint16 transfer to %T, value=%#v\n", m2["uint16"], m2["uint16"])
	} else if m1["uint16"].(uint16) != m2["uint16"].(uint16) {
		t.Errorf("uint16 serialize failed")
	}
	if _, ok := m2["uint32"].(uint32); !ok {
		t.Errorf("uint32 transfer to %T, value=%#v\n", m2["uint32"], m2["uint32"])
	} else if m1["uint32"].(uint32) != m2["uint32"].(uint32) {
		t.Errorf("uint32 serialize failed")
	}
	if _, ok := m2["uint64"].(uint64); !ok {
		t.Errorf("uint64 transfer to %T, value=%#v\n", m2["uint64"], m2["uint64"])
	} else if m1["uint64"].(uint64) != m2["uint64"].(uint64) {
		t.Errorf("uint64 serialize failed")
	}
	if _, ok := m2["int"].(int); !ok {
		t.Errorf("int transfer to %T, value=%#v\n", m2["int"], m2["int"])
	} else if m1["int"].(int) != m2["int"].(int) {
		t.Errorf("int serialize failed")
	}
	if _, ok := m2["int8"].(int8); !ok {
		t.Errorf("int8 transfer to %T, value=%#v\n", m2["int8"], m2["int8"])
	} else if m1["int8"].(int8) != m2["int8"].(int8) {
		t.Errorf("int8 serialize failed")
	}
	if _, ok := m2["int16"].(int16); !ok {
		t.Errorf("int16 transfer to %T, value=%#v\n", m2["int16"], m2["int16"])
	} else if m1["int16"].(int16) != m2["int16"].(int16) {
		t.Errorf("int16 serialize failed")
	}
	if _, ok := m2["int32"].(int32); !ok {
		t.Errorf("int32 transfer to %T, value=%#v\n", m2["int32"], m2["int32"])
	} else if m1["int32"].(int32) != m2["int32"].(int32) {
		t.Errorf("int32 serialize failed")
	}
	if _, ok := m2["int64"].(int64); !ok {
		t.Errorf("int64 transfer to %T, value=%#v\n", m2["int64"], m2["int64"])
	} else if m1["int64"].(int64) != m2["int64"].(int64) {
		t.Errorf("int64 serialize failed")
	}
	if m2["float32"].(float32) != m1["float32"].(float32) {
		t.Errorf("float32 transfer to %T, value=%#v\n", m2["float32"], m2["float32"])
	} else if m1["float32"].(float32) != m2["float32"].(float32) {
		t.Errorf("float32 serialize failed")
	}
	if m2["float64"].(float64) != m1["float64"].(float64) {
		t.Errorf("float64 transfer to %T, value=%#v\n", m2["float64"], m2["float64"])
	} else if m1["float64"].(float64) != m2["float64"].(float64) {
		t.Errorf("float64 serialize failed")
	}
	if m2["string"].(string) != m1["string"].(string) {
		t.Errorf("string transfer to %T, value=%#v\n", m2["string"], m2["string"])
	} else if m1["string"].(string) != m2["string"].(string) {
		t.Errorf("string serialize failed")
	}
	if m2["bool"].(bool) != m1["bool"].(bool) {
		t.Errorf("bool transfer to %T, value=%#v\n", m2["bool"], m2["bool"])
	} else if m1["bool"].(bool) != m2["bool"].(bool) {
		t.Errorf("bool serialize failed")
	}
	if !m2["time.Time"].(time.Time).Equal(m1["time.Time"].(time.Time)) {
		t.Errorf("time.Time transfer to %T, value=%#v\n", m2["time.Time"], m2["time.Time"])
	} else if !m1["time.Time"].(time.Time).Equal(m2["time.Time"].(time.Time)) {
		t.Errorf("time.Time serialize failed")
	}
	if !m2["*time.Time"].(*time.Time).Equal(now) {
		t.Errorf("*time.Time transfer to %T, value=%#v\n", m2["*time.Time"], m2["*time.Time"])
	}
	if m2["Person"].(Person).Name != m1["Person"].(Person).Name {
		t.Errorf("struct transfer failed to %T, value=%#v\n", m2["Person"], m2["Person"])
	} else if m1["Person"].(Person).Name != m2["Person"].(Person).Name {
		t.Errorf("struct serialize failed")
	}
	if m2["*Person"].(*Person).Name != "Anne" {
		t.Errorf("*struct transfer failed to %T, value=%#v\n", m2["*Person"], m2["*Person"])
	} else if m2["*Person"].(*Person).Age != 18 {
		t.Errorf("*struct serialize failed")
	}
	if m2["Person"].(Person).Age != m1["Person"].(Person).Age {
		t.Errorf("Person Age not match")
	}
	if m2["*Person"].(*Person).Name != "Anne" {
		t.Errorf("*Person Name not match")
	}
	if m2["*Person"].(*Person).Age != 18 {
		panic("*Person Age not match")
	}
	if m2["Named"].(Named).name() != "Alice" {
		t.Errorf("interface transfer failed to %T, value=%#v\n", m2["Named"], m2["Named"])
	} else if m2["Named"].(*Person).Age != 30 {
		t.Errorf("interface serialize failed")
	}
	ot2 := m2["outcome"].(outcome)
	if ot2.Result.(string) != ot1.Result.(string) {
		t.Errorf("outcome serialize failed,Result not equal")
	}
	if ot2.Named[0].Value.(string) != ot1.Named[0].Value.(string) {
		t.Errorf("outcome serialize failed,Named not equal")
	}
	if ot2.Unnamed[0].Value.(string) != ot1.Unnamed[0].Value.(string) {
		t.Errorf("outcome serialize failed,Unnamed not equal")
	}
	for k := range ot1.Named[0].Matches.Data {
		if ot2.Named[0].Matches.Data[k] != ot1.Named[0].Matches.Data[k] {
			t.Errorf("outcome serialize failed,*set not equal")
		}
	}
	ot3 := m2["*outcome"].(*outcome)
	if ot3.Result.(string) != ot1.Result.(string) {
		t.Errorf("*outcome serialize failed,Result not equal")
	}
	if ot3.Named[0].Value.(string) != ot1.Named[0].Value.(string) {
		t.Errorf("*outcome serialize failed,Named not equal")
	}
	if ot3.Unnamed[0].Value.(string) != ot1.Unnamed[0].Value.(string) {
		t.Errorf("*outcome serialize failed,Unnamed not equal")
	}
	for k := range ot1.Named[0].Matches.Data {
		if ot3.Named[0].Matches.Data[k] != ot1.Named[0].Matches.Data[k] {
			t.Errorf("*outcome serialize failed,*set not equal")
		}
	}
	otv2 := m2["outcomeValue"].(outcomeValue)
	if otv2.Value.(string) != otv1.Value.(string) {
		t.Errorf("outcomeValue serialize failed")
	}
	if otv2.O.Result.(string) != otv1.O.Result.(string) {
		t.Errorf("outcome serialize failed,Result not equal")
	}
	if otv2.O.Named[0].Value.(string) != otv1.O.Named[0].Value.(string) {
		t.Errorf("outcome serialize failed,Named not equal")
	}
	if otv2.O.Unnamed[0].Value.(string) != otv1.O.Unnamed[0].Value.(string) {
		t.Errorf("outcome serialize failed,Unnamed not equal")
	}
	for k := range ot1.Named[0].Matches.Data {
		if otv2.O.Named[0].Matches.Data[k] != otv1.O.Named[0].Matches.Data[k] {
			t.Errorf("outcome serialize failed,*set not equal")
		}
	}
	otv3 := m2["*outcomeValue"].(*outcomeValue)
	if otv3.Value.(string) != otv1.Value.(string) {
		t.Errorf("*outcomeValue serialize failed")
	}
	if otv3.O.Result.(string) != otv1.O.Result.(string) {
		t.Errorf("*outcome serialize failed,Result not equal")
	}
	if otv3.O.Named[0].Value.(string) != otv1.O.Named[0].Value.(string) {
		t.Errorf("*outcome serialize failed,Named not equal")
	}
	if otv3.O.Unnamed[0].Value.(string) != otv1.O.Unnamed[0].Value.(string) {
		t.Errorf("*outcome serialize failed,Unnamed not equal")
	}
	for k := range ot1.Named[0].Matches.Data {
		if otv3.O.Named[0].Matches.Data[k] != otv1.O.Named[0].Matches.Data[k] {
			t.Errorf("*outcome serialize failed,*set not equal")
		}
	}
}
