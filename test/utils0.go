package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"sync/atomic"
	"testing"
	"time"
)

const (
	normalRet int64 = iota
	panicRet
	errorRet
)

var (
	times int64
)

type EqualityImpl struct {
	age  int
	name string
}

type PanicComparableImpl struct {
	Content string
}

func (p *PanicComparableImpl) Equal(_ any) bool {
	println("here0")
	if atomic.LoadInt64(&letGo) == 0 {
		return true
	}
	atomic.AddInt64(&current, 10)
	panic("implement me")
}

func (p *PanicComparableImpl) Less(_ any) bool {
	println("here1")
	if atomic.LoadInt64(&letGo) == 0 {
		return true
	}
	atomic.AddInt64(&current, 10)
	panic("implement me")
}

type ComparableImpl struct {
	age int
}

func (ei *EqualityImpl) Equal(other any) bool {
	e, ok := other.(*EqualityImpl)
	if !ok {
		return false
	}
	return ei.age == e.age && ei.name == e.name
}

func (ci *ComparableImpl) Equal(other any) bool {
	c, ok := other.(*ComparableImpl)
	if !ok {
		return false
	}
	return ci.age == c.age
}

func (ci *ComparableImpl) Less(other any) bool {
	c, ok := other.(*ComparableImpl)
	if !ok {
		return false
	}
	return ci.age < c.age
}

func resetTimes() {
	atomic.StoreInt64(&times, 0)
}

func Times(i int64) func() bool {
	return func() bool {
		return atomic.LoadInt64(&times) == i
	}
}

type Proto interface {
	//Get(key string) (value any, exist bool)
	//Set(key string, value any)
	ExplainStatus() []string
	Has(enum ...*flow.StatusEnum) bool
	Success() bool
	Name() string
	ID() string
	StartTime() *time.Time
	EndTime() *time.Time
	CostTime() time.Duration
}

type flowWrapper struct {
	flow.WorkFlow
}

func (fw flowWrapper) Get(_ string) (value any, exist bool) {
	panic("not implemented")
}

func (fw flowWrapper) Set(_ string, _ any) {
	panic("not implemented")
}

type FlexibleBuilder[T Proto] struct {
	*testing.T
	err   error
	doing []interface{}
	fxRet *int64
	index int
}

type ResourceI interface {
	Attach(resName string, initParam any) (flow.Resource, error)
	Acquire(resName string) (flow.Resource, bool)
}

func Fx[T Proto](t *testing.T) *FlexibleBuilder[T] {
	i := int64(0)
	return &FlexibleBuilder[T]{T: t, fxRet: &i}
}

func (fx *FlexibleBuilder[T]) CheckDepend() *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, func(s flow.Step) (any, error) {
		for _, dep := range s.Dependents() {
			fn := CheckCtx(dep)
			if _, err := fn(s); err != nil {
				panic(err)
			}
			atomic.AddInt64(&current, 1)
			return true, nil
		}
		return true, nil
	})
	return fx
}

func (fx *FlexibleBuilder[T]) SetCtx() *FlexibleBuilder[T] {
	f := SetCtx0("")
	fx.doing = append(fx.doing, func(ctx Ctx) (any, error) {
		_, err := f(ctx)
		if err != nil {
			panic(err)
		}
		atomic.AddInt64(&current, 1)
		return nil, nil
	})
	return fx
}

func (fx *FlexibleBuilder[T]) CheckCtx(preview string) *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, func(s flow.Step) (any, error) {
		fn := CheckCtx(preview)
		if _, err := fn(s); err != nil {
			panic(err)
		}
		return true, nil
	})
	return fx
}

func (fx *FlexibleBuilder[T]) CheckCtx0(preview, prefix string) *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, func(s flow.Step) (any, error) {
		fn := CheckCtx0(preview, prefix)
		if _, err := fn(s); err != nil {
			panic(err)
		}
		return true, nil
	})
	return fx
}

func (fx *FlexibleBuilder[T]) Inc() *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, func(s T) (any, error) {
		atomic.AddInt64(&current, 1)
		return nil, nil
	})
	return fx
}

func (fx *FlexibleBuilder[T]) Attach(resName string, initParam any, kv map[string]any) *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, func(s T) (any, error) {
		fx.Logf("attach resource %s", resName)
		res := any(s).(ResourceI)
		resource, err := res.Attach(resName, initParam)
		if err != nil {
			return s.Name(), err
		}
		for k, v := range kv {
			resource.Put(k, v)
		}
		atomic.AddInt64(&current, 1)
		return s.Name(), nil
	})
	return fx
}

func (fx *FlexibleBuilder[T]) Acquire(resName string, check any, kv map[string]any) *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, func(s T) (any, error) {
		res := any(s).(ResourceI)
		resource, exist := res.Acquire(resName)
		if !exist {
			panic(fmt.Sprintf("Resource [%s] not exist", resName))
		}
		if resource.Entity() != check {
			fx.Errorf("Check resource %s failed, expect %v, actual %v", resName, check, resource.Entity())
		}
		if resource.Entity() != nil {
			fx.Logf("Resource[ %s ] check entity success", resName)
		}
		for k, v := range kv {
			if target, find := resource.Fetch(k); !find {
				fx.Errorf("Check resource %s failed, key %s not exist", resName, k)
			} else if target != v {
				fx.Errorf("Check resource %s failed, key %s expect %v, actual %v", resName, k, v, target)
			}
		}
		atomic.AddInt64(&current, 1)
		return s.Name(), nil
	})
	return fx
}

func (fx *FlexibleBuilder[T]) Wait() *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, func(s T) (any, error) {
		ch := make(chan bool)
		go func() {
			for atomic.LoadInt64(&letGo) == 0 {
			}
			ch <- true
		}()
		select {
		case <-ch:
			return true, nil
		case <-time.After(1 * time.Second):
			fx.Errorf("Unit[%s] timeout", s.Name())
			return false, fmt.Errorf("timeout")
		}
	})
	return fx
}

func (fx *FlexibleBuilder[T]) Broadcast() *FlexibleBuilder[T] {
	last := fx.doing[len(fx.doing)-1].(func(s T) (any, error))
	fx.doing[len(fx.doing)-1] = func(s T) (any, error) {
		defer func() {
			atomic.StoreInt64(&letGo, 1)
		}()
		return last(s)
	}
	return fx
}

func (fx *FlexibleBuilder[T]) Error() *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, func(s T) (any, error) {
		atomic.AddInt64(&current, 1)
		return nil, fmt.Errorf("error")
	})
	return fx
}

func (fx *FlexibleBuilder[T]) Panic() *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, func(s T) (any, error) {
		atomic.AddInt64(&current, 1)
		panic("panic")
	})
	return fx
}

func (fx *FlexibleBuilder[T]) Add(f ...interface{}) *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, f...)
	return fx
}

func (fx *FlexibleBuilder[T]) Cond(suffixes ...string) *FlexibleBuilder[T] {
	suffix := ""
	if len(suffixes) != 0 {
		suffix = suffixes[0]
	}
	fx.doing = append(fx.doing, func(s flow.Step) (any, error) {
		fx.Logf("Step[ %s ] set condition", s.Name())
		s.Set(suffix+"int", int(1))
		s.Set(suffix+"int8", int8(2))
		s.Set(suffix+"int16", int16(3))
		s.Set(suffix+"int32", int32(4))
		s.Set(suffix+"int64", int64(5))
		s.Set(suffix+"uint", uint(6))
		s.Set(suffix+"uint8", uint8(7))
		s.Set(suffix+"uint16", uint16(8))
		s.Set(suffix+"uint32", uint32(9))
		s.Set(suffix+"uint64", uint64(10))
		s.Set(suffix+"float32", float32(11.1))
		s.Set(suffix+"float64", float64(12.2))
		s.Set(suffix+"bool", true)
		s.Set(suffix+"string", "hello")
		t, err := time.Parse(time.RFC3339, "2024-09-04T15:04:05Z")
		if err != nil {
			panic(err.Error())
		}
		s.Set(suffix+"time", t)
		s.Set(suffix+"*time", &t)
		s.Set(suffix+"Equality", EqualityImpl{age: 13, name: "Alice"})
		s.Set(suffix+"*Equality", &EqualityImpl{age: 14, name: "Bob"})
		s.Set(suffix+"*Comparable", &ComparableImpl{age: 16})
		s.Set(suffix+"PanicComparable", &PanicComparableImpl{Content: "panic"})
		atomic.AddInt64(&current, 1)
		fx.Logf("Step[ %s ] set condition succeed", s.Name())
		return s.Name(), nil
	})
	return fx
}

func (fx *FlexibleBuilder[T]) Condition(i int, ret int64) *FlexibleBuilder[T] {
	atomic.StoreInt64(fx.fxRet, ret)
	fx.index = i
	return fx
}

func (fx *FlexibleBuilder[T]) Step() func(flow.Step) (any, error) {
	return func(s flow.Step) (_ any, errs error) {
		defer println()
		fx.Logf("Step[ %s ] start running", s.Name())
		for i, f := range fx.doing {
			if i == fx.index {
				switch atomic.LoadInt64(fx.fxRet) {
				case errorRet:
					return nil, fmt.Errorf("error")
				case panicRet:
					panic("panic")
				}
			}
			switch fn := f.(type) {
			case func(flow.Step) (any, error):
				_, err := fn(s)
				if err != nil {
					fx.Logf("Step[ %s ] failed: %s", s.Name(), err)
					return s.Name(), err
				}
			case func(flow.Step) (bool, error):
				_, err := fn(s)
				if err != nil {
					fx.Logf("Step[ %s ] failed: %s", s.Name(), err)
					return s.Name(), err
				}
			case func(proto Proto) (any, error):
				_, err := fn(s)
				if err != nil {
					fx.Logf("Step[ %s ] failed: %s", s.Name(), err)
					return s.Name(), err
				}
			case func(ctx Ctx) (any, error):
				_, err := fn(s)
				if err != nil {
					fx.Logf("Step[ %s ] failed: %s", s.Name(), err)
					return s.Name(), err
				}
			default:
				panic(fmt.Sprintf("unsupported function type: %T", fn))
			}
		}
		switch atomic.LoadInt64(fx.fxRet) {
		case errorRet:
			return nil, fmt.Errorf("error")
		case panicRet:
			panic("panic")
		}
		fx.Logf("Step[ %s ] succeed", s.Name())
		return s.Name(), nil
	}
}

func (fx *FlexibleBuilder[T]) Callback() func(T) (bool, error) {
	return func(arg T) (_ bool, errs error) {
		defer println()
		flag := 0
		switch v := any(arg).(type) {
		case flow.Step:
			flag = 1
			fx.Logf("Step[ %s ] invoke callback", v.Name())
		case flow.Process:
			flag = 2
			fx.Logf("Process[ %s ] invoke callback", v.Name())
		case flow.WorkFlow:
			flag = 3
			fx.Logf("WorkFlow[ %s ] invoke callback", v.Name())
		default:
			panic(fmt.Sprintf("unsupported type: %T", v))
		}
		var err error
		for i, f := range fx.doing {
			if i == fx.index {
				switch atomic.LoadInt64(fx.fxRet) {
				case errorRet:
					return true, fmt.Errorf("error")
				case panicRet:
					panic("panic")
				}
			}
			switch fn := f.(type) {
			case func(Proto) (bool, error):
				_, err = fn(arg)
			case func(flow.Step) (bool, error):
				_, err = fn(any(arg).(flow.Step))
			case func(flow.Step) (any, error):
				_, err = fn(any(arg).(flow.Step))
			case func(flow.Process) (bool, error):
				_, err = fn(any(arg).(flow.Process))
			case func(flow.Process) (any, error):
				_, err = fn(any(arg).(flow.Process))
			case func(flow.WorkFlow) (bool, error):
				_, err = fn(any(arg).(flow.WorkFlow))
			case func(flow.WorkFlow) (any, error):
				_, err = fn(any(arg).(flow.WorkFlow))
			case func(Ctx) (bool, error):
				_, err = fn(any(arg).(Ctx))
			default:
				panic(fmt.Sprintf("unsupported function type: %T", fn))
			}
			if err != nil {
				switch flag {
				case 1:
					fx.Logf("Step[ %s ] failed: %s", arg.Name(), err)
				case 2:
					fx.Logf("Process[ %s ] failed: %s", arg.Name(), err)
				case 3:
					fx.Logf("WorkFlow[ %s ] failed: %s", arg.Name(), err)
				}
				return true, err
			}
		}
		switch atomic.LoadInt64(fx.fxRet) {
		case errorRet:
			return true, fmt.Errorf("error")
		case panicRet:
			panic("panic")
		}
		switch flag {
		case 1:
			fx.Logf("Step[ %s ] callback succeed", arg.Name())
		case 2:
			fx.Logf("Process[ %s ] callback succeed", arg.Name())
		case 3:
			fx.Logf("WorkFlow[ %s ] callback succeed", arg.Name())
		}
		return true, nil
	}
}
