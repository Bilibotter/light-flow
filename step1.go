package light_flow

import (
	"fmt"
	"math"
	"time"
)

const (
	noneFlag int64 = 0
	boolF    int64 = 1 << iota
	strF
	intF
	uintF
	floatF
	timeF
	equalityF
)

var (
	flagName = map[int64]string{boolF: "bool", strF: "string", intF: "int", uintF: "uint", floatF: "float", timeF: "time", equalityF: "Equality"}
)

var (
	accurate float64 = 1e-9
)

type operator int8

const (
	noneOP operator = iota // Avoiding the effects of default values
	andOP
	orOP
	notOP
)

type condition int8

type flags int64

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

type evalGroup struct {
	evaluators []*evaluator
	target     string
	name       string // only exact match evaluator has name
}

type evaluator struct {
	op        operator  // int8 And Or Not
	cond      condition // int8 EQ NE LT LE GT GE IN NOTIN
	flags     flags     // int64 bool string int uint float time
	criterias []interface{}
	evaluate  func(e evalValues, flags flags) (flags, bool)
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

// Equality func (ei *EqualityImpl) Equal(other any) bool is invalid
// func (ei *EqualityImpl) Equal(other any) bool is valid
type Equality interface {
	Equal(other any) bool
}

type Comparable[T any] interface {
	Equality
	Less(other T) bool
}

func (eg *evalGroup) evaluate(named, unnamed evalValues) bool {
	flag, ok := flags(0), false
	for _, f := range eg.evaluators {
		// must evaluate unnamed first,
		// otherwise flag will make evaluate named failed
		flag, ok = f.evaluate(unnamed, flag)
		if !ok {
			return false
		}
		flag, ok = f.evaluate(named, flag)
		if !ok {
			return false
		}
	}
	return true
}

func (tf *flags) SetFlag(flag int64) {
	*tf |= flags(flag)
}

func (tf *flags) ClearFlag(flag int64) {
	*tf &= ^flags(flag)
}

func (tf *flags) Exist(flag int64) bool {
	return *tf&flags(flag) != 0
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

func (meta *StepMeta) EQ(depend interface{}, value ...any) {
	meta.AddDepend(depend)
	target := toStepName(depend)
	group := &evalGroup{
		name:   toStepName(depend),
		target: target,
	}
	e := &evaluator{
		cond:      equalC,
		criterias: value,
	}
	e.flags, e.evaluate = buildEqEvaluate(meta.stepName, target, value...)
	group.evaluators = append(group.evaluators, e)
	meta.evalGroups = append(meta.evalGroups, group)
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

// buildEqEvaluate flags is the set of values's type
func buildEqEvaluate(name, target string, values ...any) (flags, func(e evalValues, flags flags) (flags, bool)) {
	already, types := buildTypeMap(name, target, values...)
	f := func(e evalValues, flags flags) (flags, bool) {
		for _, current := range e {
			flag, convert := matchAndTransform(name, types, &flags, current)
			switch flag {
			case boolF:
				if convert.(bool) != types[boolF].(bool) {
					return flags, false
				}
			case strF:
				if convert.(string) != types[strF].(string) {
					return flags, false
				}
			case intF:
				if convert.(int64) != types[intF].(int64) {
					return flags, false
				}
			case uintF:
				if convert.(uint64) != types[uintF].(uint64) {
					return flags, false
				}
			case floatF:
				if math.Abs(convert.(float64)-types[floatF].(float64)) > accurate {
					return flags, false
				}
			case timeF:
				if !convert.(time.Time).Equal(types[timeF].(time.Time)) {
					return flags, false
				}
			case equalityF:
				if !types[equalityF].(Equality).Equal(convert) {
					return flags, false
				}
			}
		}
		return flags, flags.Exist(int64(already))
	}
	return already, f
}

func buildTypeMap(name, target string, values ...any) (flags, map[int64]any) {
	types := make(map[int64]any)
	// condition match has a high time complexity,must limit the number of conditions
	for _, value := range values {
		switch value.(type) {
		case bool:
			checkAndAdd(name, target, boolF, value, types)
		case string:
			checkAndAdd(name, target, strF, value, types)
		case int, int8, int16, int32, int64:
			checkAndAdd(name, target, intF, value, types)
		case uint, uint8, uint16, uint32, uint64:
			checkAndAdd(name, target, uintF, value, types)
		case float32, float64:
			checkAndAdd(name, target, floatF, value, types)
		case time.Time:
			checkAndAdd(name, target, timeF, value, types)
		case Equality:
			checkAndAdd(name, target, equalityF, value, types)
		default:
			panic(fmt.Sprintf("Step[%s] set condition Step[%s]EQ failed, Type[%T] has not Equality, please implement Equality interface or not use pointers as method receivers", name, target, value))
		}
	}
	already := flags(0)
	for index := range types {
		already.SetFlag(index)
	}
	return already, types
}

func checkAndAdd(name, target string, flag int64, value any, types map[int64]any) {
	if _, ok := types[flag]; ok {
		panic(fmt.Sprintf("Step[%s] set condition Step[%s]EQ failed, Type[%T] has not Equality, please implement Equality interface or not use pointers as method receivers", name, target, value))
	}
	switch flag {
	case intF:
		types[flag] = toInt64(value)
	case uintF:
		types[flag] = toUint64(value)
	case floatF:
		types[flag] = toFloat64(value)
	default:
		types[flag] = value
	}
}

func matchAndTransform(name string, types map[int64]any, f *flags, e *evalValue) (flag int64, convert interface{}) {
	if e.matches != nil && !e.matches.Contains(name) {
		return noneFlag, nil
	}
	switch e.value.(type) {
	case bool:
		return trySetFlagAndConvert(boolF, types, f, e)
	case string:
		return trySetFlagAndConvert(strF, types, f, e)
	case int64:
		// int8, int16, int32, int64, int has converted to int64 while setCond
		return trySetFlagAndConvert(intF, types, f, e)
	case uint64:
		// uint8, uint16, uint32, uint64, uint has converted to uint64 while setCond
		return trySetFlagAndConvert(uintF, types, f, e)
	case float64:
		// float32, float64 has converted to float64 while setCond
		return trySetFlagAndConvert(floatF, types, f, e)
	case time.Time:
		return trySetFlagAndConvert(timeF, types, f, e)
	case Equality:
		return trySetFlagAndConvert(equalityF, types, f, e)
	}
	return noneFlag, nil
}

func trySetFlagAndConvert(match int64, types map[int64]any, f *flags, e *evalValue) (flag int64, convert interface{}) {
	// types does not have a key with a value of nil
	if f.Exist(match) || types[match] == nil {
		return noneFlag, nil
	}
	f.SetFlag(match)
	return match, e.value
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
