package light_flow

import (
	"fmt"
	"math"
	"reflect"
	"time"
)

const (
	stepCD   = "=%s"
	accurate = 1e-9
)

// Equality func (ei *EqualityImpl) Equal(other any) bool is invalid
// func (ei *EqualityImpl) Equal(other any) bool is valid
type Equality interface {
	Equal(other any) bool
}

type Comparable interface {
	Equality
	Less(other any) bool
}

type ExecCondition interface {
	SkipWithDependents(excludes ...any) ExecCondition
	EQ(match string, expect any) ExecCondition
	NEQ(match string, expect any) ExecCondition
	GT(match string, expect any) ExecCondition
	GTE(match string, expect any) ExecCondition
	LT(match string, expect any) ExecCondition
	LTE(match string, expect any) ExecCondition
	Condition(func(step Step) bool) ExecCondition
	OR() ExecCondition
}

type condition struct {
	conditions         []*condition // condition from merged
	groups             [][]func(step Step) bool
	skipWithDependents bool
	exclude            []string
}

// write to avoid `expect == expect` unsatisfied code check
func equal(a, b any) bool {
	return a == b
}

func gte(expect any) func(raw any) bool {
	f1 := gt(expect)
	f2 := eq(expect)
	return func(raw any) bool {
		return f1(raw) || f2(raw)
	}
}

func lte(expect any) func(raw any) bool {
	f1 := lt(expect)
	f2 := eq(expect)
	return func(raw any) bool {
		return f1(raw) || f2(raw)
	}
}

func neq(match string, expect any) func(step Step) bool {
	f := eq(expect)
	return func(step Step) (ret bool) {
		value, exist := step.Get(match)
		if !exist {
			return true
		}
		defer func() {
			if r := recover(); r != nil {
				ret = false
			}
		}()
		return !f(value)
	}
}

func eq(expect any) func(raw any) bool {
	return func(raw any) (ret bool) {
		switch actual := raw.(type) {
		case Equality:
			return actual.Equal(expect)
		case time.Time:
			if eTime, ok := expect.(time.Time); ok {
				return actual.Equal(eTime)
			}
		case *time.Time:
			if eTime, ok := expect.(*time.Time); ok {
				return actual.Equal(*eTime)
			}
		case float32:
			if e64, ok := expect.(float64); ok {
				return math.Abs(e64-float64(actual)) < accurate
			}
		case float64:
			if e64, ok := expect.(float64); ok {
				return math.Abs(e64-actual) < accurate
			}
		case int, int8, int16, int32, int64:
			if eInt, ok := expect.(int64); ok {
				return reflect.ValueOf(actual).Int() == eInt
			}
		case uint, uint8, uint16, uint32, uint64:
			if eUint, ok := expect.(uint64); ok {
				return reflect.ValueOf(actual).Uint() == eUint
			}
		default:
			return raw == expect
		}
		return false
	}
}

func gt(expect any) func(raw any) bool {
	f := lt(expect)
	return func(raw any) (ret bool) {
		return !f(raw)
	}
}

func lt(expect any) func(raw any) bool {
	return func(raw any) (ret bool) {
		switch actual := raw.(type) {
		case Comparable:
			return actual.Less(expect)
		case time.Time:
			if eTime, ok := expect.(time.Time); ok {
				return actual.Before(eTime)
			}
		case *time.Time:
			if eTime, ok := expect.(*time.Time); ok {
				return actual.Before(*eTime)
			}
		case int, int8, int16, int32, int64:
			if eInt, ok := expect.(int64); ok {
				return reflect.ValueOf(actual).Int() < eInt
			}
		case uint, uint8, uint16, uint32, uint64:
			if eUint, ok := expect.(uint64); ok {
				return reflect.ValueOf(actual).Uint() < eUint
			}
		case float32, float64:
			if eFloat, ok := expect.(float64); ok {
				return reflect.ValueOf(actual).Float() < eFloat
			}
		default:
			ret = false
		}
		return
	}
}

func (c *condition) meetCondition(step Step) (meet bool) {
	meet = c.evaluate(step)
	if meet && len(c.conditions) == 0 {
		return
	}

	for _, cond := range c.conditions {
		if !meet {
			break
		}
		meet = cond.evaluate(step)
	}

	if meet {
		return
	}

	step.append(Skip)
	if !c.skipWithDependents {
		return
	}
	step.append(skipped)
	for _, exclude := range c.exclude {
		step.setInternal(fmt.Sprintf(stepCD, exclude), nil)
	}
	for _, cond := range c.conditions {
		for _, exclude := range cond.exclude {
			step.setInternal(fmt.Sprintf(stepCD, exclude), nil)
		}
	}
	return
}

func (c *condition) evaluate(step Step) (pass bool) {
	if len(c.groups) == 0 {
		return true
	}
	for _, group := range c.groups {
		pass = true
		for _, cond := range group {
			if !cond(step) {
				pass = false
				break
			}
		}
		if pass {
			return
		}
	}
	return
}

func guard(match string, f func(raw any) bool) func(step Step) bool {
	return func(step Step) (ret bool) {
		value, exist := step.Get(match)
		if !exist {
			return false
		}
		defer func() {
			if r := recover(); r != nil {
				ret = false
			}
		}()
		return f(value)
	}
}

func (c *condition) SkipWithDependents(excludes ...any) ExecCondition {
	c.skipWithDependents = true
	c.exclude = make([]string, 0, len(excludes))
	for _, exclude := range excludes {
		switch actual := exclude.(type) {
		case string:
			c.exclude = append(c.exclude, actual)
		case func(ctx Step) (any, error):
			c.exclude = append(c.exclude, getFuncName(actual))
		default:
			panic(fmt.Sprintf("condition failed: exclude type[%T] not supported", exclude))
		}
	}
	return c
}

func (c *condition) EQ(match string, expect any) ExecCondition {
	return c.addEvaluate(match, expect, "EQ")
}

func (c *condition) NEQ(match string, expect any) ExecCondition {
	return c.addEvaluate(match, expect, "NEQ")
}

func (c *condition) LT(match string, expect any) ExecCondition {
	return c.addEvaluate(match, expect, "LT")
}

func (c *condition) LTE(match string, expect any) ExecCondition {
	return c.addEvaluate(match, expect, "LTE")
}

func (c *condition) GT(match string, expect any) ExecCondition {
	return c.addEvaluate(match, expect, "GT")
}

func (c *condition) GTE(match string, expect any) ExecCondition {
	return c.addEvaluate(match, expect, "GTE")
}

func (c *condition) Condition(f func(step Step) bool) ExecCondition {
	if len(c.groups) == 0 {
		c.groups = make([][]func(step Step) bool, 1)
	}
	foo := func(step Step) (ret bool) {
		defer func() {
			if r := recover(); r != nil {
				ret = false
			}
		}()
		return f(step)
	}
	c.groups[len(c.groups)-1] = append(c.groups[len(c.groups)-1], foo)
	return c
}

func (c *condition) OR() ExecCondition {
	if len(c.groups) == 0 || len(c.groups[len(c.groups)-1]) == 0 {
		panic(fmt.Sprintf("condition failed: OR() must be used after at leas one condition is added"))
	}
	c.groups = append(c.groups, make([]func(step Step) bool, 0))
	return c
}

func (c *condition) addEvaluate(match string, expect any, operator string) ExecCondition {
	if len(c.groups) == 0 {
		c.groups = make([][]func(step Step) bool, 1)
	}
	// ensure that the equality operation does not result in a panic
	if equality, ok := expect.(Equality); !ok {
		equal(expect, expect)
	} else {
		equality.Equal(expect)
	}
	// convert to a unified type for easy comparison.
	switch expect.(type) {
	case float32, float64:
		expect = reflect.ValueOf(expect).Float()
	case int, int8, int16, int32, int64:
		expect = reflect.ValueOf(expect).Int()
	case uint, uint8, uint16, uint32, uint64:
		expect = reflect.ValueOf(expect).Uint()
	}
	switch operator {
	case "EQ":
		c.groups[len(c.groups)-1] = append(c.groups[len(c.groups)-1], guard(match, eq(expect)))
	case "NEQ":
		c.groups[len(c.groups)-1] = append(c.groups[len(c.groups)-1], neq(match, expect))
	case "LT":
		c.groups[len(c.groups)-1] = append(c.groups[len(c.groups)-1], guard(match, lt(expect)))
	case "LTE":
		c.groups[len(c.groups)-1] = append(c.groups[len(c.groups)-1], guard(match, lte(expect)))
	case "GT":
		c.groups[len(c.groups)-1] = append(c.groups[len(c.groups)-1], guard(match, gt(expect)))
	case "GTE":
		c.groups[len(c.groups)-1] = append(c.groups[len(c.groups)-1], guard(match, gte(expect)))
	default:
		panic(fmt.Sprintf("condition failed: Operator[%s] not supported", operator))
	}
	return c
}

func (c *condition) mergeCond(other *condition) {
	c.conditions = append(c.conditions, other)
	c.skipWithDependents = c.skipWithDependents || other.skipWithDependents
}
