package light_flow

import (
	"fmt"
	"math"
	"time"
)

type typeFlag int64

const (
	boolF typeFlag = 1 << iota
	strF
	intF
	uintF
	floatF
	timeF
	equalityF
	noneFlag typeFlag = 0
)

var (
	accurate float64 = 1e-9
)

type logical int8

const (
	noneLg logical = iota // Avoiding the effects of default values
	andLg
	orLg
	notLg
)

type comparator int8

type flags int64

const (
	noneC comparator = iota // Avoiding the effects of default values
	equalC
	notEqualC
	lessC
	lessAndEqualC
	greaterC
	greaterAndEqualC
	inC
	notInC
)

type evaluate func(value1, value2 any) bool

var (
	defaultTrue evaluate = func(value1, value2 any) bool { return true }
)

type Empty struct{}

type SliceSet[T comparable] []T

type simpleSet[T comparable] map[T]Empty

type elementSet[T Element[T]] struct {
	m map[string]T
}

type evalGroup struct {
	evaluator    func(named, unnamed evalValues) bool
	typeValues   map[typeFlag]any
	typeEvaluate map[typeFlag]evaluate
	name         string // only exact match evaluator has name
	depend       string
	expect       flags
}

type evaluator struct {
	op          logical    // int8 And Or Not
	comparator  comparator // int8 EQ NE LT LE GT GE IN NOTIN
	typeFlags   flags      // int64 bool string int uint float time
	comparative []interface{}
	evaluate    func(e evalValues, flags flags) (flags, bool)
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

func getTypeFlag(value any) typeFlag {
	switch value.(type) {
	case bool:
		return boolF
	case string:
		return strF
	case int, int8, int16, int32, int64:
		return intF
	case uint, uint8, uint16, uint32, uint64:
		return uintF
	case float32, float64:
		return floatF
	case time.Time:
		return timeF
	case Equality:
		return equalityF
	}
	return noneFlag
}

func (eg *evalGroup) update(cmp comparator, values ...any) {
	for _, value := range values {
		normalized, flag := normalizeValue(value, cmp)
		if _, exist := eg.typeValues[flag]; exist {
			panic(fmt.Sprintf("Type[%T] duplicate", normalized))
		}
		eg.typeValues[flag] = normalized
		eg.typeEvaluate[flag] = eg.createEvaluate(flag, cmp)
		eg.expect.SetFlag(int64(flag))
	}
}

func (eg *evalGroup) createEvaluate(flag typeFlag, cmp comparator) evaluate {
	switch flag {
	case intF:
		return eg.createInt64Evaluate(cmp)
	case boolF:
		return eg.createBoolEvaluate(cmp)
	case uintF:
		return eg.createUint64Evaluate(cmp)
	case floatF:
		return eg.createFloat64Evaluate(cmp)
	case strF:
		return eg.createStringEvaluate(cmp)
	case timeF:
		return eg.createTimeEvaluate(cmp)
	case equalityF:
		return eg.createEqualityEvaluate(cmp)
	}
	panic(fmt.Sprintf("Unexpected error occurred while createEvaluate"))
}
func (eg *evalGroup) createInt64Evaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(int64) == v2.(int64) }
	case notEqualC:
		return func(v1, v2 any) bool { return v1.(int64) != v2.(int64) }
	}
	return defaultTrue
}

func (eg *evalGroup) createUint64Evaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(uint64) == v2.(uint64) }
	case notEqualC:
		return func(v1, v2 any) bool { return v1.(uint64) != v2.(uint64) }
	}
	return defaultTrue
}

func (eg *evalGroup) createFloat64Evaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return math.Abs(v1.(float64)-v2.(float64)) <= accurate }
	case notEqualC:
		return func(v1, v2 any) bool { return math.Abs(v1.(float64)-v2.(float64)) > accurate }
	}
	return defaultTrue
}

func (eg *evalGroup) createBoolEvaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(bool) == v2.(bool) }
	case notEqualC:
		return func(v1, v2 any) bool { return v1.(bool) != v2.(bool) }
	}
	return defaultTrue
}

func (eg *evalGroup) createTimeEvaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(time.Time).Equal(v2.(time.Time)) }
	case notEqualC:
		return func(v1, v2 any) bool { return !v1.(time.Time).Equal(v2.(time.Time)) }
	}
	return defaultTrue
}

func (eg *evalGroup) createStringEvaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(string) == v2.(string) }
	case notEqualC:
		return func(v1, v2 any) bool { return v1.(string) != v2.(string) }
	}
	return defaultTrue
}

func (eg *evalGroup) evaluateConditions(named, unnamed evalValues) bool {
	flag := flags(0)
	if !eg.evaluate(&flag, named) {
		return false
	}
	if !eg.evaluate(&flag, unnamed) {
		return false
	}
	return eg.expect == flag
}

func (eg *evalGroup) createEqualityEvaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(Equality).Equal(v2) }
	case notEqualC:
		return func(v1, v2 any) bool { return !v1.(Equality).Equal(v2) }
	}
	return defaultTrue
}

func (eg *evalGroup) evaluate(flag *flags, values evalValues) bool {
	for _, v := range values {
		if v.matches != nil && !v.matches.Contains(eg.name) {
			continue
		}
		if !eg.evaluateByType(flag, getTypeFlag(v.value), v.value) {
			return false
		}
	}
	// this is intermediate result, only if eg.expect == flags is fully evaluated
	return true
}

func (eg *evalGroup) evaluateByType(current *flags, flag typeFlag, value any) bool {
	if current.Exist(int64(flag)) || eg.typeEvaluate[flag] == nil {
		return true
	}
	if !eg.typeEvaluate[flag](value, eg.typeValues[flag]) {
		return false
	}
	current.SetFlag(int64(flag))
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

func (meta *StepMeta) EQ(depend interface{}, value ...any) *evalGroup {
	meta.addDepend(depend)
	return meta.addEvalGroup(toStepName(depend), equalC, value...)
}

func (meta *StepMeta) addEvalGroup(depend string, cmp comparator, values ...any) *evalGroup {
	group := &evalGroup{
		name:         meta.stepName,
		depend:       depend,
		typeValues:   make(map[typeFlag]any, 1),
		typeEvaluate: make(map[typeFlag]evaluate, 1),
	}
	meta.evalGroups = append(meta.evalGroups, group)
	group.update(cmp, values...)
	return group
}

func normalizeValue(value any, cmp comparator) (any, typeFlag) {
	switch value.(type) {
	case bool:
		return value, boolF
	case string:
		return value, strF
	case int, int8, int16, int32, int64:
		return toInt64(value), intF
	case uint, uint8, uint16, uint32, uint64:
		return toUint64(value), uintF
	case float32, float64:
		return toFloat64(value), floatF
	case time.Time:
		return value, timeF
	case Equality:
		return value, equalityF
	}
	switch cmp {
	case equalC, notEqualC:
		panic(fmt.Sprintf("Type[%T] not implements Equality.You can try changing the method receiver to a non-pointer object", value))
	case greaterC, greaterAndEqualC, lessC, lessAndEqualC:
		panic(fmt.Sprintf("Type[%T] not implements Comparable.You can try changing the method receiver to a non-pointer object", value))
	default:
		panic(fmt.Sprintf("Type[%T] not support", value))
	}
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

func (meta *StepMeta) addDepend(depends ...any) {
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
