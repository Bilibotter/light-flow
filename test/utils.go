package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var (
	current    int64
	letGo      = false
	executeSuc = false
)

var (
	shallCallbackKey = "~"
)

type Named interface {
	name() string
}

type Ctx interface {
	Get(key string) (value any, exist bool)
	Set(key string, value any)
	Name() string
}

type FuncBuilder struct {
	t      *testing.T
	err    error
	doing  []func(ctx Ctx) (any, error)
	onSuc  []func(ctx Ctx) (any, error)
	onFail []func(ctx Ctx) (any, error)
}

type Person struct {
	Name string
	Age  int
}

func (p *Person) name() string {
	return p.Name
}

func (s *FuncBuilder) Step() func(ctx flow.Step) (result any, err error) {
	return func(ctx flow.Step) (result any, err error) {
		return s.Fn(ctx)
	}
}

func (s *FuncBuilder) ErrStep() func(ctx flow.Step) (result any, err error) {
	s.err = fmt.Errorf("error")
	return func(ctx flow.Step) (result any, err error) {
		return s.Fn(ctx)
	}
}

func (s *FuncBuilder) Proc() func(ctx flow.Process) (keepOn bool, err error) {
	return func(ctx flow.Process) (keepOn bool, err error) {
		_, err = s.Fn(ctx)
		return true, err
	}
}

func (s *FuncBuilder) ErrProc() func(ctx flow.Process) (keepOn bool, err error) {
	s.err = fmt.Errorf("error")
	return func(ctx flow.Process) (keepOn bool, err error) {
		_, err = s.Fn(ctx)
		return true, err
	}
}

func (s *FuncBuilder) Fn(ctx Ctx) (result any, err error) {
	s.t.Logf("task[%s] start\n", ctx.Name())
	for _, f := range s.doing {
		result, err = f(ctx)
		if err != nil {
			panic(err)
		}
	}
	if len(s.doing) > 0 {
		atomic.AddInt64(&current, 1)
	}
	if executeSuc {
		for _, f := range s.onSuc {
			result, err = f(ctx)
			if err != nil {
				return nil, err
			}
		}
		if result == nil {
			result = ctx.Name()
		}
		if len(s.onSuc) > 0 {
			atomic.AddInt64(&current, 1)
		}
		s.t.Logf("task[%s] end\n", ctx.Name())
		return
	}
	for _, f := range s.onFail {
		result, err = f(ctx)
		if err != nil {
			panic(err)
		}
	}
	if len(s.onFail) > 0 {
		atomic.AddInt64(&current, 1)
	}
	if s.err != nil {
		s.t.Logf("task[%s] failed\n", ctx.Name())
		return nil, s.err
	}
	s.t.Logf("task[%s] end\n", ctx.Name())
	return ctx.Name(), nil
}

func (s *FuncBuilder) Do(f ...func(Ctx) (any, error)) *FuncBuilder {
	s.doing = append(s.doing, f...)
	return s
}

func (s *FuncBuilder) Suc(f ...func(Ctx) (any, error)) *FuncBuilder {
	s.onSuc = append(s.onSuc, f...)
	return s
}

func (s *FuncBuilder) Fail(f ...func(Ctx) (any, error)) *FuncBuilder {
	s.onFail = append(s.onFail, f...)
	return s
}

func (s *FuncBuilder) Normal() func(ctx flow.Step) (result any, err error) {
	return func(ctx flow.Step) (result any, err error) {
		atomic.AddInt64(&current, 1)
		s.t.Logf("Step[%s] start\n", ctx.Name())
		s.t.Logf("Step[%s] end\n", ctx.Name())
		return ctx.Name(), nil
	}
}

func (s *FuncBuilder) All(fs ...func(ctx flow.Step) (result any, err error)) func(ctx flow.Step) (result any, err error) {
	return func(ctx flow.Step) (result any, err error) {
		for _, f := range fs {
			result, err = f(ctx)
		}
		return
	}
}

func (s *FuncBuilder) WaitLetGO(i ...int64) func(ctx flow.Step) (result any, err error) {
	return func(ctx flow.Step) (result any, err error) {
		atomic.AddInt64(&current, 1)
		s.t.Logf("Step[%s] start\n", ctx.Name())
		start := time.Now()
		duration := 100 * time.Millisecond
		for {
			if letGo {
				s.t.Logf("Step[%s] finish wait\n", ctx.Name())
				if len(i) > 0 {
					atomic.AddInt64(&current, i[0])
				}
				return ctx.Name(), nil
			}
			if time.Since(start) > duration {
				s.t.Errorf("Step[%s] timeout\n", ctx.Name())
				return ctx.Name(), nil
			}
			runtime.Gosched()
		}
	}
}

func (s *FuncBuilder) Panic(i ...int64) func(ctx flow.Step) (result any, err error) {
	return func(ctx flow.Step) (result any, err error) {
		if len(i) > 0 {
			atomic.AddInt64(&current, i[0])
		} else {
			atomic.AddInt64(&current, 1)
		}
		s.t.Logf("Step[%s] panic\n", ctx.Name())
		panic(fmt.Sprintf("Step[%s] panic", ctx.Name()))
	}
}

func (s *FuncBuilder) Errors(i ...int64) func(ctx flow.Step) (result any, err error) {
	return func(ctx flow.Step) (result any, err error) {
		if len(i) > 0 {
			atomic.AddInt64(&current, i[0])
		} else {
			atomic.AddInt64(&current, 1)
		}
		s.t.Logf("Step[%s] error\n", ctx.Name())
		return ctx.Name(), fmt.Errorf("step[%s] error", ctx.Name())
	}
}

func (s *FuncBuilder) SetCtx() func(ctx flow.Step) (result any, err error) {
	f := Fn(s.t).Do(SetCtx()).Step()
	return f
}

func (s *FuncBuilder) SetCtx0(prefix string) *FuncBuilder {
	f := Fn(s.t)
	f.Do(SetCtx0(prefix)).Step()
	return f
}

func (s *FuncBuilder) CheckCtx(preview string, notCheckResult ...int) func(ctx flow.Step) (result any, err error) {
	f := Fn(s.t)
	fn := f.Do(CheckCtx(preview, notCheckResult...)).Step()
	return fn
}

func (s *FuncBuilder) CheckCtx0(preview string, prefix string, notCheckResult ...int) *FuncBuilder {
	f := Fn(s.t)
	f.Do(CheckCtx0(preview, prefix, notCheckResult...))
	return f
}

func resetCurrent() {
	atomic.StoreInt64(&current, 0)
}

func waitCurrent(i int) {
	if atomic.LoadInt64(&current) == int64(i) {
		return
	}
	start := time.Now()
	duration := 200 * time.Millisecond
	for {
		if atomic.LoadInt64(&current) == int64(i) {
			return
		}
		runtime.Gosched()
		if atomic.LoadInt64(&current) == int64(i) {
			return
		}
		if time.Since(start) > duration {
			panic(fmt.Sprintf("current is not %d, current is %d\n", i, atomic.LoadInt64(&current)))
		}
	}
}

func Fn(t *testing.T) *FuncBuilder {
	return &FuncBuilder{t: t}
}

func SetCtx() func(ctx Ctx) (any, error) {
	return SetCtx0("")
}

func SetCtx0(prefix string) func(ctx Ctx) (any, error) {
	return func(ctx Ctx) (any, error) {
		ctx.Set(prefix+"int", 1)
		ctx.Set(prefix+"int8", int8(2))
		ctx.Set(prefix+"int16", int16(3))
		ctx.Set(prefix+"int32", int32(4))
		ctx.Set(prefix+"int64", int64(5))
		ctx.Set(prefix+"uint", uint(6))
		ctx.Set(prefix+"uint8", uint8(7))
		ctx.Set(prefix+"uint16", uint16(8))
		ctx.Set(prefix+"uint32", uint32(9))
		ctx.Set(prefix+"uint64", uint64(10))
		ctx.Set(prefix+"float32", float32(11.01))
		ctx.Set(prefix+"float64", float64(12.01))
		ctx.Set(prefix+"bool", true)
		ctx.Set(prefix+"string", ctx.Name())
		ctx.Set(prefix+"password", ctx.Name())
		ctx.Set(prefix+"pwd", ctx.Name())
		ctx.Set(prefix+"Person", Person{ctx.Name(), 11})
		ctx.Set(prefix+"*Person", &Person{ctx.Name(), 11})
		ctx.Set(shallCallbackKey, 11)
		var named Named
		named = &Person{ctx.Name(), 11}
		ctx.Set(prefix+"Named", named)
		return ctx.Name(), nil
	}
}

func CheckCtx(preview string, notCheckResult ...int) func(ctx Ctx) (any, error) {
	return CheckCtx0(preview, "", notCheckResult...)
}

func CheckCtx1(notCheckResult ...int) func(ctx Ctx) (any, error) {
	return func(ctx Ctx) (any, error) {
		if step, ok := ctx.(flow.Step); ok {
			for _, depend := range step.Dependents() {
				return CheckCtx0(depend, "", notCheckResult...)(ctx)
			}
		}
		return ctx.Name(), nil
	}
}

func CheckCtx0(preview, prefix string, notCheckResult ...int) func(ctx Ctx) (any, error) {
	return func(ctx Ctx) (any, error) {
		if step, ok := ctx.(flow.Step); ok && len(notCheckResult) == 0 {
			if res, exist := step.Result(preview); exist {
				if res.(string) != preview {
					panic(fmt.Sprintf("Step[%s] check Result[%s] failed, result=[%#v]", ctx.Name(), preview, res))
				}
			} else {
				panic(fmt.Sprintf("Step[%s] check Result[%s] failed, result not exist", ctx.Name(), preview))
			}
		}
		if value, exist := ctx.Get(prefix + "int"); exist {
			if value.(int) != 1 {
				return ctx.Name(), fmt.Errorf("Step[%s] check int failed, int=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check int failed, int=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "int8"); exist {
			if value.(int8) != 2 {
				return ctx.Name(), fmt.Errorf("Step[%s] check int8 failed, int8=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check int8 failed, int8=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "int16"); exist {
			if value.(int16) != 3 {
				return ctx.Name(), fmt.Errorf("Step[%s] check int16 failed, int16=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check int16 failed, int16=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "int32"); exist {
			if value.(int32) != 4 {
				return ctx.Name(), fmt.Errorf("Step[%s] check int32 failed, int32=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check int32 failed, int32=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "int64"); exist {
			if value.(int64) != 5 {
				return ctx.Name(), fmt.Errorf("Step[%s] check int64 failed, int64=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check int64 failed, int64=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "uint"); exist {
			if value.(uint) != 6 {
				return ctx.Name(), fmt.Errorf("Step[%s] check uint failed, uint=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check uint failed, uint=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "uint8"); exist {
			if value.(uint8) != 7 {
				return ctx.Name(), fmt.Errorf("Step[%s] check uint8 failed, uint8=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check uint8 failed, uint8=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "uint16"); exist {
			if value.(uint16) != 8 {
				return ctx.Name(), fmt.Errorf("Step[%s] check uint16 failed, uint16=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check uint16 failed, uint16=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "uint32"); exist {
			if value.(uint32) != 9 {
				return ctx.Name(), fmt.Errorf("Step[%s] check uint32 failed, uint32=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check uint32 failed, uint32=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "uint64"); exist {
			if value.(uint64) != 10 {
				return ctx.Name(), fmt.Errorf("Step[%s] check uint64 failed, uint64=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check uint64 failed, uint64=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "float32"); exist {
			if value.(float32) != 11.01 {
				return ctx.Name(), fmt.Errorf("Step[%s] check float32 failed, float32=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check float32 failed, float32=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "float64"); exist {
			if value.(float64) != 12.01 {
				return ctx.Name(), fmt.Errorf("Step[%s] check float64 failed, float64=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check float64 failed, float64=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "bool"); exist {
			if value.(bool) != true {
				return ctx.Name(), fmt.Errorf("Step[%s] check bool failed, bool=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check bool failed, bool=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "Person"); exist {
			if value.(Person).Name != preview || value.(Person).Age != 11 {
				return ctx.Name(), fmt.Errorf("Step[%s] check Person failed, Person=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check Person failed, Person=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "*Person"); exist {
			if value.(*Person).Name != preview || value.(*Person).Age != 11 {
				return ctx.Name(), fmt.Errorf("Step[%s] check *Person failed, *Person=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check *Person failed, *Person=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "Named"); exist {
			if value.(*Person).Name != preview || value.(*Person).Age != 11 {
				return ctx.Name(), fmt.Errorf("Step[%s] check Named failed, Named=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check Named failed, Named=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "string"); exist {
			if value.(string) != preview {
				return ctx.Name(), fmt.Errorf("Step[%s] check string failed, string=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check string failed, string=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "password"); exist {
			if value.(string) != preview {
				return ctx.Name(), fmt.Errorf("Step[%s] check password failed, password=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check password failed, password=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "pwd"); exist {
			if value.(string) != preview {
				return ctx.Name(), fmt.Errorf("Step[%s] check pwd failed, pwd=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check pwd failed, pwd=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(shallCallbackKey); exist {
			if value.(int) != 11 {
				return ctx.Name(), fmt.Errorf("Step[%s] check shallCallbackKey failed, shallCallbackKey=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check shallCallbackKey failed, shallCallbackKey=[%#v]", ctx.Name(), value)
		}
		return ctx.Name(), nil
	}
}

func CheckResult(t *testing.T, check int64, statuses ...*flow.StatusEnum) func(flow.WorkFlow) (keepOn bool, err error) {
	return func(workFlow flow.WorkFlow) (keepOn bool, err error) {
		ss := make([]string, len(statuses))
		for i, status := range statuses {
			ss[i] = status.Message()
		}
		t.Logf("start check, expected current=%d, status include %s", check, strings.Join(ss, ","))
		if current != check {
			t.Errorf("execute %d step, but current = %d\n", check, current)
		}
		for _, status := range statuses {
			if status == flow.Success && !workFlow.Success() {
				t.Errorf("WorkFlow executed failed\n")
			}
			if !workFlow.HasAny(status) {
				t.Errorf("workFlow has not %s status\n", status.Message())
			}
		}
		t.Logf("status expalin=%s", strings.Join(workFlow.ExplainStatus(), ","))
		t.Logf("finish check")
		println()
		return true, nil
	}
}
