package light_flow

import (
	"fmt"
	"time"
)

const (
	noneIndex int8 = iota // Avoiding the effects of default values
	boolIndex
	strIndex
	intIndex
	uintIndex
	floatIndex
	timeIndex
	equalityIndex
)

type operator int8

const (
	noneOP operator = iota // Avoiding the effects of default values
	AndOP
	OrOP
	NotOP
)

type condition int8

type typeFlags int64

const (
	noneC condition = iota // Avoiding the effects of default values
	equalC
	notEqualC
	lessC
	lessAndEqualC
	greaterC
	greaterAndEqualC
	inC
	notInC
)
const ()

type Empty struct{}

type SliceSet[T comparable] []T

type simpleSet[T comparable] map[T]Empty

type elementSet[T Element[T]] struct {
	m map[string]T
}

type evaluator struct {
	op        operator  // int8 And Or Not
	cond      condition // int8 EQ NE LT LE GT GE IN NOTIN
	flags     typeFlags // int64 bool string int uint float time
	target    string    // only the head node's target value is not null
	name      string    // only the head node's name is not null
	criterias []interface{}
	evaluate  func(e evalValues, flags typeFlags) (typeFlags, bool)
	next      *evaluator
}

type condStep struct {
	*StepMeta
	evaluators []evaluator
}

type Set interface {
	Find(key any) bool
}

type Element[T any] interface {
	Equality
	String() string
}

type Equality interface {
	Equal(other any) bool
}

type Comparable[T any] interface {
	Equality
	Less(other T) bool
}

func (tf *typeFlags) SetFlag(index int8) {
	*tf |= 1 << index
}

func (tf *typeFlags) ClearFlag(index int8) {
	*tf &= ^(1 << index)
}

func (tf *typeFlags) Exist(index int8) bool {
	return *tf&(1<<index) != 0
}

func (ss simpleSet[T]) Add(element T) {
	ss[element] = Empty{}
}

func (ss simpleSet[T]) Remove(element T) {
	delete(ss, element)
}

func (ss simpleSet[T]) Find(element T) bool {
	_, ok := ss[element]
	return ok
}

func (ss SliceSet[T]) Find(element any) bool {
	convert, ok := element.(T)
	if !ok {
		return false
	}
	for _, t := range ss {
		if t == convert {
			return true
		}
	}
	return false
}

func (es *elementSet[T]) Add(element T) {
	es.m[element.String()] = element
}

func (es *elementSet[T]) Remove(element T) {
	delete(es.m, element.String())
}

func (es *elementSet[T]) Find(element any) bool {
	ele, ok := element.(T)
	if !ok {
		return false
	}
	value, ok := es.m[ele.String()]
	if !ok {
		return false
	}
	if value.Equal(ele) {
		return true
	}
	return false
}

func NewSimpleSet[T comparable]() simpleSet[T] {
	return make(simpleSet[T], 1)
}

func NewElementSet[T Element[T]]() elementSet[T] {
	return elementSet[T]{m: make(map[string]T)}
}

// EQ reflect is inefficient, so I have to write a pile of shit.
func (meta *StepMeta) EQ(depend interface{}, value ...any) {
	meta.AddDepend(depend)
	target := toStepName(depend)
	e := &evaluator{
		cond:      equalC,
		name:      meta.stepName,
		target:    target,
		criterias: value,
	}
	meta.evaluators = append(meta.evaluators, e)
	flags, types := buildTypeMap(meta.stepName, target, value...)
	f := buildEqEvaluate(meta.stepName, target, types)
	e.flags = flags
	e.evaluate = f
}

func NEQ(depend interface{}, value any) {

}

func True() {}

func False() {

}

func GT() {}

func GTE() {}

func LT() {}

func LTE() {}

func In() {}

func NotIn() {}

func (meta *StepMeta) AddDepend(depends ...any) {
	for _, wrap := range depends {
		dependName := toStepName(wrap)
		depend, exist := meta.belong.steps[dependName]
		if !exist {
			panic(fmt.Sprintf("step[%s]'s depend[%s] not found.]", meta.stepName, dependName))
		}
		meta.depends = append(meta.depends, depend)
	}
	meta.wireDepends()
}

func buildEqEvaluate(name, target string, types map[int8]any) func(e evalValues, flags typeFlags) (typeFlags, bool) {
	f := func(e evalValues, flags typeFlags) (typeFlags, bool) {
		for _, current := range e {
			if current.matches != nil && !current.matches.Contains(name) {
				continue
			}
			switch current.value.(type) {
			case bool:
				if criteria, exist := types[boolIndex]; exist {
					flags.SetFlag(boolIndex)
					return flags, criteria.(bool) == current.value.(bool)
				}
			case string:
				if criteria, exist := types[strIndex]; exist {
					flags.SetFlag(strIndex)
					return flags, criteria.(string) == current.value.(string)
				}
			case int8, int16, int32, int64, int:
				if criteria, exist := types[intIndex]; exist {
					flags.SetFlag(intIndex)
					return flags, toInt64(current.value) == criteria.(int64)
				}
			case uint8, uint16, uint32, uint64, uint:
				if criteria, exist := types[uintIndex]; exist {
					flags.SetFlag(uintIndex)
					return flags, toUint64(current.value) == criteria.(uint64)
				}
			case float32, float64:
				if criteria, exist := types[floatIndex]; exist {
					flags.SetFlag(floatIndex)
					return flags, toFloat64(current.value) == criteria.(float64)
				}
			case time.Time:
				if criteria, exist := types[timeIndex]; exist {
					flags.SetFlag(timeIndex)
					return flags, criteria.(time.Time) == current.value.(time.Time)
				}
			case Equality:
				if criteria, exist := types[equalityIndex]; exist {
					flags.SetFlag(equalityIndex)
					return flags, criteria.(Equality).Equal(current.value)
				}
			}
		}
		return flags, false
	}
	return f
}

func buildTypeMap(name, target string, values ...any) (typeFlags, map[int8]any) {
	types := make(map[int8]any)
	// condition match has a high time complexity,must limit the number of conditions
	for _, value := range values {
		switch value.(type) {
		case bool:
			if _, ok := types[boolIndex]; ok {
				panic(fmt.Sprintf("step[%s] set condition [%s]EQ failed, type bool has already exist", name, target))
			}
			types[boolIndex] = value
		case string:
			if _, ok := types[strIndex]; ok {
				panic(fmt.Sprintf("step[%s] set condition [%s]EQ set failed, type string has already exist", name, target))
			}
			types[strIndex] = value
		case int8, int16, int32, int64, int:
			if _, ok := types[intIndex]; ok {
				panic(fmt.Sprintf("step[%s] set condition [%s]EQ failed, type int has already exist", name, target))
			}
			types[intIndex] = toInt64(value)
		case uint8, uint16, uint32, uint64, uint:
			if _, ok := types[uintIndex]; ok {
				panic(fmt.Sprintf("step[%s] set condition [%s]EQ failed, type uint has already exist", name, target))
			}
			types[uintIndex] = toUint64(value)
		case float32, float64:
			if _, ok := types[floatIndex]; ok {
				panic(fmt.Sprintf("step[%s] set condition [%s]EQ failed, type float has already exist", name, target))
			}
			types[floatIndex] = toFloat64(value)
		case time.Time:
			if _, ok := types[timeIndex]; ok {
				panic(fmt.Sprintf("step[%s] set condition condition [%s]EQ failed, type time.Time has already exist", name, target))
			}
			types[timeIndex] = value
		case Equality:
			if _, ok := types[equalityIndex]; ok {
				panic(fmt.Sprintf("step[%s] set condition [%s]EQ failed, type Equality has already exist", name, target))
			}
			types[equalityIndex] = value
		default:
			panic(fmt.Sprintf("step[%s] set condition [%s]EQ failed, type[%T] has not Equality, please implement Equality interface", name, target, value))
		}
	}
	flags := typeFlags(0)
	for index := range types {
		flags.SetFlag(index)
	}
	return flags, types
}

func toUint64(i any) uint64 {
	switch v := i.(type) {
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uint64:
		return v
	case uint:
		return uint64(v)
	}
	return 0
}

func toInt64(i any) int64 {
	switch v := i.(type) {
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

func toFloat64(i any) float64 {
	switch v := i.(type) {
	case float32:
		return float64(v)
	case float64:
		return v
	}
	return 0
}
