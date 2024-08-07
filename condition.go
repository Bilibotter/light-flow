package light_flow

import (
	"fmt"
	"math"
	"time"
)

type typeFlag int64

type comparator int8

type flags int64

type sliceSet[T comparable] []T

const (
	boolF typeFlag = 1 << iota
	strF
	intF
	uintF
	floatF
	timeF
	equalityF
	comparableF
	noneFlag typeFlag = 0
)

var (
	accurate = 1e-9
)

const (
	noneC comparator = iota // Avoiding the effects of default values
	equalC
	notEqualC
	lessC
	lessAndEqualC
	greaterC
	greaterAndEqualC
)

type evaluate func(value1, value2 any) bool

var (
	alwaysTrue evaluate = func(value1, value2 any) bool { return true }
)

type evaluator struct {
	typeValues     map[typeFlag]any
	typeEvaluate   map[typeFlag]evaluate
	typeComparator map[typeFlag]comparator
	name           string // only exact match evaluator has name
	depend         string
	expect         flags
}

type Evaluator interface {
	Identify(name string) Evaluator
	EQ(value ...any) Evaluator
	NEQ(value ...any) Evaluator
	GT(value ...any) Evaluator
	GTE(value ...any) Evaluator
	LT(value ...any) Evaluator
	LTE(value ...any) Evaluator
}

// Equality func (ei *EqualityImpl) Equal(other any) bool is invalid
// func (ei *EqualityImpl) Equal(other any) bool is valid
type Equality interface {
	Equal(other any) bool
}

type Comparable interface {
	Equality
	Less(other any) bool
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
	case Comparable:
		return comparableF
	case Equality:
		return equalityF
	}
	return noneFlag
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

func (eg *evaluator) Identify(name string) Evaluator {
	eg.name = name
	return eg
}

func (eg *evaluator) EQ(value ...any) Evaluator {
	return eg.update(equalC, value...)
}

func (eg *evaluator) NEQ(value ...any) Evaluator {
	return eg.update(notEqualC, value...)
}

func (eg *evaluator) GT(value ...any) Evaluator {
	return eg.update(greaterC, value...)
}

func (eg *evaluator) GTE(value ...any) Evaluator {
	return eg.update(greaterAndEqualC, value...)
}

func (eg *evaluator) LT(value ...any) Evaluator {
	return eg.update(lessC, value...)
}

func (eg *evaluator) LTE(value ...any) Evaluator {
	return eg.update(lessAndEqualC, value...)
}

func (eg *evaluator) update(cmp comparator, values ...any) Evaluator {
	for _, value := range values {
		normalized, flag := normalizeValue(value, cmp)
		if _, exist := eg.typeValues[flag]; exist {
			panic(fmt.Sprintf("Type[%T] duplicate", normalized))
		}
		eg.typeValues[flag] = normalized
		eg.typeEvaluate[flag] = eg.createEvaluate(flag, cmp)
		eg.typeComparator[flag] = cmp
		eg.expect.SetFlag(int64(flag))
	}
	return eg
}

func (eg *evaluator) createEvaluate(flag typeFlag, cmp comparator) evaluate {
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
	case comparableF:
		return eg.createComparableEvaluate(cmp)
	case equalityF:
		return eg.createEqualityEvaluate(cmp)
	}
	panic("Unexpected error occurred while createEvaluate")
}
func (eg *evaluator) createInt64Evaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(int64) == v2.(int64) }
	case notEqualC:
		return func(v1, v2 any) bool { return v1.(int64) != v2.(int64) }
	case greaterC:
		return func(v1, v2 any) bool { return v1.(int64) > v2.(int64) }
	case greaterAndEqualC:
		return func(v1, v2 any) bool { return v1.(int64) >= v2.(int64) }
	case lessC:
		return func(v1, v2 any) bool { return v1.(int64) < v2.(int64) }
	case lessAndEqualC:
		return func(v1, v2 any) bool { return v1.(int64) <= v2.(int64) }
	}
	return alwaysTrue
}

func (eg *evaluator) createUint64Evaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(uint64) == v2.(uint64) }
	case notEqualC:
		return func(v1, v2 any) bool { return v1.(uint64) != v2.(uint64) }
	case greaterC:
		return func(v1, v2 any) bool { return v1.(uint64) > v2.(uint64) }
	case greaterAndEqualC:
		return func(v1, v2 any) bool { return v1.(uint64) >= v2.(uint64) }
	case lessC:
		return func(v1, v2 any) bool { return v1.(uint64) < v2.(uint64) }
	case lessAndEqualC:
		return func(v1, v2 any) bool { return v1.(uint64) <= v2.(uint64) }
	}
	return alwaysTrue
}

func (eg *evaluator) createFloat64Evaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return math.Abs(v1.(float64)-v2.(float64)) <= accurate }
	case notEqualC:
		return func(v1, v2 any) bool { return math.Abs(v1.(float64)-v2.(float64)) > accurate }
	case greaterC:
		return func(v1, v2 any) bool { return v1.(float64) > v2.(float64) }
	case greaterAndEqualC:
		return func(v1, v2 any) bool { return v1.(float64) >= v2.(float64) }
	case lessC:
		return func(v1, v2 any) bool { return v1.(float64) < v2.(float64) }
	case lessAndEqualC:
		return func(v1, v2 any) bool { return v1.(float64) <= v2.(float64) }
	}
	return alwaysTrue
}

func (eg *evaluator) createBoolEvaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(bool) == v2.(bool) }
	case notEqualC:
		return func(v1, v2 any) bool { return v1.(bool) != v2.(bool) }
	case greaterC:
		panic("boolean types do not support greater-than comparisons")
	case greaterAndEqualC:
		panic("boolean types do not support greater-than-and-equal comparisons")
	case lessC:
		panic("boolean types do not support less-than comparisons")
	case lessAndEqualC:
		panic("boolean types do not support less-than-and-equal comparisons")
	}
	return alwaysTrue
}

func (eg *evaluator) createTimeEvaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(time.Time).Equal(v2.(time.Time)) }
	case notEqualC:
		return func(v1, v2 any) bool { return !v1.(time.Time).Equal(v2.(time.Time)) }
	case greaterC:
		return func(v1, v2 any) bool { return v1.(time.Time).After(v2.(time.Time)) }
	case greaterAndEqualC:
		return func(v1, v2 any) bool {
			return v1.(time.Time).After(v2.(time.Time)) || v1.(time.Time).Equal(v2.(time.Time))
		}
	case lessC:
		return func(v1, v2 any) bool { return v1.(time.Time).Before(v2.(time.Time)) }
	case lessAndEqualC:
		return func(v1, v2 any) bool {
			return v1.(time.Time).Before(v2.(time.Time)) || v1.(time.Time).Equal(v2.(time.Time))
		}
	}
	return alwaysTrue
}

func (eg *evaluator) createStringEvaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(string) == v2.(string) }
	case notEqualC:
		return func(v1, v2 any) bool { return v1.(string) != v2.(string) }
	case greaterC:
		panic("string types do not support greater-than comparisons")
	case greaterAndEqualC:
		panic("string types do not support greater-than-and-equal comparisons")
	case lessC:
		panic("string types do not support less-than comparisons")
	case lessAndEqualC:
		panic("string types do not support less-than-and-equal comparisons")
	}
	return alwaysTrue
}

func (eg *evaluator) createComparableEvaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(Equality).Equal(v2) }
	case notEqualC:
		return func(v1, v2 any) bool { return !v1.(Equality).Equal(v2) }
	case greaterC:
		return func(v1, v2 any) bool { return !v1.(Comparable).Less(v2) && !v1.(Comparable).Equal(v2) }
	case greaterAndEqualC:
		return func(v1, v2 any) bool { return !v1.(Comparable).Less(v2) }
	case lessC:
		return func(v1, v2 any) bool { return v1.(Comparable).Less(v2) }
	case lessAndEqualC:
		return func(v1, v2 any) bool { return v1.(Comparable).Less(v2) || v1.(Comparable).Equal(v2) }
	}
	return alwaysTrue
}

func (eg *evaluator) createEqualityEvaluate(cmp comparator) evaluate {
	switch cmp {
	case equalC:
		return func(v1, v2 any) bool { return v1.(Equality).Equal(v2) }
	case notEqualC:
		return func(v1, v2 any) bool { return !v1.(Equality).Equal(v2) }
	case greaterC:
		panic("Type[Equality] not implements Comparable.")
	case lessC:
		panic("Type[Equality] not implements Comparable.")
	}
	return alwaysTrue
}

func (eg *evaluator) evaluate(named, unnamed []evalValue) bool {
	flag := flags(0)
	if !eg.evaluateValues(&flag, named) {
		return false
	}
	if !eg.evaluateValues(&flag, unnamed) {
		return false
	}
	return eg.meetExpect(flag)
}

func (eg *evaluator) meetExpect(flag flags) bool {
	mask := flags(1)
	for unexpect := eg.expect ^ flag; unexpect >= mask; mask <<= 1 {
		// notEqualC should be true when the condition is missing
		if mask&unexpect != 0 && eg.typeComparator[typeFlag(mask)] != notEqualC {
			return false
		}
	}
	return true
}

func (eg *evaluator) evaluateValues(flag *flags, values []evalValue) bool {
	for _, v := range values {
		if v.Matches != nil && !v.Matches.Contains(eg.name) {
			continue
		}
		if !eg.evaluateByType(flag, getTypeFlag(v.Value), v.Value) {
			return false
		}
	}
	// this is intermediate result, only if eg.expect == flags is fully evaluated
	return true
}

func (eg *evaluator) evaluateByType(current *flags, flag typeFlag, value any) bool {
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

func (ss sliceSet[T]) Find(element any) bool {
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

func (meta *StepMeta) EQ(depend interface{}, value ...any) Evaluator {
	meta.addDepend(depend)
	return meta.addEvalGroup(toStepName(depend), equalC, value...)
}

func (meta *StepMeta) NEQ(depend interface{}, value ...any) Evaluator {
	meta.addDepend(depend)
	return meta.addEvalGroup(toStepName(depend), notEqualC, value...)
}

func (meta *StepMeta) GT(depend interface{}, value ...any) Evaluator {
	meta.addDepend(depend)
	return meta.addEvalGroup(toStepName(depend), greaterC, value...)
}

func (meta *StepMeta) GTE(depend interface{}, value ...any) Evaluator {
	meta.addDepend(depend)
	return meta.addEvalGroup(toStepName(depend), greaterAndEqualC, value...)
}

func (meta *StepMeta) LT(depend interface{}, value ...any) Evaluator {
	meta.addDepend(depend)
	return meta.addEvalGroup(toStepName(depend), lessC, value...)
}

func (meta *StepMeta) LTE(depend interface{}, value ...any) Evaluator {
	meta.addDepend(depend)
	return meta.addEvalGroup(toStepName(depend), lessAndEqualC, value...)
}

func (meta *StepMeta) addEvalGroup(depend string, cmp comparator, values ...any) *evaluator {
	group := &evaluator{
		name:           meta.name,
		depend:         depend,
		typeValues:     make(map[typeFlag]any, 1),
		typeEvaluate:   make(map[typeFlag]evaluate, 1),
		typeComparator: make(map[typeFlag]comparator, 1),
	}
	meta.evaluators = append(meta.evaluators, group)
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
	case Comparable:
		return value, comparableF
	case Equality:
		return value, equalityF
	}
	switch cmp {
	case equalC, notEqualC:
		panic(fmt.Sprintf("Type[%T] not implements Equality.You can trying to use a pointer object as argument", value))
	case greaterC, greaterAndEqualC, lessC, lessAndEqualC:
		panic(fmt.Sprintf("Type[%T] not implements Comparable.You can trying to use a pointer object as a argument", value))
	default:
		panic(fmt.Sprintf("Type[%T] not support", value))
	}
}

func (meta *StepMeta) addDepend(depends ...any) {
	for _, wrap := range depends {
		dependName := toStepName(wrap)
		if dependName == meta.name {
			panic(fmt.Sprintf("Step[%s] can't depend on itself.", meta.name))
		}
		if meta.existDepend(dependName) {
			continue
		}
		depend, exist := meta.belong.steps[dependName]
		if !exist {
			panic(fmt.Sprintf("Step[%s]'s depend[%s] not found.", meta.name, dependName))
		}
		meta.depends = append(meta.depends, depend)
	}
	meta.wireDepends()
}

func (meta *StepMeta) existDepend(name string) bool {
	for _, depend := range meta.depends {
		if depend.name == name {
			return true
		}
	}
	return false
}
