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
	StartTime() time.Time
	EndTime() time.Time
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

func (fx *FlexibleBuilder[T]) Normal() *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, func(s T) (any, error) {
		atomic.AddInt64(&current, 1)
		return nil, nil
	})
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

func (fx *FlexibleBuilder[T]) SetCond() *FlexibleBuilder[T] {
	fx.doing = append(fx.doing, func(s flow.Step) (any, error) {
		fx.Logf("step[%s] set condition", s.Name())
		atomic.AddInt64(&current, 1)
		s.SetCondition(true)
		s.SetCondition(1)
		s.SetCondition(uint(2))
		s.SetCondition(float32(3))
		s.SetCondition("condition")
		return true, nil
	})
	return fx
}

func MatchCond(e *flow.StepMeta, depend string) {
	e.EQ(depend, 1, uint(2), float32(3), "condition")
}

func NotMatchCond(e *flow.StepMeta, depend string) {
	e.NEQ(depend, 1, uint(2), float32(3), "condition")
}

func (fx *FlexibleBuilder[T]) Condition(i int, ret int64) *FlexibleBuilder[T] {
	atomic.StoreInt64(fx.fxRet, ret)
	fx.index = i
	return fx
}

func (fx *FlexibleBuilder[T]) Step() func(flow.Step) (any, error) {
	return func(s flow.Step) (_ any, errs error) {
		defer println()
		fx.Logf("step[%s] start running", s.Name())
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
					fx.Logf("step[%s] failed: %s", s.Name(), err)
					return s.Name(), err
				}
			case func(flow.Step) (bool, error):
				_, err := fn(s)
				if err != nil {
					fx.Logf("step[%s] failed: %s", s.Name(), err)
					return s.Name(), err
				}
			case func(proto Proto) (any, error):
				_, err := fn(s)
				if err != nil {
					fx.Logf("step[%s] failed: %s", s.Name(), err)
					return s.Name(), err
				}
			case func(ctx Ctx) (any, error):
				_, err := fn(s)
				if err != nil {
					fx.Logf("step[%s] failed: %s", s.Name(), err)
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
		fx.Logf("step[%s] succeed", s.Name())
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
			fx.Logf("step[%s] invoke callback", v.Name())
		case flow.Process:
			flag = 2
			fx.Logf("process[%s] invoke callback", v.Name())
		case flow.WorkFlow:
			flag = 3
			fx.Logf("workflow[%s] invoke callback", v.Name())
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
					fx.Logf("step[%s] failed: %s", arg.Name(), err)
				case 2:
					fx.Logf("process[%s] failed: %s", arg.Name(), err)
				case 3:
					fx.Logf("workflow[%s] failed: %s", arg.Name(), err)
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
			fx.Logf("step[%s] callback succeed", arg.Name())
		case 2:
			fx.Logf("process[%s] callback succeed", arg.Name())
		case 3:
			fx.Logf("workflow[%s] callback succeed", arg.Name())
		}
		return true, nil
	}
}
